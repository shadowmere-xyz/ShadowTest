package offline_cache

import (
	"sync"
	"time"
)

type SafeIsOfflineCache struct {
	isOffline          bool
	offlineCacheExpiry time.Time
	mu                 sync.Mutex
}

func (s *SafeIsOfflineCache) Expired() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return time.Now().After(s.offlineCacheExpiry)
}

func (s *SafeIsOfflineCache) SetIsOfflineToCache(offlineCache bool, expiry time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isOffline = offlineCache
	s.offlineCacheExpiry = time.Now().Add(expiry)
}

func (s *SafeIsOfflineCache) GetIsOfflineFromCache() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isOffline
}

func (s *SafeIsOfflineCache) IsZero() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.offlineCacheExpiry.IsZero()
}
