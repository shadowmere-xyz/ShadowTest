package ssproxy

import (
	"sync"
	"time"
)

type safeIsOfflineCache struct {
	isOffline          bool
	offlineCacheExpiry time.Time
	mu                 sync.Mutex
}

func (s *safeIsOfflineCache) expired() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return time.Now().After(s.offlineCacheExpiry)
}

func (s *safeIsOfflineCache) setIsOfflineToCache(offlineCache bool, expiry time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isOffline = offlineCache
	s.offlineCacheExpiry = time.Now().Add(expiry)
}

func (s *safeIsOfflineCache) getIsOfflineFromCache() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isOffline
}

func (s *safeIsOfflineCache) isZero() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.offlineCacheExpiry.IsZero()
}
