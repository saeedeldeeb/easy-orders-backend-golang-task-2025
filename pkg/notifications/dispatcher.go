package notifications

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"easy-orders-backend/pkg/logger"
)

// DispatcherConfig configures the notification dispatcher
type DispatcherConfig struct {
	WorkerCount        int           `json:"worker_count"`
	QueueSize          int           `json:"queue_size"`
	BatchSize          int           `json:"batch_size"`
	FlushInterval      time.Duration `json:"flush_interval"`
	RetryCheckInterval time.Duration `json:"retry_check_interval"`
	MaxRetryDelay      time.Duration `json:"max_retry_delay"`
	EnableMetrics      bool          `json:"enable_metrics"`
}

// DefaultDispatcherConfig returns default configuration
func DefaultDispatcherConfig() *DispatcherConfig {
	return &DispatcherConfig{
		WorkerCount:        10,
		QueueSize:          1000,
		BatchSize:          50,
		FlushInterval:      5 * time.Second,
		RetryCheckInterval: 30 * time.Second,
		MaxRetryDelay:      time.Hour,
		EnableMetrics:      true,
	}
}

// NotificationDispatcher manages asynchronous notification processing
type NotificationDispatcher struct {
	config   *DispatcherConfig
	provider *NotificationProvider
	logger   *logger.Logger

	// Channels for communication
	notificationQueue chan *Notification
	retryQueue        chan *Notification
	stopChan          chan struct{}

	// Worker management
	workers []*DispatchWorker
	wg      sync.WaitGroup
	running int32 // atomic

	// Batching
	batchMutex sync.Mutex
	batches    map[string][]*Notification // keyed by channel name
	lastFlush  time.Time

	// Metrics
	metrics *DispatcherMetrics
}

// DispatcherMetrics tracks dispatcher performance
type DispatcherMetrics struct {
	NotificationsSent    int64         `json:"notifications_sent"`
	NotificationsFailed  int64         `json:"notifications_failed"`
	NotificationsRetried int64         `json:"notifications_retried"`
	NotificationsExpired int64         `json:"notifications_expired"`
	AverageLatency       time.Duration `json:"average_latency"`
	TotalLatency         time.Duration `json:"total_latency"`
	QueueDepth           int           `json:"queue_depth"`
	ActiveWorkers        int           `json:"active_workers"`
	mutex                sync.RWMutex
}

// NewNotificationDispatcher creates a new notification dispatcher
func NewNotificationDispatcher(config *DispatcherConfig, provider *NotificationProvider, logger *logger.Logger) *NotificationDispatcher {
	if config == nil {
		config = DefaultDispatcherConfig()
	}

	return &NotificationDispatcher{
		config:            config,
		provider:          provider,
		logger:            logger,
		notificationQueue: make(chan *Notification, config.QueueSize),
		retryQueue:        make(chan *Notification, config.QueueSize/2),
		stopChan:          make(chan struct{}),
		batches:           make(map[string][]*Notification),
		lastFlush:         time.Now(),
		metrics:           &DispatcherMetrics{},
	}
}

// Start initializes and starts the notification dispatcher
func (nd *NotificationDispatcher) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&nd.running, 0, 1) {
		return fmt.Errorf("dispatcher is already running")
	}

	nd.logger.Info("Starting notification dispatcher",
		"worker_count", nd.config.WorkerCount,
		"queue_size", nd.config.QueueSize)

	// Start workers
	nd.workers = make([]*DispatchWorker, nd.config.WorkerCount)
	for i := 0; i < nd.config.WorkerCount; i++ {
		worker := NewDispatchWorker(i+1, nd, nd.logger)
		nd.workers[i] = worker

		nd.wg.Add(1)
		go func(w *DispatchWorker) {
			defer nd.wg.Done()
			w.Start(ctx)
		}(worker)
	}

	// Start batch processor
	nd.wg.Add(1)
	go func() {
		defer nd.wg.Done()
		nd.batchProcessor(ctx)
	}()

	// Start retry processor
	nd.wg.Add(1)
	go func() {
		defer nd.wg.Done()
		nd.retryProcessor(ctx)
	}()

	nd.logger.Info("Notification dispatcher started successfully")
	return nil
}

// Stop gracefully shuts down the dispatcher
func (nd *NotificationDispatcher) Stop() error {
	if !atomic.CompareAndSwapInt32(&nd.running, 1, 0) {
		return fmt.Errorf("dispatcher is not running")
	}

	nd.logger.Info("Stopping notification dispatcher")

	// Signal all goroutines to stop
	close(nd.stopChan)
	close(nd.notificationQueue)
	close(nd.retryQueue)

	// Wait for all workers to finish
	nd.wg.Wait()

	nd.logger.Info("Notification dispatcher stopped")
	return nil
}

// Dispatch adds a notification to the processing queue
func (nd *NotificationDispatcher) Dispatch(notification *Notification) error {
	if atomic.LoadInt32(&nd.running) == 0 {
		return fmt.Errorf("dispatcher is not running")
	}

	select {
	case nd.notificationQueue <- notification:
		nd.logger.Debug("Notification queued for dispatch",
			"id", notification.ID,
			"type", notification.Type,
			"channel", notification.Channel,
			"priority", notification.Priority)
		return nil
	default:
		return fmt.Errorf("notification queue is full")
	}
}

// DispatchBatch adds multiple notifications to the processing queue
func (nd *NotificationDispatcher) DispatchBatch(notifications []*Notification) error {
	if atomic.LoadInt32(&nd.running) == 0 {
		return fmt.Errorf("dispatcher is not running")
	}

	// Sort by priority (highest first)
	sort.Slice(notifications, func(i, j int) bool {
		return notifications[i].GetPriorityScore() > notifications[j].GetPriorityScore()
	})

	var failed []string
	for _, notification := range notifications {
		if err := nd.Dispatch(notification); err != nil {
			failed = append(failed, notification.ID)
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("failed to queue %d notifications: %v", len(failed), failed)
	}

	nd.logger.Info("Batch dispatched", "count", len(notifications))
	return nil
}

// GetMetrics returns current dispatcher metrics
func (nd *NotificationDispatcher) GetMetrics() *DispatcherMetrics {
	nd.metrics.mutex.RLock()
	defer nd.metrics.mutex.RUnlock()

	// Update queue depth
	nd.metrics.QueueDepth = len(nd.notificationQueue)

	// Count active workers
	activeWorkers := 0
	for _, worker := range nd.workers {
		if worker.IsActive() {
			activeWorkers++
		}
	}
	nd.metrics.ActiveWorkers = activeWorkers

	// Calculate average latency
	if nd.metrics.NotificationsSent > 0 {
		nd.metrics.AverageLatency = time.Duration(int64(nd.metrics.TotalLatency) / nd.metrics.NotificationsSent)
	}

	// Return a copy of metrics
	return &DispatcherMetrics{
		NotificationsSent:    nd.metrics.NotificationsSent,
		NotificationsFailed:  nd.metrics.NotificationsFailed,
		NotificationsRetried: nd.metrics.NotificationsRetried,
		NotificationsExpired: nd.metrics.NotificationsExpired,
		AverageLatency:       nd.metrics.AverageLatency,
		TotalLatency:         nd.metrics.TotalLatency,
		QueueDepth:           len(nd.notificationQueue),
		ActiveWorkers:        activeWorkers,
	}
}

// batchProcessor handles batching notifications for efficient processing
func (nd *NotificationDispatcher) batchProcessor(ctx context.Context) {
	ticker := time.NewTicker(nd.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			nd.flushAllBatches()
			return
		case <-nd.stopChan:
			nd.flushAllBatches()
			return
		case <-ticker.C:
			nd.flushBatchesIfNeeded()
		case notification := <-nd.notificationQueue:
			nd.addToBatch(notification)
			nd.flushBatchesIfNeeded()
		}
	}
}

// retryProcessor handles failed notifications that need to be retried
func (nd *NotificationDispatcher) retryProcessor(ctx context.Context) {
	ticker := time.NewTicker(nd.config.RetryCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-nd.stopChan:
			return
		case <-ticker.C:
			// Process retry queue is handled by the main queue processor
		case notification := <-nd.retryQueue:
			if notification.IsExpired() {
				nd.updateMetrics(func(m *DispatcherMetrics) {
					m.NotificationsExpired++
				})
				nd.logger.Warn("Notification expired", "id", notification.ID)
				continue
			}

			if notification.ShouldRetry() {
				notification.MarkAsRetrying()

				// Re-queue for retry after delay
				go func(n *Notification) {
					time.Sleep(time.Until(n.GetNextRetryTime()))

					select {
					case nd.notificationQueue <- n:
						nd.updateMetrics(func(m *DispatcherMetrics) {
							m.NotificationsRetried++
						})
					case <-nd.stopChan:
						return
					}
				}(notification)
			}
		}
	}
}

// addToBatch adds a notification to the appropriate batch
func (nd *NotificationDispatcher) addToBatch(notification *Notification) {
	nd.batchMutex.Lock()
	defer nd.batchMutex.Unlock()

	channelName := notification.Channel
	nd.batches[channelName] = append(nd.batches[channelName], notification)
}

// flushBatchesIfNeeded flushes batches if they're full or if enough time has passed
func (nd *NotificationDispatcher) flushBatchesIfNeeded() {
	nd.batchMutex.Lock()
	defer nd.batchMutex.Unlock()

	timeSinceLastFlush := time.Since(nd.lastFlush)
	shouldFlushTime := timeSinceLastFlush >= nd.config.FlushInterval

	for channelName, batch := range nd.batches {
		shouldFlushSize := len(batch) >= nd.config.BatchSize

		if shouldFlushTime || shouldFlushSize {
			nd.processBatch(channelName, batch)
			delete(nd.batches, channelName)
		}
	}

	if shouldFlushTime {
		nd.lastFlush = time.Now()
	}
}

// flushAllBatches flushes all pending batches
func (nd *NotificationDispatcher) flushAllBatches() {
	nd.batchMutex.Lock()
	defer nd.batchMutex.Unlock()

	for channelName, batch := range nd.batches {
		if len(batch) > 0 {
			nd.processBatch(channelName, batch)
		}
	}

	nd.batches = make(map[string][]*Notification)
}

// processBatch processes a batch of notifications for a specific channel
func (nd *NotificationDispatcher) processBatch(channelName string, notifications []*Notification) {
	if len(notifications) == 0 {
		return
	}

	nd.logger.Debug("Processing notification batch",
		"channel", channelName,
		"count", len(notifications))

	// Find an available worker
	for _, worker := range nd.workers {
		if !worker.IsActive() {
			worker.ProcessBatch(channelName, notifications)
			break
		}
	}
}

// updateMetrics safely updates dispatcher metrics
func (nd *NotificationDispatcher) updateMetrics(updateFunc func(*DispatcherMetrics)) {
	nd.metrics.mutex.Lock()
	defer nd.metrics.mutex.Unlock()
	updateFunc(nd.metrics)
}
