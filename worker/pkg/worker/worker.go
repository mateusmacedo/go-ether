package worker

import (
	"context"
	"fmt"
	"sync"

	jobpkg "github.com/mateusmacedo/go-ether/worker/pkg/job"
)

type Worker struct {
	ID int
	wg *sync.WaitGroup
}

func (w *Worker) Start(ctx context.Context, jobs <-chan jobpkg.Job, results chan<- jobpkg.Result, errs chan<- error) {
	defer w.wg.Done()
	for job := range jobs {
		if job.ID == jobpkg.SHUTDOWN_SIGNAL {
			return
		}
		jobCtx, cancel := context.WithTimeout(ctx, jobpkg.CONTEXT_TIMEOUT)
		defer cancel()

		jobCtx = context.WithValue(jobCtx, jobpkg.WORKER_ID_KEY, w.ID)
		jobCtx = context.WithValue(jobCtx, jobpkg.JOB_ID_KEY, job.ID)

		result, err := job.Execute(jobCtx)
		if err != nil {
			errs <- fmt.Errorf("worker %d: job %d error: %s", w.ID, job.ID, err)
			continue
		}
		results <- jobpkg.Result{WorkerID: w.ID, JobID: job.ID, Data: result}
	}
}

func New(number int, wg *sync.WaitGroup) Worker {
	wg.Add(1)
	return Worker{ID: number, wg: wg}
}