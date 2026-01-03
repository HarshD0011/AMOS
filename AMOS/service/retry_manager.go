package service

import (
	"log"
	"sync"
	"time"

	"github.com/HarshD0011/AMOS/AMOS/config"
)

// RetryManager tracks remediation attempts per fault
type RetryManager struct {
	config      config.RemediationConfig
	attempts    map[string]int // Key: resourceID (namespace/name/kind)
	mu          sync.Mutex
	lastAttempt map[string]time.Time
}

// NewRetryManager creates a manager
func NewRetryManager(cfg config.RemediationConfig) *RetryManager {
	return &RetryManager{
		config:      cfg,
		attempts:    make(map[string]int),
		lastAttempt: make(map[string]time.Time),
	}
}

// CanRetry checks if we have retries remaining for this resource
func (rm *RetryManager) CanRetry(resourceID string) bool {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	count := rm.attempts[resourceID]
	
	// Check backoff
	if last, ok := rm.lastAttempt[resourceID]; ok {
		if time.Since(last) < time.Duration(rm.config.RetryBackoffSeconds)*time.Second {
			log.Printf("Retry backoff check: Waiting for %s... (Time since last: %v)", resourceID, time.Since(last))
			return false
		}
	}

	return count < rm.config.MaxRetries
}

// IncrementRetry increments the attempt counter
func (rm *RetryManager) IncrementRetry(resourceID string) int {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.attempts[resourceID]++
	rm.lastAttempt[resourceID] = time.Now()
	
	log.Printf("Incremented retry count for %s: %d/%d", resourceID, rm.attempts[resourceID], rm.config.MaxRetries)
	return rm.attempts[resourceID]
}

// Reset clears the methods for a resolved resource
func (rm *RetryManager) Reset(resourceID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	delete(rm.attempts, resourceID)
	delete(rm.lastAttempt, resourceID)
	log.Printf("Reset retry count for %s", resourceID)
}

// GetAttemptCount returns current attempts
func (rm *RetryManager) GetAttemptCount(resourceID string) int {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	return rm.attempts[resourceID]
}
