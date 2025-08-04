package concurrency

import (
	"context"
	"fmt"
	"time"

	"easy-orders-backend/pkg/logger"
)

// RedisLock implements DistributedLock using Redis
type RedisLock struct {
	logger *logger.Logger
	// TODO: Add Redis client when Redis is integrated
	// For now, we'll implement an in-memory version for testing
	locks map[string]lockInfo
}

type lockInfo struct {
	acquired time.Time
	ttl      time.Duration
	extended time.Time
}

// NewRedisLock creates a new Redis-based distributed lock
func NewRedisLock(logger *logger.Logger) *RedisLock {
	return &RedisLock{
		logger: logger,
		locks:  make(map[string]lockInfo),
	}
}

// Acquire attempts to acquire a lock with the given key and TTL
func (r *RedisLock) Acquire(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	r.logger.Debug("Attempting to acquire lock", "key", key, "ttl", ttl)

	// Check if lock already exists and is still valid
	if info, exists := r.locks[key]; exists {
		lastActivity := info.extended
		if lastActivity.IsZero() {
			lastActivity = info.acquired
		}

		if time.Since(lastActivity) < info.ttl {
			r.logger.Debug("Lock already held by another process", "key", key)
			return false, nil
		}

		// Lock has expired, remove it
		delete(r.locks, key)
		r.logger.Debug("Removed expired lock", "key", key)
	}

	// Acquire the lock
	r.locks[key] = lockInfo{
		acquired: time.Now(),
		ttl:      ttl,
	}

	r.logger.Debug("Lock acquired successfully", "key", key, "ttl", ttl)
	return true, nil
}

// Release releases the lock with the given key
func (r *RedisLock) Release(ctx context.Context, key string) error {
	r.logger.Debug("Releasing lock", "key", key)

	if _, exists := r.locks[key]; !exists {
		r.logger.Warn("Attempted to release non-existent lock", "key", key)
		return fmt.Errorf("lock %s does not exist", key)
	}

	delete(r.locks, key)
	r.logger.Debug("Lock released successfully", "key", key)
	return nil
}

// Extend extends the TTL of an existing lock
func (r *RedisLock) Extend(ctx context.Context, key string, ttl time.Duration) error {
	r.logger.Debug("Extending lock TTL", "key", key, "ttl", ttl)

	info, exists := r.locks[key]
	if !exists {
		r.logger.Warn("Attempted to extend non-existent lock", "key", key)
		return fmt.Errorf("lock %s does not exist", key)
	}

	// Update the lock info
	info.ttl = ttl
	info.extended = time.Now()
	r.locks[key] = info

	r.logger.Debug("Lock TTL extended successfully", "key", key, "new_ttl", ttl)
	return nil
}

// IsLocked checks if a key is currently locked
func (r *RedisLock) IsLocked(ctx context.Context, key string) (bool, error) {
	info, exists := r.locks[key]
	if !exists {
		return false, nil
	}

	lastActivity := info.extended
	if lastActivity.IsZero() {
		lastActivity = info.acquired
	}

	isLocked := time.Since(lastActivity) < info.ttl
	return isLocked, nil
}

// cleanup removes expired locks (for maintenance)
func (r *RedisLock) cleanup() {
	now := time.Now()
	for key, info := range r.locks {
		lastActivity := info.extended
		if lastActivity.IsZero() {
			lastActivity = info.acquired
		}

		if now.Sub(lastActivity) >= info.ttl {
			delete(r.locks, key)
			r.logger.Debug("Cleaned up expired lock", "key", key)
		}
	}
}
