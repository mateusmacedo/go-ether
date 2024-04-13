package main

import (
	"context"
	"fmt"
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

type Worker struct {
	ID int
	wg *sync.WaitGroup
}

func (w *Worker) Start(ctx context.Context, jobs <-chan Job, results chan<- Result, errs chan<- error) {
	defer w.wg.Done()

	for job := range jobs {
		result, err := job.Execute(ctx)
		if err != nil {
			if err == context.DeadlineExceeded {
				errs <- fmt.Errorf("Worker %d: Job %d timeout error: %s", w.ID, job.ID, err)
			} else {
				errs <- fmt.Errorf("Worker %d: Job %d error: %s", w.ID, job.ID, err)
			}
		} else {
			results <- Result{WorkerID: w.ID, JobID: job.ID, Data: result}
		}
	}
	fmt.Printf("Worker %d stopped\n", w.ID)
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
}

func NewWorkerPool(maxWorkers int) *WorkerPool {
	return &WorkerPool{
		jobs:       make(chan Job),
		results:    make(chan Result),
		errs:       make(chan error),
		maxWorkers: maxWorkers,
	}
}

func (wp *WorkerPool) Run(ctx context.Context) {
	var wg sync.WaitGroup

	for i := 0; i < wp.maxWorkers; i++ {
		worker := NewWorker(i+1, &wg)
		go worker.Start(ctx, wp.jobs, wp.results, wp.errs)
	}

	wg.Wait()
	close(wp.results)
	close(wp.errs)
}

func (wp *WorkerPool) SubmitJob(job Job) {
	wp.jobs <- job
}

func (wp *WorkerPool) Close() {
	close(wp.jobs)
}

func main() {
	workerPool := NewWorkerPool(3)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		for result := range workerPool.results {
			fmt.Printf("Result from Worker %d, Job %d: %v\n", result.WorkerID, result.JobID, result.Data)
		}
	}()

	go func() {
		for err := range workerPool.errs {
			fmt.Printf("Error: %v\n", err)
		}
	}()

	go workerPool.Run(ctx)

	for i := 1; i <= 30; i++ {
		job := Job{
			ID: i,
			Execute: func(jobctx context.Context) (interface{}, error) {
				select {
				case <-time.After(800 * time.Millisecond): // Assume some jobs might take longer than the 1 second ideal processing time
					return fmt.Sprintf("Job %d completed successfully", i), nil
				case <-jobctx.Done():
					return nil, jobctx.Err() // Properly handle context cancellation due to timeout
				}
			},
		}

		workerPool.SubmitJob(job)
	}

	workerPool.Close()
	fmt.Println("All jobs have been processed or timed out.")
}
