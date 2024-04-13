package main

import (
	"fmt"
	"sync"
	"time"
)

type Job struct {
    ID      int
    Process func() error
}

type Worker struct {
    ID          int
    jobChannel  chan Job
    wg          *sync.WaitGroup
}

func (w *Worker) Start(workerPool chan chan Job) {
	go func() {
        for {
            workerPool <- w.jobChannel
            job, ok := <-w.jobChannel
            if !ok {
                return // exit if channel is closed
            }
            fmt.Printf("Worker %d started job %d\n", w.ID, job.ID)
            err := job.Process()
            if err != nil {
                fmt.Printf("Worker %d error: %s\n", w.ID, err.Error())
            }
            fmt.Printf("Worker %d finished job %d\n", w.ID, job.ID)
            w.wg.Done()
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

func (d *Dispatcher) dispatch() {
    for i := 0; i < d.maxWorkers; i++ {
        worker := NewWorker(i+1, &d.wg)
		worker.Start(d.workerPool)
        fmt.Printf("Worker %d created\n", worker.ID)
    }

    for {
        job := <-d.jobQueue // receive a job from the main job queue
        fmt.Printf("Dispatcher received job %d\n", job.ID)
        jobChannel := <-d.workerPool // get an available worker's job channel
        jobChannel <- job            // dispatch job to worker
    }
}

func (d *Dispatcher) run() {
    go d.dispatch()
}

func main() {
    jobs := make(chan Job, 100)
    dispatcher := NewDispatcher(jobs, 3) // create dispatcher with 3 workers

    jobCount := 100 // number of jobs to process

    dispatcher.run() // start dispatching jobs

    for i := 1; i <= jobCount; i++ {
        dispatcher.wg.Add(1)
        jobs <- Job{ID: i, Process: func() error {
            time.Sleep(100 * time.Millisecond) // simulate a task that takes 1 second
            return nil
        }}
        fmt.Printf("Job %d queued\n", i)
    }

    dispatcher.wg.Wait() // wait for all jobs to be processed
    fmt.Println("All jobs processed")
}
