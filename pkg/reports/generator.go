package reports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"easy-orders-backend/pkg/logger"
)

// ReportGenerator defines the interface for generating reports
type ReportGenerator interface {
	// GenerateReport generates a report of the specified type
	GenerateReport(ctx context.Context, req *ReportRequest) (*ReportResult, error)

	// GetSupportedTypes returns the report types this generator supports
	GetSupportedTypes() []ReportType

	// GetName returns the generator name
	GetName() string

	// EstimateGenerationTime estimates how long the report will take to generate
	EstimateGenerationTime(req *ReportRequest) time.Duration
}

// ReportManager manages concurrent report generation with caching and scheduling
type ReportManager struct {
	generators    map[ReportType]ReportGenerator
	cache         map[string]*ReportCache
	cacheMutex    sync.RWMutex
	generatorPool chan struct{} // Semaphore for concurrent generation limit
	metrics       *ReportMetrics
	metricsMutex  sync.RWMutex
	logger        *logger.Logger

	// Configuration
	maxConcurrentReports int
	defaultCacheTTL      time.Duration
	maxCacheSize         int
}

// ReportManagerConfig configures the report manager
type ReportManagerConfig struct {
	MaxConcurrentReports int           `json:"max_concurrent_reports"`
	DefaultCacheTTL      time.Duration `json:"default_cache_ttl"`
	MaxCacheSize         int           `json:"max_cache_size"`
	CleanupInterval      time.Duration `json:"cleanup_interval"`
}

// DefaultReportManagerConfig returns default configuration
func DefaultReportManagerConfig() *ReportManagerConfig {
	return &ReportManagerConfig{
		MaxConcurrentReports: 10,
		DefaultCacheTTL:      30 * time.Minute,
		MaxCacheSize:         1000,
		CleanupInterval:      time.Hour,
	}
}

// NewReportManager creates a new report manager
func NewReportManager(config *ReportManagerConfig, logger *logger.Logger) *ReportManager {
	if config == nil {
		config = DefaultReportManagerConfig()
	}

	rm := &ReportManager{
		generators:           make(map[ReportType]ReportGenerator),
		cache:                make(map[string]*ReportCache),
		generatorPool:        make(chan struct{}, config.MaxConcurrentReports),
		metrics:              &ReportMetrics{},
		logger:               logger,
		maxConcurrentReports: config.MaxConcurrentReports,
		defaultCacheTTL:      config.DefaultCacheTTL,
		maxCacheSize:         config.MaxCacheSize,
	}

	// Start cache cleanup routine
	go rm.cacheCleanupRoutine(config.CleanupInterval)

	return rm
}

// RegisterGenerator registers a report generator for specific report types
func (rm *ReportManager) RegisterGenerator(generator ReportGenerator) {
	supportedTypes := generator.GetSupportedTypes()
	for _, reportType := range supportedTypes {
		rm.generators[reportType] = generator
		rm.logger.Info("Report generator registered",
			"generator", generator.GetName(),
			"type", string(reportType))
	}
}

// GenerateReportAsync generates a report asynchronously
func (rm *ReportManager) GenerateReportAsync(ctx context.Context, req *ReportRequest) (*ReportResult, error) {
	rm.logger.Info("Starting async report generation",
		"id", req.ID,
		"type", string(req.Type),
		"format", string(req.Format),
		"priority", req.Priority)

	// Check cache first
	if cachedResult := rm.checkCache(req); cachedResult != nil {
		rm.logger.Debug("Report served from cache", "id", req.ID, "cache_key", rm.generateCacheKey(req))
		rm.updateMetrics(func(m *ReportMetrics) {
			// Update cache hit rate
			totalRequests := m.TotalReports + 1
			hitRate := (m.CacheHitRate*float64(m.TotalReports) + 1.0) / float64(totalRequests)
			m.CacheHitRate = hitRate
		})
		return cachedResult, nil
	}

	// Create initial result
	result := &ReportResult{
		ID:        fmt.Sprintf("result_%s_%d", req.ID, time.Now().UnixNano()),
		RequestID: req.ID,
		Type:      req.Type,
		Format:    req.Format,
		Status:    ReportStatusPending,
		Metadata:  make(map[string]interface{}),
	}

	// Update metrics
	rm.updateMetrics(func(m *ReportMetrics) {
		m.TotalReports++
		m.PendingReports++
		m.QueueDepth++
	})

	// Generate report in goroutine
	go rm.generateReportConcurrent(ctx, req, result)

	return result, nil
}

// GenerateReportSync generates a report synchronously
func (rm *ReportManager) GenerateReportSync(ctx context.Context, req *ReportRequest) (*ReportResult, error) {
	rm.logger.Info("Starting sync report generation",
		"id", req.ID,
		"type", string(req.Type))

	// Check cache first
	if cachedResult := rm.checkCache(req); cachedResult != nil {
		rm.logger.Debug("Report served from cache", "id", req.ID)
		return cachedResult, nil
	}

	// Create result
	result := &ReportResult{
		ID:        fmt.Sprintf("result_%s_%d", req.ID, time.Now().UnixNano()),
		RequestID: req.ID,
		Type:      req.Type,
		Format:    req.Format,
		Status:    ReportStatusGenerating,
		Metadata:  make(map[string]interface{}),
	}

	// Update metrics
	rm.updateMetrics(func(m *ReportMetrics) {
		m.TotalReports++
		m.ActiveGenerators++
	})

	// Generate report synchronously
	startTime := time.Now()
	err := rm.generateReport(ctx, req, result)
	duration := time.Since(startTime)

	// Update metrics
	rm.updateMetrics(func(m *ReportMetrics) {
		m.ActiveGenerators--
		m.TotalGenTime += duration
		if m.TotalReports > 0 {
			m.AverageGenTime = m.TotalGenTime / time.Duration(m.TotalReports)
		}

		if err != nil {
			m.FailedReports++
		} else {
			m.CompletedReports++
		}
	})

	if err != nil {
		result.Status = ReportStatusFailed
		result.Error = err.Error()
		rm.logger.Error("Sync report generation failed", "id", req.ID, "error", err)
		return result, err
	}

	// Cache the result
	rm.cacheResult(req, result)

	rm.logger.Info("Sync report generation completed",
		"id", req.ID,
		"duration_ms", duration.Milliseconds())

	return result, nil
}

// generateReportConcurrent generates a report with concurrency control
func (rm *ReportManager) generateReportConcurrent(ctx context.Context, req *ReportRequest, result *ReportResult) {
	// Acquire semaphore slot
	select {
	case rm.generatorPool <- struct{}{}:
		defer func() { <-rm.generatorPool }()
	case <-ctx.Done():
		result.Status = ReportStatusCancelled
		result.Error = "Context cancelled while waiting for generator slot"
		return
	}

	// Update status and metrics
	result.Status = ReportStatusGenerating
	rm.updateMetrics(func(m *ReportMetrics) {
		m.PendingReports--
		m.ActiveGenerators++
		m.QueueDepth--
	})

	startTime := time.Now()
	err := rm.generateReport(ctx, req, result)
	duration := time.Since(startTime)
	result.ProcessingTime = duration

	// Update metrics
	rm.updateMetrics(func(m *ReportMetrics) {
		m.ActiveGenerators--
		m.TotalGenTime += duration
		if m.TotalReports > 0 {
			m.AverageGenTime = m.TotalGenTime / time.Duration(m.TotalReports)
		}

		if err != nil {
			m.FailedReports++
		} else {
			m.CompletedReports++
		}
	})

	if err != nil {
		result.Status = ReportStatusFailed
		result.Error = err.Error()
		rm.logger.Error("Async report generation failed", "id", req.ID, "error", err)
		return
	}

	// Cache the result
	rm.cacheResult(req, result)

	rm.logger.Info("Async report generation completed",
		"id", req.ID,
		"duration_ms", duration.Milliseconds())
}

// generateReport performs the actual report generation
func (rm *ReportManager) generateReport(ctx context.Context, req *ReportRequest, result *ReportResult) error {
	// Find appropriate generator
	generator, exists := rm.generators[req.Type]
	if !exists {
		return fmt.Errorf("no generator found for report type: %s", req.Type)
	}

	rm.logger.Debug("Generating report with generator",
		"generator", generator.GetName(),
		"type", string(req.Type),
		"id", req.ID)

	// Generate the report
	generatedResult, err := generator.GenerateReport(ctx, req)
	if err != nil {
		return fmt.Errorf("generator failed: %w", err)
	}

	// Copy result data
	result.Data = generatedResult.Data
	result.FilePath = generatedResult.FilePath
	result.FileSize = generatedResult.FileSize
	result.RowCount = generatedResult.RowCount
	result.Status = ReportStatusCompleted
	now := time.Now()
	result.GeneratedAt = &now

	// Set expiration time
	if req.ExpiresAt != nil {
		result.ExpiresAt = req.ExpiresAt
	} else {
		// Default expiration: 24 hours for data reports, 7 days for static reports
		var expiration time.Time
		if req.Type == ReportTypeLowStock || req.Type == ReportTypeInventoryValue {
			expiration = now.Add(4 * time.Hour) // Inventory data changes frequently
		} else {
			expiration = now.Add(24 * time.Hour) // Sales data is more stable
		}
		result.ExpiresAt = &expiration
	}

	// Add metadata
	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}
	result.Metadata["generator"] = generator.GetName()
	result.Metadata["cache_key"] = rm.generateCacheKey(req)
	result.Metadata["estimated_time"] = generator.EstimateGenerationTime(req).String()

	return nil
}

// checkCache checks if a cached result exists for the request
func (rm *ReportManager) checkCache(req *ReportRequest) *ReportResult {
	cacheKey := rm.generateCacheKey(req)

	rm.cacheMutex.RLock()
	cachedReport, exists := rm.cache[cacheKey]
	rm.cacheMutex.RUnlock()

	if !exists {
		return nil
	}

	// Check if cache is expired
	if cachedReport.IsExpired() {
		rm.logger.Debug("Cached report expired", "cache_key", cacheKey)
		rm.removeCacheEntry(cacheKey)
		return nil
	}

	// Update cache access statistics
	rm.cacheMutex.Lock()
	cachedReport.HitCount++
	cachedReport.LastAccessed = time.Now()
	rm.cacheMutex.Unlock()

	// Convert cached data to result format
	result := &ReportResult{
		ID:             fmt.Sprintf("cached_%s_%d", req.ID, time.Now().UnixNano()),
		RequestID:      req.ID,
		Type:           req.Type,
		Format:         req.Format,
		Status:         ReportStatusCompleted,
		Data:           cachedReport.Data,
		GeneratedAt:    &cachedReport.GeneratedAt,
		ExpiresAt:      &cachedReport.ExpiresAt,
		ProcessingTime: 0, // Cached result
		Metadata: map[string]interface{}{
			"cached":        true,
			"cache_key":     cacheKey,
			"hit_count":     cachedReport.HitCount,
			"last_accessed": cachedReport.LastAccessed,
		},
	}

	return result
}

// cacheResult stores a generated report result in cache
func (rm *ReportManager) cacheResult(req *ReportRequest, result *ReportResult) {
	if result.Status != ReportStatusCompleted || result.Data == nil {
		return
	}

	cacheKey := rm.generateCacheKey(req)
	expiresAt := time.Now().Add(rm.defaultCacheTTL)
	if result.ExpiresAt != nil {
		expiresAt = *result.ExpiresAt
	}

	cache := &ReportCache{
		Key:          cacheKey,
		ReportType:   req.Type,
		Parameters:   req.Parameters,
		Data:         result.Data,
		GeneratedAt:  time.Now(),
		ExpiresAt:    expiresAt,
		HitCount:     0,
		LastAccessed: time.Now(),
	}

	rm.cacheMutex.Lock()
	defer rm.cacheMutex.Unlock()

	// Check cache size limit
	if len(rm.cache) >= rm.maxCacheSize {
		rm.evictOldestCache()
	}

	rm.cache[cacheKey] = cache

	rm.logger.Debug("Report cached",
		"cache_key", cacheKey,
		"type", string(req.Type),
		"expires_at", expiresAt)
}

// generateCacheKey generates a unique cache key for a report request
func (rm *ReportManager) generateCacheKey(req *ReportRequest) string {
	// Create a canonical representation of the request
	keyData := map[string]interface{}{
		"type":       string(req.Type),
		"format":     string(req.Format),
		"parameters": req.Parameters,
	}

	// Serialize to JSON for consistent hashing
	keyBytes, _ := json.Marshal(keyData)
	hash := sha256.Sum256(keyBytes)
	return hex.EncodeToString(hash[:])
}

// removeCacheEntry removes a cache entry
func (rm *ReportManager) removeCacheEntry(key string) {
	rm.cacheMutex.Lock()
	defer rm.cacheMutex.Unlock()
	delete(rm.cache, key)
}

// evictOldestCache removes the oldest cache entry
func (rm *ReportManager) evictOldestCache() {
	var oldestKey string
	var oldestTime time.Time

	for key, cache := range rm.cache {
		if oldestKey == "" || cache.LastAccessed.Before(oldestTime) {
			oldestKey = key
			oldestTime = cache.LastAccessed
		}
	}

	if oldestKey != "" {
		delete(rm.cache, oldestKey)
		rm.logger.Debug("Evicted oldest cache entry", "cache_key", oldestKey)
	}
}

// cacheCleanupRoutine periodically cleans up expired cache entries
func (rm *ReportManager) cacheCleanupRoutine(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		rm.cleanupExpiredCache()
	}
}

// cleanupExpiredCache removes expired cache entries
func (rm *ReportManager) cleanupExpiredCache() {
	rm.cacheMutex.Lock()
	defer rm.cacheMutex.Unlock()

	var expiredKeys []string
	now := time.Now()

	for key, cache := range rm.cache {
		if now.After(cache.ExpiresAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, key := range expiredKeys {
		delete(rm.cache, key)
	}

	if len(expiredKeys) > 0 {
		rm.logger.Info("Cleaned up expired cache entries",
			"removed_count", len(expiredKeys),
			"remaining_count", len(rm.cache))
	}
}

// GetMetrics returns current report generation metrics
func (rm *ReportManager) GetMetrics() *ReportMetrics {
	rm.metricsMutex.RLock()
	defer rm.metricsMutex.RUnlock()

	// Update queue depth
	metrics := *rm.metrics
	metrics.QueueDepth = len(rm.generatorPool)

	return &metrics
}

// updateMetrics safely updates metrics
func (rm *ReportManager) updateMetrics(updateFunc func(*ReportMetrics)) {
	rm.metricsMutex.Lock()
	defer rm.metricsMutex.Unlock()
	updateFunc(rm.metrics)
}

// GetCacheStats returns cache statistics
func (rm *ReportManager) GetCacheStats() map[string]interface{} {
	rm.cacheMutex.RLock()
	defer rm.cacheMutex.RUnlock()

	totalHits := 0
	expiredCount := 0
	now := time.Now()

	for _, cache := range rm.cache {
		totalHits += cache.HitCount
		if now.After(cache.ExpiresAt) {
			expiredCount++
		}
	}

	return map[string]interface{}{
		"total_entries":   len(rm.cache),
		"expired_entries": expiredCount,
		"total_hits":      totalHits,
		"max_size":        rm.maxCacheSize,
		"default_ttl":     rm.defaultCacheTTL.String(),
	}
}

// FlushCache clears all cached reports
func (rm *ReportManager) FlushCache() {
	rm.cacheMutex.Lock()
	defer rm.cacheMutex.Unlock()

	cacheCount := len(rm.cache)
	rm.cache = make(map[string]*ReportCache)

	rm.logger.Info("Cache flushed", "cleared_entries", cacheCount)
}

// GetSupportedTypes returns all supported report types
func (rm *ReportManager) GetSupportedTypes() []ReportType {
	var types []ReportType
	for reportType := range rm.generators {
		types = append(types, reportType)
	}
	return types
}
