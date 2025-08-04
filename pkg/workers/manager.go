package workers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"easy-orders-backend/pkg/logger"
)

// PoolManager manages multiple worker pools for different job types
type PoolManager struct {
	pools  map[string]*WorkerPool
	logger *logger.Logger
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

// NewPoolManager creates a new pool manager
func NewPoolManager(logger *logger.Logger) *PoolManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &PoolManager{
		pools:  make(map[string]*WorkerPool),
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}
}

// CreatePool creates a new worker pool with the given configuration
func (pm *PoolManager) CreatePool(config *WorkerPoolConfig) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.pools[config.Name]; exists {
		return fmt.Errorf("pool %s already exists", config.Name)
	}

	pool := NewWorkerPool(config, pm.logger)
	pm.pools[config.Name] = pool

	pm.logger.Info("Worker pool created", "name", config.Name)
	return nil
}

// StartPool starts a specific worker pool
func (pm *PoolManager) StartPool(name string) error {
	pm.mu.RLock()
	pool, exists := pm.pools[name]
	pm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("pool %s not found", name)
	}

	return pool.Start(pm.ctx)
}

// StopPool stops a specific worker pool
func (pm *PoolManager) StopPool(name string) error {
	pm.mu.RLock()
	pool, exists := pm.pools[name]
	pm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("pool %s not found", name)
	}

	return pool.Stop()
}

// StartAllPools starts all worker pools
func (pm *PoolManager) StartAllPools() error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for name, pool := range pm.pools {
		if err := pool.Start(pm.ctx); err != nil {
			pm.logger.Error("Failed to start pool", "name", name, "error", err)
			return err
		}
	}

	pm.logger.Info("All worker pools started", "count", len(pm.pools))
	return nil
}

// StopAllPools stops all worker pools
func (pm *PoolManager) StopAllPools() error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var errors []error
	for name, pool := range pm.pools {
		if err := pool.Stop(); err != nil {
			pm.logger.Error("Failed to stop pool", "name", name, "error", err)
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to stop %d pools", len(errors))
	}

	pm.logger.Info("All worker pools stopped", "count", len(pm.pools))
	return nil
}

// SubmitJob submits a job to the appropriate worker pool based on job type
func (pm *PoolManager) SubmitJob(job Job) error {
	poolName := pm.getPoolNameForJobType(job.GetType())

	pm.mu.RLock()
	pool, exists := pm.pools[poolName]
	pm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no pool found for job type %s (expected pool: %s)", job.GetType(), poolName)
	}

	return pool.SubmitJob(job)
}

// SubmitJobToPool submits a job to a specific pool
func (pm *PoolManager) SubmitJobToPool(poolName string, job Job) error {
	pm.mu.RLock()
	pool, exists := pm.pools[poolName]
	pm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("pool %s not found", poolName)
	}

	return pool.SubmitJob(job)
}

// GetPoolMetrics returns metrics for a specific pool
func (pm *PoolManager) GetPoolMetrics(poolName string) (*PoolMetrics, error) {
	pm.mu.RLock()
	pool, exists := pm.pools[poolName]
	pm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("pool %s not found", poolName)
	}

	return pool.GetMetrics(), nil
}

// GetAllPoolMetrics returns metrics for all pools
func (pm *PoolManager) GetAllPoolMetrics() map[string]*PoolMetrics {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	metrics := make(map[string]*PoolMetrics)
	for name, pool := range pm.pools {
		metrics[name] = pool.GetMetrics()
	}

	return metrics
}

// GetPoolNames returns the names of all pools
func (pm *PoolManager) GetPoolNames() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	names := make([]string, 0, len(pm.pools))
	for name := range pm.pools {
		names = append(names, name)
	}

	return names
}

// Shutdown gracefully shuts down the pool manager and all pools
func (pm *PoolManager) Shutdown() error {
	pm.logger.Info("Shutting down pool manager")

	// Cancel context to signal all pools to stop
	pm.cancel()

	// Stop all pools
	if err := pm.StopAllPools(); err != nil {
		return err
	}

	pm.logger.Info("Pool manager shutdown complete")
	return nil
}

// getPoolNameForJobType maps job types to pool names
func (pm *PoolManager) getPoolNameForJobType(jobType string) string {
	switch jobType {
	case JobTypeReportGeneration:
		return "reports"
	case JobTypeNotification:
		return "notifications"
	case JobTypeAuditProcessing:
		return "audit"
	case JobTypeBulkProcessing:
		return "bulk"
	case JobTypeExternalIntegration:
		return "external"
	case JobTypeCacheWarming:
		return "cache"
	case JobTypeCleanup:
		return "cleanup"
	case JobTypeDataExport:
		return "export"
	default:
		return "general"
	}
}

// InitializeDefaultPools creates standard worker pools for common job types
func (pm *PoolManager) InitializeDefaultPools() error {
	pools := []*WorkerPoolConfig{
		{
			Name:            "reports",
			WorkerCount:     3,
			QueueSize:       50,
			ShutdownTimeout: 60 * time.Second, // Reports can take longer
			RetryDelay:      2 * time.Second,
			EnableMetrics:   true,
		},
		{
			Name:            "notifications",
			WorkerCount:     10,
			QueueSize:       200,
			ShutdownTimeout: 30 * time.Second,
			RetryDelay:      1 * time.Second,
			EnableMetrics:   true,
		},
		{
			Name:            "audit",
			WorkerCount:     5,
			QueueSize:       100,
			ShutdownTimeout: 30 * time.Second,
			RetryDelay:      1 * time.Second,
			EnableMetrics:   true,
		},
		{
			Name:            "bulk",
			WorkerCount:     2,
			QueueSize:       20,
			ShutdownTimeout: 120 * time.Second, // Bulk operations can take very long
			RetryDelay:      5 * time.Second,
			EnableMetrics:   true,
		},
		{
			Name:            "external",
			WorkerCount:     8,
			QueueSize:       100,
			ShutdownTimeout: 45 * time.Second,
			RetryDelay:      3 * time.Second,
			EnableMetrics:   true,
		},
		{
			Name:            "cache",
			WorkerCount:     2,
			QueueSize:       30,
			ShutdownTimeout: 30 * time.Second,
			RetryDelay:      1 * time.Second,
			EnableMetrics:   true,
		},
		{
			Name:            "cleanup",
			WorkerCount:     1,
			QueueSize:       10,
			ShutdownTimeout: 60 * time.Second,
			RetryDelay:      5 * time.Second,
			EnableMetrics:   true,
		},
		{
			Name:            "export",
			WorkerCount:     2,
			QueueSize:       20,
			ShutdownTimeout: 90 * time.Second,
			RetryDelay:      3 * time.Second,
			EnableMetrics:   true,
		},
		{
			Name:            "general",
			WorkerCount:     5,
			QueueSize:       100,
			ShutdownTimeout: 30 * time.Second,
			RetryDelay:      2 * time.Second,
			EnableMetrics:   true,
		},
	}

	for _, config := range pools {
		if err := pm.CreatePool(config); err != nil {
			return err
		}
	}

	pm.logger.Info("Default worker pools initialized", "count", len(pools))
	return nil
}
