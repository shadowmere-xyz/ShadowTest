package offlinecache

import (
	"sync"
	"time"
)

// SafeIsOfflineCache provides thread-safe caching of offline status.
// It stores the current offline state along with an expiration time
// and protects access to this data with a mutex for concurrent usage.
type SafeIsOfflineCache struct {
	isOffline          bool
	offlineCacheExpiry time.Time
	mu                 sync.Mutex
}

// Expired checks if the cached offline status has expired.
func (s *SafeIsOfflineCache) Expired() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return time.Now().After(s.offlineCacheExpiry)
}

// SetIsOfflineToCache sets the offline status and its expiration time.
func (s *SafeIsOfflineCache) SetIsOfflineToCache(offlineCache bool, expiry time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isOffline = offlineCache
	s.offlineCacheExpiry = time.Now().Add(expiry)
}

// GetIsOfflineFromCache retrieves the cached offline status.
func (s *SafeIsOfflineCache) GetIsOfflineFromCache() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isOffline
}

// IsZero checks if the offline cache has not been initialized or set.
func (s *SafeIsOfflineCache) IsZero() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.offlineCacheExpiry.IsZero()
}
