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
    MAX_WORKERS = 3
    MAX_JOBS = 30
)

func NoopExecute(ctx context.Context) (interface{}, error) {
	time.Sleep(1 * time.Second)
	return fmt.Sprintf("Worker: %d completed Job: %d successfully.", ctx.Value(WORKER_ID_KEY), ctx.Value(JOB_ID_KEY)), nil
}

func NewJob(id int, execute func(context.Context) (interface{}, error)) Job {
	return Job{ID: id, Execute: execute}
}

type Worker struct {
	ID int
	wg *sync.WaitGroup
}

func (w *Worker) Start(ctx context.Context, jobs <-chan Job, results chan<- Result, errs chan<- error) {
	for job := range jobs {
		result, err := job.Execute(func(c context.Context) context.Context {
			cjob := context.WithValue(c, JOB_ID_KEY, job.ID)
			return context.WithValue(cjob, WORKER_ID_KEY, w.ID)
		}(ctx))
		w.wg.Done()
		if err != nil {
			errs <- fmt.Errorf("Worker %d: Job %d error: %s", w.ID, job.ID, err.Error())
		} else {
			results <- Result{WorkerID: w.ID, JobID: job.ID, Data: result}
		}
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
}

func NewWorkerPool(maxWorkers int) *WorkerPool {
	pool := &WorkerPool{
		jobs:       make(chan Job),
		results:    make(chan Result),
		errs:       make(chan error),
		maxWorkers: maxWorkers,
		wg:         sync.WaitGroup{},
	}
	return pool
}

func (wp *WorkerPool) Run(ctx context.Context) {
	for i := 0; i < wp.maxWorkers; i++ {
		worker := NewWorker(i+1, &wp.wg)
		go worker.Start(ctx, wp.jobs, wp.results, wp.errs)
	}
}

func (wp *WorkerPool) SubmitJob(job Job) {
	wp.wg.Add(1)
	wp.jobs <- job
}

func main() {
	log.SetOutput(os.Stdout)
	workerPool := NewWorkerPool(3)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer func() {
		cancel()
        if r := recover(); r != nil {
			log.Println("Recovered in main; deferred functions are running")
		}
		close(workerPool.jobs)
		close(workerPool.results)
		close(workerPool.errs)
	}()

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

    go workerPool.Run(ctx)

	for i := 1; i <= MAX_JOBS; i++ {
		j := i
		workerPool.SubmitJob(Job{
			ID: j,
			Execute: func(jobctx context.Context) (interface{}, error) {
				return NoopExecute(jobctx)
			},
		})
	}

	workerPool.wg.Wait()

	log.Println("All jobs have been completed successfully.")
}
