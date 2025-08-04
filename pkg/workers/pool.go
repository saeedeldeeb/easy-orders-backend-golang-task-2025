package workers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"easy-orders-backend/pkg/logger"
)

// Job represents a unit of work to be processed
type Job interface {
	// Execute runs the job and returns an error if it fails
	Execute(ctx context.Context) error

	// GetID returns a unique identifier for this job
	GetID() string

	// GetType returns the type/category of this job
	GetType() string

	// GetPriority returns the priority level (higher number = higher priority)
	GetPriority() int

	// GetRetryCount returns current retry attempt count
	GetRetryCount() int

	// IncrementRetryCount increments the retry counter
	IncrementRetryCount()

	// GetMaxRetries returns maximum allowed retries
	GetMaxRetries() int
}

// JobResult represents the result of job execution
type JobResult struct {
	JobID     string        `json:"job_id"`
	JobType   string        `json:"job_type"`
	Success   bool          `json:"success"`
	Error     error         `json:"error,omitempty"`
	Duration  time.Duration `json:"duration"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	WorkerID  int           `json:"worker_id"`
}

// WorkerPoolConfig defines the configuration for a worker pool
type WorkerPoolConfig struct {
	Name            string        `json:"name"`
	WorkerCount     int           `json:"worker_count"`
	QueueSize       int           `json:"queue_size"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`
	RetryDelay      time.Duration `json:"retry_delay"`
	EnableMetrics   bool          `json:"enable_metrics"`
}

// DefaultWorkerPoolConfig returns a sensible default configuration
func DefaultWorkerPoolConfig(name string) *WorkerPoolConfig {
	return &WorkerPoolConfig{
		Name:            name,
		WorkerCount:     5,
		QueueSize:       100,
		ShutdownTimeout: 30 * time.Second,
		RetryDelay:      1 * time.Second,
		EnableMetrics:   true,
	}
}

// WorkerPool manages a pool of workers processing jobs
type WorkerPool struct {
	config *WorkerPoolConfig
	logger *logger.Logger

	// Channels
	jobQueue    chan Job
	resultQueue chan JobResult
	shutdown    chan struct{}

	// Worker management
	workers []*Worker
	wg      sync.WaitGroup

	// State management
	running    bool
	runningMux sync.RWMutex

	// Metrics
	metrics *PoolMetrics
}

// PoolMetrics tracks worker pool performance
type PoolMetrics struct {
	JobsProcessed  int64         `json:"jobs_processed"`
	JobsSucceeded  int64         `json:"jobs_succeeded"`
	JobsFailed     int64         `json:"jobs_failed"`
	JobsRetried    int64         `json:"jobs_retried"`
	AverageLatency time.Duration `json:"average_latency"`
	TotalLatency   time.Duration `json:"total_latency"`
	ActiveWorkers  int           `json:"active_workers"`
	QueueDepth     int           `json:"queue_depth"`
	mux            sync.RWMutex
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(config *WorkerPoolConfig, logger *logger.Logger) *WorkerPool {
	if config == nil {
		config = DefaultWorkerPoolConfig("default")
	}

	pool := &WorkerPool{
		config:      config,
		logger:      logger,
		jobQueue:    make(chan Job, config.QueueSize),
		resultQueue: make(chan JobResult, config.QueueSize),
		shutdown:    make(chan struct{}),
		workers:     make([]*Worker, config.WorkerCount),
		metrics:     &PoolMetrics{},
	}

	return pool
}

// Start initializes and starts the worker pool
func (wp *WorkerPool) Start(ctx context.Context) error {
	wp.runningMux.Lock()
	defer wp.runningMux.Unlock()

	if wp.running {
		return fmt.Errorf("worker pool %s is already running", wp.config.Name)
	}

	wp.logger.Info("Starting worker pool",
		"name", wp.config.Name,
		"worker_count", wp.config.WorkerCount,
		"queue_size", wp.config.QueueSize)

	// Start workers
	for i := 0; i < wp.config.WorkerCount; i++ {
		worker := NewWorker(i+1, wp.jobQueue, wp.resultQueue, wp.logger)
		wp.workers[i] = worker

		wp.wg.Add(1)
		go func(w *Worker) {
			defer wp.wg.Done()
			w.Start(ctx)
		}(worker)
	}

	// Start result processor if metrics are enabled
	if wp.config.EnableMetrics {
		wp.wg.Add(1)
		go func() {
			defer wp.wg.Done()
			wp.processResults(ctx)
		}()
	}

	wp.running = true
	wp.logger.Info("Worker pool started successfully", "name", wp.config.Name)

	return nil
}

// Stop gracefully shuts down the worker pool
func (wp *WorkerPool) Stop() error {
	wp.runningMux.Lock()
	defer wp.runningMux.Unlock()

	if !wp.running {
		return fmt.Errorf("worker pool %s is not running", wp.config.Name)
	}

	wp.logger.Info("Stopping worker pool", "name", wp.config.Name)

	// Signal shutdown
	close(wp.shutdown)
	close(wp.jobQueue)

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		wp.logger.Info("Worker pool stopped gracefully", "name", wp.config.Name)
	case <-time.After(wp.config.ShutdownTimeout):
		wp.logger.Warn("Worker pool shutdown timeout", "name", wp.config.Name)
	}

	wp.running = false
	return nil
}

// SubmitJob adds a job to the processing queue
func (wp *WorkerPool) SubmitJob(job Job) error {
	wp.runningMux.RLock()
	defer wp.runningMux.RUnlock()

	if !wp.running {
		return fmt.Errorf("worker pool %s is not running", wp.config.Name)
	}

	select {
	case wp.jobQueue <- job:
		wp.logger.Debug("Job submitted to pool",
			"pool", wp.config.Name,
			"job_id", job.GetID(),
			"job_type", job.GetType(),
			"priority", job.GetPriority())
		return nil
	default:
		return fmt.Errorf("job queue is full for pool %s", wp.config.Name)
	}
}

// SubmitJobWithPriority adds a high-priority job that should be processed first
func (wp *WorkerPool) SubmitJobWithPriority(job Job) error {
	// For now, treat as regular job. Could implement priority queue later
	return wp.SubmitJob(job)
}

// GetMetrics returns current pool metrics
func (wp *WorkerPool) GetMetrics() *PoolMetrics {
	wp.metrics.mux.RLock()
	defer wp.metrics.mux.RUnlock()

	// Update queue depth
	wp.metrics.QueueDepth = len(wp.jobQueue)

	// Count active workers
	activeWorkers := 0
	for _, worker := range wp.workers {
		if worker.IsActive() {
			activeWorkers++
		}
	}
	wp.metrics.ActiveWorkers = activeWorkers

	// Calculate average latency
	if wp.metrics.JobsProcessed > 0 {
		wp.metrics.AverageLatency = time.Duration(int64(wp.metrics.TotalLatency) / wp.metrics.JobsProcessed)
	}

	// Return copy of metrics
	return &PoolMetrics{
		JobsProcessed:  wp.metrics.JobsProcessed,
		JobsSucceeded:  wp.metrics.JobsSucceeded,
		JobsFailed:     wp.metrics.JobsFailed,
		JobsRetried:    wp.metrics.JobsRetried,
		AverageLatency: wp.metrics.AverageLatency,
		TotalLatency:   wp.metrics.TotalLatency,
		ActiveWorkers:  activeWorkers,
		QueueDepth:     len(wp.jobQueue),
	}
}

// IsRunning returns whether the pool is currently running
func (wp *WorkerPool) IsRunning() bool {
	wp.runningMux.RLock()
	defer wp.runningMux.RUnlock()
	return wp.running
}

// GetQueueDepth returns the current number of jobs in the queue
func (wp *WorkerPool) GetQueueDepth() int {
	return len(wp.jobQueue)
}

// processResults processes job results and updates metrics
func (wp *WorkerPool) processResults(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-wp.shutdown:
			return
		case result := <-wp.resultQueue:
			wp.updateMetrics(result)

			// Handle job retry logic
			if !result.Success && needsRetry(result) {
				wp.logger.Debug("Retrying failed job",
					"job_id", result.JobID,
					"job_type", result.JobType,
					"attempt", "retry",
					"error", result.Error)

				// TODO: Implement job retry with backoff
				// This would require storing the original job
			}
		}
	}
}

// updateMetrics updates pool metrics with job result
func (wp *WorkerPool) updateMetrics(result JobResult) {
	wp.metrics.mux.Lock()
	defer wp.metrics.mux.Unlock()

	wp.metrics.JobsProcessed++
	wp.metrics.TotalLatency += result.Duration

	if result.Success {
		wp.metrics.JobsSucceeded++
	} else {
		wp.metrics.JobsFailed++
	}
}

// needsRetry determines if a job should be retried based on the result
func needsRetry(result JobResult) bool {
	// Implement retry logic based on error type, job type, etc.
	// For now, return false - retry logic to be implemented
	return false
}
