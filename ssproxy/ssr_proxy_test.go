package ssproxy

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildSSRURL(host, port, protocol, method, obfs, password string) string {
	b64pass := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte(password))
	inner := fmt.Sprintf("%s:%s:%s:%s:%s:%s", host, port, protocol, method, obfs, b64pass)
	encoded := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte(inner))
	return "ssr://" + encoded
}

func TestGetSSRProxyDetails(t *testing.T) {
	ssrURL := buildSSRURL("127.0.0.1", "16276", "origin", "aes-256-cfb", "plain", "testpassword")

	details, err := GetSSRProxyDetails(ssrURL, true, 30)
	require.NoError(t, err)
	assert.NotEmpty(t, details.YourFuckingIPAddress)
	assert.NotEmpty(t, details.YourFuckingLocation)
}

func TestGetSSRProxyDetailsWrongPassword(t *testing.T) {
	ssrURL := buildSSRURL("127.0.0.1", "16276", "origin", "aes-256-cfb", "plain", "wrongpassword")

	details, err := GetSSRProxyDetails(ssrURL, true, 5)
	assert.Error(t, err)
	assert.Empty(t, details.YourFuckingIPAddress)
}

func TestGetSSRProxyDetailsWrongPort(t *testing.T) {
	ssrURL := buildSSRURL("127.0.0.1", "19999", "origin", "aes-256-cfb", "plain", "testpassword")

	details, err := GetSSRProxyDetails(ssrURL, true, 5)
	assert.Error(t, err)
	assert.Empty(t, details.YourFuckingIPAddress)
}

func TestGetSSRProxyDetailsInvalidURL(t *testing.T) {
	_, err := GetSSRProxyDetails("ssr://invalid", true, 5)
	assert.Error(t, err)
}

func TestGetSSRProxyDetailsUnsupportedCipher(t *testing.T) {
	ssrURL := buildSSRURL("127.0.0.1", "16276", "origin", "not-a-cipher", "plain", "testpassword")

	_, err := GetSSRProxyDetails(ssrURL, true, 5)
	assert.Error(t, err)
}
