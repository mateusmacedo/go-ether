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
	MAX_WORKERS   = 6
	MAX_JOBS      = 30
	SHUTDOWN_SIGNAL = -1
)

func NoopExecute(ctx context.Context) (interface{}, error) {
	workerID, workerOk := ctx.Value(WORKER_ID_KEY).(int)
	jobID, jobOk := ctx.Value(JOB_ID_KEY).(int)
	time.Sleep(1 * time.Second) // Simulate work
	if !workerOk || !jobOk {
		return nil, fmt.Errorf("context values missing or wrong type: workerID=%v, jobID=%v, workerOk=%t, jobOk=%t", workerID, jobID, workerOk, jobOk)
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

func (w *Worker) Start(ctx context.Context, jobs <-chan Job, results chan<- Result, errs chan<- error) {
	for job := range jobs {
		if job.ID == SHUTDOWN_SIGNAL {
			return
		}

		jobCtx := context.WithValue(ctx, WORKER_ID_KEY, w.ID)
		jobCtx = context.WithValue(jobCtx, JOB_ID_KEY, job.ID)

		result, err := job.Execute(jobCtx)
		w.wg.Done()
		if err != nil {
			errs <- fmt.Errorf("worker %d: job %d error: %s", w.ID, job.ID, err)
			continue
		}
		results <- Result{WorkerID: w.ID, JobID: job.ID, Data: result}

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
		jobs:       make(chan Job, MAX_JOBS),
		results:    make(chan Result, MAX_JOBS),
		errs:       make(chan error, MAX_JOBS),
		maxWorkers: maxWorkers,
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

func (wp *WorkerPool) SubmitJob(job Job) {
	wp.wg.Add(1)
	wp.jobs <- job
}

func (wp *WorkerPool) Shutdown() {
	for i := 0; i < wp.maxWorkers; i++ {
		wp.jobs <- Job{ID: SHUTDOWN_SIGNAL}
	}
	close(wp.jobs)
	close(wp.results)
	close(wp.errs)
	log.Println("All jobs have been completed successfully.")
}

func main() {
	log.SetOutput(os.Stdout)
	ctx := context.Background()

	workerPool := NewWorkerPool(MAX_WORKERS, ctx)
	go workerPool.Run()

	go func() {
		for err := range workerPool.errs {
			log.Printf("Error: %v\n", err)
		}
	}()

	go func() {
		for result := range workerPool.results {
			log.Printf("Result from Worker %d, Job %d: %v\n", result.WorkerID, result.JobID, result.Data)
		}
	}()

	for i := 1; i <= MAX_JOBS; i++ {
		job := NewJob(i, NoopExecute)
		workerPool.SubmitJob(job)
	}

	workerPool.wg.Wait()
	workerPool.Shutdown()
}
