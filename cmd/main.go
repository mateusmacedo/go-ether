package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Result represents the result of a job
type Result struct {
    JobID int
    Data  interface{}
}

// Job represents a job to be executed
type Job struct {
    ID      int
    Process func(context.Context) (interface{}, error)
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
                result, err := job.Process(ctx)
                if err != nil {
                    fmt.Printf("Worker %d job %d error: %s\n", w.ID, job.ID, err.Error())
                    errs <- err
                } else {
                    fmt.Printf("Worker %d finished job %d\n", w.ID, job.ID)
                    results <- Result{JobID: job.ID, Data: result}
                }
                w.wg.Done()
            case <-ctx.Done():
                fmt.Printf("Worker %d stopped\n", w.ID)
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

// Dispatcher sends the jobs to available workers
type Dispatcher struct {
    jobs       chan Job
    maxWorkers int
    wg         sync.WaitGroup
}

// NewDispatcher creates a new dispatcher
func NewDispatcher(maxWorkers int) *Dispatcher {
    return &Dispatcher{
        jobs:       make(chan Job, 100),
        maxWorkers: maxWorkers,
    }
}

// Dispatch starts the dispatcher
func (d *Dispatcher) Dispatch(ctx context.Context, results chan<- Result, errs chan<- error) {
    for i := 0; i < d.maxWorkers; i++ {
        worker := NewWorker(i+1, &d.wg)
        worker.Start(ctx, d.jobs, results, errs)
        fmt.Printf("Worker %d created\n", worker.ID)
    }

    go func() {
        <-ctx.Done()
        close(d.jobs)
        fmt.Println("Dispatcher stopped")
    }()
}

// AddJob adds a job to the dispatcher
func (d *Dispatcher) AddJob(job Job) {
    d.jobs <- job
    d.wg.Add(1)
}

// Wait waits for all jobs to be processed
func (d *Dispatcher) Wait() {
    d.wg.Wait()
}

func main() {
    dispatcher := NewDispatcher(3) // create dispatcher with 3 workers

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    results := make(chan Result)
    errs := make(chan error)

    go func() {
        for result := range results {
            fmt.Printf("Job %d result: %v\n", result.JobID, result.Data)
        }
    }()

    go func() {
        for err := range errs {
            fmt.Println("Error:", err)
        }
    }()

    dispatcher.Dispatch(ctx, results, errs) // start dispatching jobs with a context

    for i := 1; i <= 20; i++ {
        job := Job{
            ID: i,
            Process: func(jobctx context.Context) (interface{}, error) {
                select {
                case <-time.After(1 * time.Second): // simulate a task
                    return fmt.Sprintf("Job completed: %d", i*2), nil
                case <-jobctx.Done():
                    return nil, jobctx.Err()
                }
            },
        }
        dispatcher.AddJob(job)
        fmt.Printf("Job %d queued\n", i)
    }

    dispatcher.Wait() // wait for all jobs to be processed
    close(results)
    close(errs)
    fmt.Println("All jobs processed")
}