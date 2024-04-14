package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

type Result struct {
	WorkerID int
	JobID    int
	Data     interface{}
}

type Job struct {
	ID      int
	Execute func(context.Context) (interface{}, error)
}

const (
	JOB_ID_KEY    = "JobID"
	WORKER_ID_KEY = "WorkerID"
	MAX_WORKERS   = 3
	MAX_JOBS      = 30
)

func NoopExecute(ctx context.Context) (interface{}, error) {
	workerID := ctx.Value(WORKER_ID_KEY)
	jobID := ctx.Value(JOB_ID_KEY)
	time.Sleep(1 * time.Second) // simulate work
	if workerID == nil || jobID == nil {
		return nil, fmt.Errorf("context values missing")
	}
	return fmt.Sprintf("Worker: %d completed Job: %d successfully.", workerID, jobID), nil
}

func NewJob(id int, execute func(context.Context) (interface{}, error)) Job {
	return Job{ID: id, Execute: execute}
}

type Worker struct {
	ID int
	wg *sync.WaitGroup
}

// func (w *Worker) Start(ctx context.Context, jobs <-chan Job, results chan<- Result, errs chan<- error) {
// 	for job := range jobs {
// 		if job.ID == -1 { // shutdown signal
// 			return
// 		}

// 		jobCtx := context.WithValue(ctx, JOB_ID_KEY, job.ID)
// 		jobCtx = context.WithValue(jobCtx, WORKER_ID_KEY, w.ID)

// 		result, err := job.Execute(jobCtx)
// 		if err != nil {
// 			errs <- fmt.Errorf("Worker %d: Job %d error: %s", w.ID, job.ID, err)
// 		} else {
// 			results <- Result{WorkerID: w.ID, JobID: job.ID, Data: result}
// 		}
// 		w.wg.Done()
// 	}
// }

func (w *Worker) Start(ctx context.Context, jobs <-chan Job, results chan<- Result, errs chan<- error) {
    for job := range jobs {
        if job.ID == -1 { // shutdown signal
            return
        }

        jobCtx := context.WithValue(ctx, JOB_ID_KEY, job.ID)
        jobCtx = context.WithValue(jobCtx, WORKER_ID_KEY, w.ID)

        result, err := job.Execute(jobCtx)
        if err != nil {
            errs <- fmt.Errorf("Worker %d: Job %d error: %s", w.ID, job.ID, err)
            w.wg.Done()
            continue
        }
        results <- Result{WorkerID: w.ID, JobID: job.ID, Data: result}
        w.wg.Done()
    }
}

func NewWorker(number int, wg *sync.WaitGroup) Worker {
	return Worker{ID: number, wg: wg}
}

type WorkerPool struct {
	jobs       chan Job
	results    chan Result
	errs       chan error
	maxWorkers int
	wg         sync.WaitGroup
	ctx        context.Context
}

func NewWorkerPool(maxWorkers int, ctx context.Context) *WorkerPool {
	pool := &WorkerPool{
		jobs:       make(chan Job),
		results:    make(chan Result),
		errs:       make(chan error),
		maxWorkers: maxWorkers,
		wg:         sync.WaitGroup{},
		ctx:        ctx,
	}
	return pool
}

func (wp *WorkerPool) Run() {
	for i := 0; i < wp.maxWorkers; i++ {
		worker := NewWorker(i+1, &wp.wg)
		go worker.Start(wp.ctx, wp.jobs, wp.results, wp.errs)
	}
}

// func (wp *WorkerPool) SubmitJob(job Job, jobTimeout time.Duration) {
// 	ctx, cancel := context.WithTimeout(wp.ctx, jobTimeout)
// 	wp.wg.Add(1)
// 	wp.jobs <- Job{
// 		ID: job.ID,
// 		Execute: func(jobCtx context.Context) (interface{}, error) {
// 			defer cancel() // ensure resources are freed
// 			return job.Execute(ctx)
// 		},
// 	}
// }

func (wp *WorkerPool) SubmitJob(job Job, jobTimeout time.Duration) {
    _, cancel := context.WithTimeout(wp.ctx, jobTimeout)
    defer cancel()  // It's safe to defer cancel here as it only impacts this submission scope
    wp.wg.Add(1)
    wp.jobs <- Job{
        ID: job.ID,
        Execute: func(innerCtx context.Context) (interface{}, error) {
            // Propagate the context with job and worker ID keys
            return job.Execute(innerCtx)
        },
    }
}

func (wp *WorkerPool) Shutdown() {
    for i := 0; i < wp.maxWorkers; i++ {
        wp.jobs <- Job{ID: -1}  // send shutdown signal
    }
    wp.wg.Wait()  // wait for all workers to finish
    close(wp.jobs)
    close(wp.results)
    close(wp.errs)
}

func main() {
	log.SetOutput(os.Stdout)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	workerPool := NewWorkerPool(MAX_WORKERS, ctx)
	defer workerPool.Shutdown() // Ensure graceful shutdown

	go func() {
		for result := range workerPool.results {
			log.Printf("Result from Worker %d, Job %d: %v\n", result.WorkerID, result.JobID, result.Data)
		}
	}()

	go func() {
		for err := range workerPool.errs {
			log.Printf("Error: %v\n", err)
		}
	}()

	go workerPool.Run()

	for i := 1; i <= MAX_JOBS; i++ {
		workerPool.SubmitJob(NewJob(i, NoopExecute), 2*time.Second)
	}

	workerPool.wg.Wait()

	log.Println("All jobs have been completed successfully.")
}
