package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mateusmacedo/goether/worker/pkg/job"
	"github.com/mateusmacedo/goether/worker/pkg/workerpool"
)

func dummyExecute(ctx context.Context) (interface{}, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(job.EXECUTE_TIME_AFTER):
		workerID, workerOk := ctx.Value(job.WORKER_ID_KEY).(int)
		jobID, jobOk := ctx.Value(job.JOB_ID_KEY).(int)
		if !workerOk || !jobOk {
			return nil, fmt.Errorf("context values missing or wrong type: workerID=%v, jobID=%v, workerOk=%t, jobOk=%t", workerID, jobID, workerOk, jobOk)
		}
		return fmt.Sprintf("Worker: %d completed Job: %d successfully.", workerID, jobID), nil
	}
}

func main() {
	log.SetOutput(os.Stdout)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	workerPool := workerpool.New(job.MAX_WORKERS, ctx)
	go workerPool.Run()

	for i := 1; i <= job.MAX_JOBS; i++ {
		workerPool.Submit(job.New(i, dummyExecute))
	}

	go func() {
		for result := range workerPool.Results() {
			log.Printf("Result from Worker %d, Job %d: %v\n", result.WorkerID, result.JobID, result.Data)
		}
	}()

	go func() {
		for err := range workerPool.Errs() {
			log.Printf("Error: %v\n", err)
		}
	}()

	workerPool.Wait()
	workerPool.Shutdown()
}
