package ssproxy

import (
	"ShadowTest/offlinecache"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestParseUrl(t *testing.T) {
	address := "ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTpwYXNzd29yZA@localhost:6276/?outline=1"
	addr, cipher, password, err := parseURL(address)
	require.NoError(t, err)
	assert.Equal(t, "localhost:6276", addr)
	assert.Equal(t, "chacha20-ietf-poly1305", cipher)
	assert.Equal(t, "password", password)
}

func TestParseBadURL(t *testing.T) {
	address := "aaa"
	_, _, _, err := parseURL(address)
	assert.Error(t, err)
}

func TestParseURLWithSpecialCharactersInPassword(t *testing.T) {
	address := "ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTp0Lm1lL091dGxpbmVWcG5PZmZpY2lhbA==@www.outline.network.lki33eqtfhgp5p4qrtdkyv5cqjp3r6zqhxacluztckcvj7qdorl4ulpt9jf6u29.fr8678825324247b8176d59f83c30bd94d23d2e3ac5cd4a743bkwqeikvdyufr.cyou:8080#t.me%2FOutlineVpnOfficial%20%7C%20%281875%29%20IN"
	addr, cipher, password, err := parseURL(address)
	require.NoError(t, err)
	assert.Equal(t, "www.outline.network.lki33eqtfhgp5p4qrtdkyv5cqjp3r6zqhxacluztckcvj7qdorl4ulpt9jf6u29.fr8678825324247b8176d59f83c30bd94d23d2e3ac5cd4a743bkwqeikvdyufr.cyou:8080", addr)
	assert.Equal(t, "chacha20-ietf-poly1305", cipher)
	assert.Equal(t, "t.me/OutlineVpnOfficial", password)
}

func TestParseURLNoBase64(t *testing.T) {
	address := "ss://chacha20-ietf-poly1305:password@localhost:6276/?outline=1"
	_, _, _, err := parseURL(address)
	assert.Error(t, err)
}

func TestGetProxyDetails(t *testing.T) {
	address := "ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTpwYXNzd29yZA@localhost:6276/?outline=1"
	details, err := GetShadowsocksProxyDetails(address, true, 30)
	assert.NoError(t, err)

	assert.NotEmpty(t, details.YourFuckingIPAddress)
	assert.NotEmpty(t, details.YourFuckingLocation)
}

func TestGetProxyDetailsWrongCredentials(t *testing.T) {
	address := "ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTpiYWRwYXNzd29yZA@localhost:6276/?outline=1"
	details, err := GetShadowsocksProxyDetails(address, true, 30)
	assert.Error(t, err)

	assert.Empty(t, details.YourFuckingIPAddress)
	assert.Empty(t, details.YourFuckingLocation)
}

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
