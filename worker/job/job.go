package job

import (
	"context"
	"time"
)

const (
	JOB_ID_KEY      = "JobID"
	WORKER_ID_KEY   = "WorkerID"
	MAX_WORKERS     = 3
	MAX_JOBS        = 60
	SHUTDOWN_SIGNAL = -1
	EXECUTE_TIME_AFTER   = 3001 * time.Millisecond
	CONTEXT_TIMEOUT = 3000 * time.Millisecond
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

func New(id int, execute func(context.Context) (interface{}, error)) Job {
	return Job{ID: id, Execute: execute}
}