package ssproxy

import (
	"ShadowTest/offlinecache"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsWTFIsMyIpOffline_Online(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	offlineCache := offlinecache.SafeIsOfflineCache{}
	offline := IsWTFIsMyIpOffline(&offlineCache, server.URL)
	assert.False(t, offline)
}

func TestIsWTFIsMyIpOffline_Offline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	offlineCache := offlinecache.SafeIsOfflineCache{}
	offline := IsWTFIsMyIpOffline(&offlineCache, server.URL)
	assert.True(t, offline)
}

func TestIsWTFIsMyIpOffline_ConnectionError(t *testing.T) {
	offlineCache := offlinecache.SafeIsOfflineCache{}
	offline := IsWTFIsMyIpOffline(&offlineCache, "http://127.0.0.1:0")
	assert.True(t, offline)
}

func TestIsWTFIsMyIpOffline_UsesCache(t *testing.T) {
	offlineCache := &offlinecache.SafeIsOfflineCache{}
	offlineCache.SetIsOfflineToCache(true, 5*time.Minute)

	result := IsWTFIsMyIpOffline(offlineCache, "http://127.0.0.1:0")
	assert.True(t, result)
}

// TestTestProxyDispatchesSSR verifies that TestProxy routes ssr:// URLs to the SSR handler
func TestTestProxyDispatchesSSR(t *testing.T) {
	ssrURL := buildSSRURL("127.0.0.1", "16276", "origin", "aes-256-cfb", "plain", "testpassword")

	details, err := TestProxy(ssrURL, true, 30)
	assert.NoError(t, err)
	assert.NotEmpty(t, details.YourFuckingIPAddress)
}

// TestTestProxyDispatchesSS verifies that TestProxy still routes ss:// URLs to the SS handler
func TestTestProxyDispatchesSS(t *testing.T) {
	// This test requires ss-server running on localhost:6276
	// It will fail without it, same as TestGetProxyDetails
	address := "ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTpwYXNzd29yZA@localhost:6276/?outline=1"
	details, err := TestProxy(address, true, 30)
	assert.NoError(t, err)
	assert.NotEmpty(t, details.YourFuckingIPAddress)
}
