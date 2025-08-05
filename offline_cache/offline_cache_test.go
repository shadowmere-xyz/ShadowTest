package offline_cache

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestSetIsOfflineToCacheAndGetIsOfflineFromCache(t *testing.T) {
	cache := &SafeIsOfflineCache{}
	cache.SetIsOfflineToCache(true, time.Second*10)
	assert.True(t, cache.GetIsOfflineFromCache(), "Expected isOffline to be true")
	cache.SetIsOfflineToCache(false, time.Second*10)
	assert.False(t, cache.GetIsOfflineFromCache(), "Expected isOffline to be false")
}

func TestExpired(t *testing.T) {
	cache := &SafeIsOfflineCache{}
	cache.SetIsOfflineToCache(true, time.Millisecond*10)
	time.Sleep(time.Millisecond * 20)
	assert.True(t, cache.Expired(), "Expected cache to be expired")
	cache.SetIsOfflineToCache(true, time.Second)
	assert.False(t, cache.Expired(), "Expected cache not to be expired")
}

func TestIsZero(t *testing.T) {
	cache := &SafeIsOfflineCache{}
	assert.True(t, cache.IsZero(), "Expected offlineCacheExpiry to be zero")
	cache.SetIsOfflineToCache(true, time.Second)
	assert.False(t, cache.IsZero(), "Expected offlineCacheExpiry not to be zero")
}
