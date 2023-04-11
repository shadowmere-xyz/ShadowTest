package ssproxy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseUrl(t *testing.T) {
	address := "ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTpwYXNzd29yZA@localhost:6276/?outline=1"
	addr, cipher, password, err := parseURL(address)
	assert.NoError(t, err)
	assert.Equal(t, "localhost:6276", addr)
	assert.Equal(t, "chacha20-ietf-poly1305", cipher)
	assert.Equal(t, "password", password)
}

func TestParseBadURL(t *testing.T) {
	address := "aaa"
	_, _, _, err := parseURL(address)
	assert.Error(t, err)
}

func TestParseURLNoBase64(t *testing.T) {
	address := "ss://chacha20-ietf-poly1305:password@localhost:6276/?outline=1"
	_, _, _, err := parseURL(address)
	assert.Error(t, err)
}

func TestGetProxyDetails(t *testing.T) {
	address := "ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTpwYXNzd29yZA@localhost:6276/?outline=1"
	details, err := GetShadowsocksProxyDetails(address, true)
	assert.NoError(t, err)

	assert.NotEmpty(t, details.YourFuckingIPAddress)
	assert.NotEmpty(t, details.YourFuckingLocation)
}

func TestGetProxyDetailsWrongCredentials(t *testing.T) {
	address := "ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTpiYWRwYXNzd29yZA@localhost:6276/?outline=1"
	details, err := GetShadowsocksProxyDetails(address, true)
	assert.Error(t, err)

	assert.Empty(t, details.YourFuckingIPAddress)
	assert.Empty(t, details.YourFuckingLocation)
}
