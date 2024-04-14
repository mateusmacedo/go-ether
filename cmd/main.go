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
	MAX_WORKERS     = 2
	MAX_JOBS        = 10
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
	defer w.wg.Done() // Ensures this is called even if the method exits early
	for job := range jobs {
		if job.ID == SHUTDOWN_SIGNAL {
			return // Exit the loop (and goroutine) if we receive a shutdown signal
		}

		jobCtx := context.WithValue(ctx, WORKER_ID_KEY, w.ID)
		jobCtx = context.WithValue(jobCtx, JOB_ID_KEY, job.ID)

		result, err := job.Execute(jobCtx)
		if err != nil {
			errs <- err
			continue
		}
		results <- Result{WorkerID: w.ID, JobID: job.ID, Data: result}
	}
}

func NewWorker(number int, wg *sync.WaitGroup) Worker {
	wg.Add(1) // Properly place wg.Add(1) to avoid race condition
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
		wp.jobs <- Job{ID: SHUTDOWN_SIGNAL} // Send shutdown signal to each worker
	}
	wp.wg.Wait()   // Wait for all workers to finish
	close(wp.jobs) // Closing jobs channel after all signals are sent
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

	// Handle results
	go func() {
		for result := range workerPool.results {
			log.Printf("Result from Worker %d, Job %d: %v\n", result.WorkerID, result.JobID, result.Data)
		}
	}()

	// Handle errors
	go func() {
		for err := range workerPool.errs {
			log.Printf("Error: %v\n", err)
		}
	}()

	workerPool.wg.Wait()
	workerPool.Shutdown()
}
