package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Job struct {
    ID      int
    Process func(context.Context) error
}

type Worker struct {
    ID          int
    jobChannel  chan Job
    wg          *sync.WaitGroup
}

func (w *Worker) Start(ctx context.Context, workerPool chan chan Job) {
    go func() {
        for {
            select {
            case workerPool <- w.jobChannel:
                select {
                case job, ok := <-w.jobChannel:
                    if !ok {
                        return // exit if channel is closed
                    }
                    fmt.Printf("Worker %d started job %d\n", w.ID, job.ID)
                    err := job.Process(ctx)
                    if err != nil {
                        fmt.Printf("Worker %d job %d error: %s\n", w.ID, job.ID, err.Error())
                    } else {
                        fmt.Printf("Worker %d finished job %d\n", w.ID, job.ID)
                    }
                    w.wg.Done()
                case <-ctx.Done():
                    fmt.Printf("Worker %d stopped\n", w.ID)
                    return
                }
            case <-ctx.Done():
                fmt.Printf("Worker %d stopped\n", w.ID)
                return
            }
        }
    }()
}

func NewWorker(number int, wg *sync.WaitGroup) Worker {
    worker := Worker{
        ID:         number,
        jobChannel: make(chan Job),
        wg:         wg,
    }

    return worker
}

// Dispatcher - A struct that sends the jobs to available workers
type Dispatcher struct {
    jobQueue   chan Job
    workerPool chan chan Job
    maxWorkers int
    wg         sync.WaitGroup
}

func NewDispatcher(jobQueue chan Job, maxWorkers int) *Dispatcher {
    workerPool := make(chan chan Job, maxWorkers)
    return &Dispatcher{jobQueue: jobQueue, workerPool: workerPool, maxWorkers: maxWorkers}
}

func (d *Dispatcher) dispatch(ctx context.Context) {
    for i := 0; i < d.maxWorkers; i++ {
        worker := NewWorker(i+1, &d.wg)
        worker.Start(ctx, d.workerPool)
        fmt.Printf("Worker %d created\n", worker.ID)
    }

    for {
        select {
        case job := <-d.jobQueue:
            jobChannel := <-d.workerPool
            jobChannel <- job
        case <-ctx.Done():
            fmt.Println("Dispatcher stopped")
            return
        }
    }
}

func (d *Dispatcher) run(ctx context.Context) {
    go d.dispatch(ctx)
}

func main() {
    jobs := make(chan Job, 100)
    dispatcher := NewDispatcher(jobs, 3) // create dispatcher with 3 workers

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    dispatcher.run(ctx) // start dispatching jobs with a context

    for i := 1; i <= 20; i++ {
        dispatcher.wg.Add(1)
        job := Job{
            ID: i,
            Process: func(jobctx context.Context) error {
                select {
                case <-time.After(1 * time.Second): // simulate a task
                    fmt.Println("Job completed:", i)
                    return nil
                case <-jobctx.Done():
                    return jobctx.Err()
                }
            },
        }
        jobs <- job
        fmt.Printf("Job %d queued\n", i)
    }

    dispatcher.wg.Wait() // wait for all jobs to be processed
    fmt.Println("All jobs processed")
}