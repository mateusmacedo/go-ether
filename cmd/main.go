package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Result struct holds the outcome from a job executed by a worker.
type Result struct {
	WorkerID int
	JobID    int
	Data     interface{}
}

// Job struct encapsulates an ID and an executable function that simulates workload.
type Job struct {
	ID      int
	Execute func(context.Context) (interface{}, error)
}

// Worker struct represents a worker with an ID and a waitgroup to synchronize.
type Worker struct {
	ID int
	wg *sync.WaitGroup
}

// Start is the method to initiate the worker to process jobs from the jobs channel.
func (w *Worker) Start(ctx context.Context, jobs <-chan Job, results chan<- Result, errs chan<- error) {
	defer w.wg.Done()  // Ensures that the worker's goroutine signals it's done on completion.

	for job := range jobs { // Continuously receive job from jobs channel until it's closed.
		fmt.Printf("Worker %d started job %d\n", w.ID, job.ID)
		result, err := job.Execute(ctx)

		if err != nil {
			errs <- fmt.Errorf("Worker %d job %d error: %s", w.ID, job.ID, err)
		} else {
			results <- Result{WorkerID: w.ID, JobID: job.ID, Data: result}
		}
	}

	fmt.Printf("Worker %d stopped\n", w.ID)
}

// NewWorker creates a new worker with a given ID and a waitgroup.
func NewWorker(number int, wg *sync.WaitGroup) Worker {
	wg.Add(1)  // Increment waitgroup counter on creation of a worker
	return Worker{
		ID: number,
		wg: wg,
	}
}

// WorkerPool struct defines the pool of workers including channels for jobs and results.
type WorkerPool struct {
	jobs       chan Job
	results    chan Result
	errs       chan error
	maxWorkers int
}

// NewWorkerPool initializes a new pool of workers.
func NewWorkerPool(maxWorkers int) *WorkerPool {
	return &WorkerPool{
		jobs:       make(chan Job),
		results:    make(chan Result),
		errs:       make(chan error),
		maxWorkers: maxWorkers,
	}
}

// Run starts all workers in separate goroutines to process jobs.
func (wp *WorkerPool) Run(ctx context.Context) {
	var wg sync.WaitGroup

	for i := 0; i < wp.maxWorkers; i++ {
		worker := NewWorker(i+1, &wg)
		go worker.Start(ctx, wp.jobs, wp.results, wp.errs)
	}

	wg.Wait()  // Wait for all workers to finish
	close(wp.results)  // Close results channel after all jobs are processed
	close(wp.errs)  // Close errors channel after all jobs are processed
}

// SubmitJob sends a job to the jobs channel for the workers to pick up.
func (wp *WorkerPool) SubmitJob(job Job) {
	wp.jobs <- job
}

// Close signals to workers by closing the jobs channel.
func (wp *WorkerPool) Close() {
	close(wp.jobs)
}

func main() {
	workerPool := NewWorkerPool(3)  // create pool with 3 workers

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		for result := range workerPool.results {
			fmt.Printf("Worker %d Job %d result: %v\n", result.WorkerID, result.JobID, result.Data)
		}
	}()

	go func() {
		for err := range workerPool.errs {
			fmt.Printf("%v\n", err)
		}
	}()

	go workerPool.Run(ctx)

	for i := 1; i <= 30; i++ {
		job := Job{
			ID: i,
			Execute: func(jobctx context.Context) (interface{}, error) {
				select {
				case <-time.After(1 * time.Second):
					return fmt.Sprintf("Job completed: %d", i*2), nil
				case <-jobctx.Done():
					return nil, fmt.Errorf("Job %d canceled: %s", i, jobctx.Err())
				}
			},
		}

		workerPool.SubmitJob(job)
		fmt.Printf("Job %d queued\n", i)
	}

	workerPool.Close()
	fmt.Println("All jobs processed")
}
