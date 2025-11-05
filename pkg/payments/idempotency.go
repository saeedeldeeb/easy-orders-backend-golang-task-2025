package payments

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"easy-orders-backend/pkg/logger"
)

// IdempotencyManager handles idempotency for payment operations
type IdempotencyManager struct {
	cache      map[string]*IdempotencyRecord
	cacheMutex sync.RWMutex
	ttl        time.Duration
	logger     *logger.Logger

	// Cleanup routine
	stopCleanup chan struct{}
	cleanupWG   sync.WaitGroup
}

// IdempotencyRecord stores information about a previous operation
type IdempotencyRecord struct {
	Key            string         `json:"key"`
	PaymentID      string         `json:"payment_id"`
	RequestHash    string         `json:"request_hash"`
	Status         string         `json:"status"`
	Result         *PaymentResult `json:"result,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	LastAccessedAt time.Time      `json:"last_accessed_at"`
	ExpiresAt      time.Time      `json:"expires_at"`
}

// NewIdempotencyManager creates a new idempotency manager
func NewIdempotencyManager(ttl time.Duration, logger *logger.Logger) *IdempotencyManager {
	if ttl == 0 {
		ttl = 24 * time.Hour // Default 24 hours
	}

	manager := &IdempotencyManager{
		cache:       make(map[string]*IdempotencyRecord),
		ttl:         ttl,
		logger:      logger,
		stopCleanup: make(chan struct{}),
	}

	// Start a cleanup routine
	manager.startCleanup()

	return manager
}

// Stop gracefully stops the idempotency manager
func (im *IdempotencyManager) Stop() {
	close(im.stopCleanup)
	im.cleanupWG.Wait()
}

// GenerateRequestHash creates a hash of the payment request for comparison
func (im *IdempotencyManager) GenerateRequestHash(req *PaymentRequest) string {
	// Create a canonical representation of the request
	canonical := fmt.Sprintf("%s|%.2f|%s|%s|%s|%s",
		req.OrderID,
		req.Amount,
		req.Currency,
		req.PaymentMethod,
		req.Gateway,
		req.IdempotencyKey,
	)

	hash := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(hash[:])
}

// CheckIdempotency checks if a request has been processed before
func (im *IdempotencyManager) CheckIdempotency(ctx context.Context, req *PaymentRequest) (*IdempotencyRecord, bool) {
	im.cacheMutex.RLock()
	defer im.cacheMutex.RUnlock()

	record, exists := im.cache[req.IdempotencyKey]
	if !exists {
		return nil, false
	}

	// Check if the record has expired
	if time.Now().After(record.ExpiresAt) {
		im.logger.Debug("Idempotency record expired",
			"key", req.IdempotencyKey,
			"expired_at", record.ExpiresAt)
		return nil, false
	}

	// Update last accessed time
	record.LastAccessedAt = time.Now()

	// Verify request hash matches to prevent key reuse with different parameters
	requestHash := im.GenerateRequestHash(req)
	if record.RequestHash != requestHash {
		im.logger.Warn("Idempotency key reused with different parameters",
			"key", req.IdempotencyKey,
			"original_hash", record.RequestHash,
			"current_hash", requestHash)
		return nil, false
	}

	im.logger.Debug("Idempotency record found",
		"key", req.IdempotencyKey,
		"payment_id", record.PaymentID,
		"status", record.Status)

	return record, true
}

// StoreIdempotencyRecord stores a new idempotency record
func (im *IdempotencyManager) StoreIdempotencyRecord(ctx context.Context, req *PaymentRequest, paymentID string, status string) *IdempotencyRecord {
	im.cacheMutex.Lock()
	defer im.cacheMutex.Unlock()

	now := time.Now()
	record := &IdempotencyRecord{
		Key:            req.IdempotencyKey,
		PaymentID:      paymentID,
		RequestHash:    im.GenerateRequestHash(req),
		Status:         status,
		CreatedAt:      now,
		LastAccessedAt: now,
		ExpiresAt:      now.Add(im.ttl),
	}

	im.cache[req.IdempotencyKey] = record

	im.logger.Debug("Idempotency record stored",
		"key", req.IdempotencyKey,
		"payment_id", paymentID,
		"status", status,
		"expires_at", record.ExpiresAt)

	return record
}

// UpdateIdempotencyRecord updates an existing idempotency record
func (im *IdempotencyManager) UpdateIdempotencyRecord(ctx context.Context, key string, status string, result *PaymentResult) {
	im.cacheMutex.Lock()
	defer im.cacheMutex.Unlock()

	record, exists := im.cache[key]
	if !exists {
		im.logger.Warn("Attempted to update non-existent idempotency record", "key", key)
		return
	}

	record.Status = status
	record.Result = result
	record.LastAccessedAt = time.Now()

	im.logger.Debug("Idempotency record updated",
		"key", key,
		"status", status,
		"success", result != nil && result.Success)
}

// RemoveIdempotencyRecord removes an idempotency record
func (im *IdempotencyManager) RemoveIdempotencyRecord(ctx context.Context, key string) {
	im.cacheMutex.Lock()
	defer im.cacheMutex.Unlock()

	delete(im.cache, key)

	im.logger.Debug("Idempotency record removed", "key", key)
}

// GetStats returns statistics about the idempotency cache
func (im *IdempotencyManager) GetStats() map[string]interface{} {
	im.cacheMutex.RLock()
	defer im.cacheMutex.RUnlock()

	now := time.Now()
	activeRecords := 0
	expiredRecords := 0

	for _, record := range im.cache {
		if now.After(record.ExpiresAt) {
			expiredRecords++
		} else {
			activeRecords++
		}
	}

	return map[string]interface{}{
		"total_records":   len(im.cache),
		"active_records":  activeRecords,
		"expired_records": expiredRecords,
		"ttl_hours":       im.ttl.Hours(),
	}
}

// startCleanup starts the background cleanup routine
func (im *IdempotencyManager) startCleanup() {
	im.cleanupWG.Add(1)
	go func() {
		defer im.cleanupWG.Done()

		ticker := time.NewTicker(time.Hour) // Cleanup every hour
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				im.cleanupExpiredRecords()
			case <-im.stopCleanup:
				return
			}
		}
	}()
}

// cleanupExpiredRecords removes expired records from the cache
func (im *IdempotencyManager) cleanupExpiredRecords() {
	im.cacheMutex.Lock()
	defer im.cacheMutex.Unlock()

	now := time.Now()
	var expiredKeys []string

	for key, record := range im.cache {
		if now.After(record.ExpiresAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, key := range expiredKeys {
		delete(im.cache, key)
	}

	if len(expiredKeys) > 0 {
		im.logger.Info("Cleaned up expired idempotency records",
			"removed_count", len(expiredKeys),
			"remaining_count", len(im.cache))
	}
}

// ForceCleanup immediately cleans up expired records
func (im *IdempotencyManager) ForceCleanup() {
	im.cleanupExpiredRecords()
}
