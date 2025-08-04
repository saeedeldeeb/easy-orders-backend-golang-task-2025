package concurrency

import (
	"context"
	"fmt"
	"time"

	"easy-orders-backend/pkg/logger"
)

// DistributedLock interface for distributed locking mechanisms
type DistributedLock interface {
	// Acquire attempts to acquire a lock with the given key and TTL
	Acquire(ctx context.Context, key string, ttl time.Duration) (bool, error)

	// Release releases the lock with the given key
	Release(ctx context.Context, key string) error

	// Extend extends the TTL of an existing lock
	Extend(ctx context.Context, key string, ttl time.Duration) error

	// IsLocked checks if a key is currently locked
	IsLocked(ctx context.Context, key string) (bool, error)
}

// LockConfig defines locking behavior
type LockConfig struct {
	AcquireTimeout time.Duration `json:"acquire_timeout"`
	LockTTL        time.Duration `json:"lock_ttl"`
	ExtendInterval time.Duration `json:"extend_interval"`
	RetryInterval  time.Duration `json:"retry_interval"`
}

// DefaultLockConfig returns default lock configuration
func DefaultLockConfig() *LockConfig {
	return &LockConfig{
		AcquireTimeout: 5 * time.Second,
		LockTTL:        30 * time.Second,
		ExtendInterval: 10 * time.Second,
		RetryInterval:  100 * time.Millisecond,
	}
}

// LockManager manages distributed locks for inventory operations
type LockManager struct {
	lock   DistributedLock
	config *LockConfig
	logger *logger.Logger
}

// NewLockManager creates a new lock manager
func NewLockManager(lock DistributedLock, config *LockConfig, logger *logger.Logger) *LockManager {
	if config == nil {
		config = DefaultLockConfig()
	}

	return &LockManager{
		lock:   lock,
		config: config,
		logger: logger,
	}
}

// WithInventoryLock executes a function while holding an inventory lock
func (lm *LockManager) WithInventoryLock(ctx context.Context, productID string, operation func() error) error {
	lockKey := fmt.Sprintf("inventory:lock:%s", productID)

	// Try to acquire the lock
	acquired, err := lm.acquireLockWithRetry(ctx, lockKey)
	if err != nil {
		return fmt.Errorf("failed to acquire inventory lock for product %s: %w", productID, err)
	}
	if !acquired {
		return fmt.Errorf("failed to acquire inventory lock for product %s: timeout", productID)
	}

	lm.logger.Debug("Acquired inventory lock", "product_id", productID, "lock_key", lockKey)

	// Setup lock extension if needed
	extendCtx, cancelExtend := context.WithCancel(ctx)
	defer cancelExtend()

	go lm.extendLockPeriodically(extendCtx, lockKey)

	// Execute the operation
	defer func() {
		if releaseErr := lm.lock.Release(ctx, lockKey); releaseErr != nil {
			lm.logger.Error("Failed to release inventory lock",
				"error", releaseErr,
				"product_id", productID,
				"lock_key", lockKey)
		} else {
			lm.logger.Debug("Released inventory lock", "product_id", productID, "lock_key", lockKey)
		}
	}()

	return operation()
}

// WithBulkInventoryLock executes a function while holding multiple inventory locks
func (lm *LockManager) WithBulkInventoryLock(ctx context.Context, productIDs []string, operation func() error) error {
	if len(productIDs) == 0 {
		return operation()
	}

	// Create sorted list of lock keys to prevent deadlocks
	lockKeys := make([]string, len(productIDs))
	for i, productID := range productIDs {
		lockKeys[i] = fmt.Sprintf("inventory:lock:%s", productID)
	}

	// Sort lock keys to ensure consistent ordering and prevent deadlocks
	for i := 0; i < len(lockKeys)-1; i++ {
		for j := i + 1; j < len(lockKeys); j++ {
			if lockKeys[i] > lockKeys[j] {
				lockKeys[i], lockKeys[j] = lockKeys[j], lockKeys[i]
			}
		}
	}

	// Acquire all locks in order
	acquiredLocks := make([]string, 0, len(lockKeys))
	defer func() {
		// Release all acquired locks in reverse order
		for i := len(acquiredLocks) - 1; i >= 0; i-- {
			if releaseErr := lm.lock.Release(ctx, acquiredLocks[i]); releaseErr != nil {
				lm.logger.Error("Failed to release bulk inventory lock",
					"error", releaseErr,
					"lock_key", acquiredLocks[i])
			}
		}
	}()

	for _, lockKey := range lockKeys {
		acquired, err := lm.acquireLockWithRetry(ctx, lockKey)
		if err != nil {
			return fmt.Errorf("failed to acquire bulk inventory lock %s: %w", lockKey, err)
		}
		if !acquired {
			return fmt.Errorf("failed to acquire bulk inventory lock %s: timeout", lockKey)
		}
		acquiredLocks = append(acquiredLocks, lockKey)
	}

	lm.logger.Debug("Acquired bulk inventory locks", "lock_count", len(acquiredLocks))

	// Setup lock extension for all locks
	extendCtx, cancelExtend := context.WithCancel(ctx)
	defer cancelExtend()

	for _, lockKey := range acquiredLocks {
		go lm.extendLockPeriodically(extendCtx, lockKey)
	}

	return operation()
}

// acquireLockWithRetry attempts to acquire a lock with retry logic
func (lm *LockManager) acquireLockWithRetry(ctx context.Context, lockKey string) (bool, error) {
	deadline := time.Now().Add(lm.config.AcquireTimeout)

	for time.Now().Before(deadline) {
		acquired, err := lm.lock.Acquire(ctx, lockKey, lm.config.LockTTL)
		if err != nil {
			return false, err
		}
		if acquired {
			return true, nil
		}

		// Wait before retrying
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(lm.config.RetryInterval):
			// Continue to next attempt
		}
	}

	return false, nil
}

// extendLockPeriodically extends a lock periodically to prevent expiration
func (lm *LockManager) extendLockPeriodically(ctx context.Context, lockKey string) {
	ticker := time.NewTicker(lm.config.ExtendInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := lm.lock.Extend(ctx, lockKey, lm.config.LockTTL); err != nil {
				lm.logger.Warn("Failed to extend lock", "error", err, "lock_key", lockKey)
				return
			}
			lm.logger.Debug("Extended lock TTL", "lock_key", lockKey, "ttl", lm.config.LockTTL)
		}
	}
}
