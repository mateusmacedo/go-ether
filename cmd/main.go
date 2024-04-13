package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Result represents the result of a job
type Result struct {
	WorkerID int
	JobID    int
	Data     interface{}
}

// Job represents a job to be executed
type Job struct {
	ID      int
	Execute func(context.Context) (interface{}, error)
}

// Worker represents a worker that can execute jobs
type Worker struct {
	ID int
	wg *sync.WaitGroup
}

// Start starts the worker
func (w *Worker) Start(ctx context.Context, jobs <-chan Job, results chan<- Result, errs chan<- error) {
	go func() {
		for {
			select {
			case job, ok := <-jobs:
				if !ok {
					fmt.Printf("Worker %d stopped\n", w.ID)
					return // exit if jobs channel is closed
				}
				fmt.Printf("Worker %d started job %d\n", w.ID, job.ID)
				result, err := job.Execute(ctx)
				w.wg.Done()
				if err != nil {
					errs <- errors.New(fmt.Sprintf("Worker %d job %d error: %s", w.ID, job.ID, err))
				} else {
					results <- Result{WorkerID: w.ID, JobID: job.ID, Data: result}
				}
			case <-ctx.Done():
				errs <- errors.New(fmt.Sprintf("Worker %d stopped", w.ID))
				return
			}
		}
	}()
}

// NewWorker creates a new worker
func NewWorker(number int, wg *sync.WaitGroup) Worker {
	return Worker{
		ID: number,
		wg: wg,
	}
}

// WorkerPool sends the jobs to available workers
type WorkerPool struct {
	jobs       chan Job
	results    chan Result
	errs       chan error
	maxWorkers int
	bufferSize int
	wg         sync.WaitGroup
}

// NewWorkerPool creates a new dispatcher
func NewWorkerPool(maxWorkers, bufferSize int) *WorkerPool {
	return &WorkerPool{
		jobs:       make(chan Job, bufferSize),
		results:    make(chan Result, bufferSize),
		errs:       make(chan error, bufferSize),
		maxWorkers: maxWorkers,
		bufferSize: bufferSize,
	}
}

// Run starts the dispatcher
func (wp *WorkerPool) Run(ctx context.Context) {
	for i := 0; i < wp.maxWorkers; i++ {
		worker := NewWorker(i+1, &wp.wg)
		go func() {
			worker.Start(ctx, wp.jobs, wp.results, wp.errs)
		}()
		fmt.Printf("Worker %d created\n", worker.ID)
	}
}

// SubmitJob adds a job to the dispatcher
func (wp *WorkerPool) SubmitJob(job Job) {
	wp.wg.Add(1)
	wp.jobs <- job
}

// Wait waits for all jobs to be processed
func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
}

func (wp *WorkerPool) Close() {
	close(wp.jobs)
	close(wp.results)
	close(wp.errs)
}

func main() {
	workerpool := NewWorkerPool(3, 100) // create dispatcher with 3 workers

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		for result := range workerpool.results {
			fmt.Printf("Worker %d Job %d result: %v\n", result.WorkerID, result.JobID, result.Data)
		}
	}()

	go func() {
		for err := range workerpool.errs {
			fmt.Printf("%v\n", err)
		}
	}()

	workerpool.Run(ctx)

	for i := 1; i <= 30; i++ {
		jobNum := i // new variable to capture each job number correctly
		job := Job{
			ID: i,
			Execute: func(jobctx context.Context) (interface{}, error) {
				select {
				case <-time.After(1 * time.Second):
					return fmt.Sprintf("Job completed: %d", jobNum*2), nil
				case <-jobctx.Done():
					return nil, errors.New(fmt.Sprintf("%s", jobctx.Err()))
				}
			},
		}
		workerpool.SubmitJob(job)
		fmt.Printf("Job %d queued\n", i)
	}

	workerpool.Wait() // wait for all jobs to be processed
	fmt.Println("All jobs processed")
}
