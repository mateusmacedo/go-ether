package workerpool

import (
	"context"
	"sync"

	jobpkg "github.com/mateusmacedo/goether/worker/pkg/job"
	"github.com/mateusmacedo/goether/worker/pkg/worker"
)

type WorkerPool interface {
	Run()
	Submit(jobpkg.Job)
	Wait()
	Shutdown()
	Jobs() chan jobpkg.Job
	Results() chan jobpkg.Result
	Errs() chan error
}

type workerPool struct {
	jobs       chan jobpkg.Job
	results    chan jobpkg.Result
	errs       chan error
	maxWorkers int
	wg         sync.WaitGroup
	ctx        context.Context
}

func New(maxWorkers int, ctx context.Context) WorkerPool {
	return &workerPool{
		jobs:       make(chan jobpkg.Job, jobpkg.MAX_JOBS),
		results:    make(chan jobpkg.Result, jobpkg.MAX_JOBS),
		errs:       make(chan error, jobpkg.MAX_JOBS),
		maxWorkers: maxWorkers,
		ctx:        ctx,
	}
}

func (wp *workerPool) Jobs() chan jobpkg.Job {
	return wp.jobs
}

func (wp *workerPool) Results() chan jobpkg.Result {
	return wp.results
}

func (wp *workerPool) Errs() chan error {
	return wp.errs
}

func (wp *workerPool) Run() {
	for i := 0; i < wp.maxWorkers; i++ {
		w := worker.New(i+1, &wp.wg)
		go w.Start(wp.ctx, wp.jobs, wp.results, wp.errs)
	}
}

func (wp *workerPool) Submit(job jobpkg.Job) {
	wp.jobs <- job
}

func (wp *workerPool) Wait() {
	wp.wg.Wait()
}

func (wp *workerPool) Shutdown() {
	for i := 0; i < wp.maxWorkers; i++ {
		wp.jobs <- jobpkg.Job{ID: jobpkg.SHUTDOWN_SIGNAL}
	}
	wp.wg.Wait()
	close(wp.jobs)
	close(wp.results)
	close(wp.errs)
}