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
	JOB_ID_KEY      = "JobID"
	WORKER_ID_KEY   = "WorkerID"
	MAX_WORKERS     = 3
	MAX_JOBS        = 60
	SHUTDOWN_SIGNAL = -1
	EXECUTE_TIME_AFTER   = 3001 * time.Millisecond
	CONTEXT_TIMEOUT = 3000 * time.Millisecond
)

func NoopExecute(ctx context.Context) (interface{}, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(EXECUTE_TIME_AFTER):
		workerID, workerOk := ctx.Value(WORKER_ID_KEY).(int)
		jobID, jobOk := ctx.Value(JOB_ID_KEY).(int)
		if !workerOk || !jobOk {
			return nil, fmt.Errorf("context values missing or wrong type: workerID=%v, jobID=%v, workerOk=%t, jobOk=%t", workerID, jobID, workerOk, jobOk)
		}
		return fmt.Sprintf("Worker: %d completed Job: %d successfully.", workerID, jobID), nil
	}
}

func NewJob(id int, execute func(context.Context) (interface{}, error)) Job {
	return Job{ID: id, Execute: execute}
}

type Worker struct {
	ID int
	wg *sync.WaitGroup
}

func (w *Worker) Start(ctx context.Context, jobs <-chan Job, results chan<- Result, errs chan<- error) {
	defer w.wg.Done()
	for job := range jobs {
		if job.ID == SHUTDOWN_SIGNAL {
			return
		}
		jobCtx, cancel := context.WithTimeout(ctx, CONTEXT_TIMEOUT)
		defer cancel()

		jobCtx = context.WithValue(jobCtx, WORKER_ID_KEY, w.ID)
        jobCtx = context.WithValue(jobCtx, JOB_ID_KEY, job.ID)

        result, err := job.Execute(jobCtx)
        if err != nil {
            errs <- fmt.Errorf("worker %d: job %d error: %s", w.ID, job.ID, err)
            continue
        }
		results <- Result{WorkerID: w.ID, JobID: job.ID, Data: result}
	}
}

func NewWorker(number int, wg *sync.WaitGroup) Worker {
	wg.Add(1)
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
	return &WorkerPool{
		jobs:       make(chan Job, MAX_JOBS),
		results:    make(chan Result, MAX_JOBS),
		errs:       make(chan error, MAX_JOBS),
		maxWorkers: maxWorkers,
		ctx:        ctx,
	}
}

func (wp *WorkerPool) Run() {
	for i := 0; i < wp.maxWorkers; i++ {
		worker := NewWorker(i+1, &wp.wg)
		go worker.Start(wp.ctx, wp.jobs, wp.results, wp.errs)
	}
}

func (wp *WorkerPool) SubmitJob(job Job) {
	wp.jobs <- job
}

func (wp *WorkerPool) Shutdown() {
	for i := 0; i < wp.maxWorkers; i++ {
		wp.jobs <- Job{ID: SHUTDOWN_SIGNAL}
	}
	wp.wg.Wait()
	close(wp.jobs)
	close(wp.results)
	close(wp.errs)
	log.Println("All jobs completed.")
}

func main() {
	log.SetOutput(os.Stdout)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	workerPool := NewWorkerPool(MAX_WORKERS, ctx)
	go workerPool.Run()

	for i := 1; i <= MAX_JOBS; i++ {
		job := NewJob(i, NoopExecute)
		workerPool.SubmitJob(job)
	}

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

	workerPool.wg.Wait()
	workerPool.Shutdown()
}
