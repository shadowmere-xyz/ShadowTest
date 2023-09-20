package ssproxy

import (
	"testing"

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
