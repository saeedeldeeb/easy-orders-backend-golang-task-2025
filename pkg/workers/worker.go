package workers

import (
	"context"
	"sync/atomic"
	"time"

	"easy-orders-backend/pkg/logger"
)

// Worker represents a single worker in the pool
type Worker struct {
	id          int
	jobQueue    <-chan Job
	resultQueue chan<- JobResult
	logger      *logger.Logger
	active      int32 // atomic flag for active status
}

// NewWorker creates a new worker
func NewWorker(id int, jobQueue <-chan Job, resultQueue chan<- JobResult, logger *logger.Logger) *Worker {
	return &Worker{
		id:          id,
		jobQueue:    jobQueue,
		resultQueue: resultQueue,
		logger:      logger,
	}
}

// Start begins the worker's job processing loop
func (w *Worker) Start(ctx context.Context) {
	w.logger.Debug("Worker started", "worker_id", w.id)

	for {
		select {
		case <-ctx.Done():
			w.logger.Debug("Worker stopped due to context cancellation", "worker_id", w.id)
			return
		case job, ok := <-w.jobQueue:
			if !ok {
				w.logger.Debug("Worker stopped due to closed job queue", "worker_id", w.id)
				return
			}

			w.processJob(ctx, job)
		}
	}
}

// processJob executes a single job and reports the result
func (w *Worker) processJob(ctx context.Context, job Job) {
	atomic.StoreInt32(&w.active, 1)
	defer atomic.StoreInt32(&w.active, 0)

	startTime := time.Now()

	w.logger.Debug("Worker processing job",
		"worker_id", w.id,
		"job_id", job.GetID(),
		"job_type", job.GetType(),
		"priority", job.GetPriority(),
		"retry_count", job.GetRetryCount())

	// Execute the job
	err := job.Execute(ctx)

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	// Create result
	result := JobResult{
		JobID:     job.GetID(),
		JobType:   job.GetType(),
		Success:   err == nil,
		Error:     err,
		Duration:  duration,
		StartTime: startTime,
		EndTime:   endTime,
		WorkerID:  w.id,
	}

	// Log result
	if err != nil {
		w.logger.Warn("Job execution failed",
			"worker_id", w.id,
			"job_id", job.GetID(),
			"job_type", job.GetType(),
			"duration_ms", duration.Milliseconds(),
			"error", err)
	} else {
		w.logger.Debug("Job executed successfully",
			"worker_id", w.id,
			"job_id", job.GetID(),
			"job_type", job.GetType(),
			"duration_ms", duration.Milliseconds())
	}

	// Send result (non-blocking)
	select {
	case w.resultQueue <- result:
		// Result sent successfully
	default:
		w.logger.Warn("Result queue is full, dropping result",
			"worker_id", w.id,
			"job_id", job.GetID())
	}
}

// IsActive returns whether the worker is currently processing a job
func (w *Worker) IsActive() bool {
	return atomic.LoadInt32(&w.active) == 1
}

// GetID returns the worker's ID
func (w *Worker) GetID() int {
	return w.id
}
