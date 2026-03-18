package ssproxy

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ssrEncode(s string) string {
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte(s))
}

func TestParseSSRURL(t *testing.T) {
	// ssr://base64(host:port:protocol:method:obfs:base64pass/?obfsparam=base64(...)&protoparam=base64(...))
	inner := "192.168.1.1:8388:auth_aes128_sha1:aes-256-cfb:tls1.2_ticket_auth:" +
		ssrEncode("mypassword") +
		"/?obfsparam=" + ssrEncode("bing.com") +
		"&protoparam=" + ssrEncode("12345:auth")

	url := "ssr://" + ssrEncode(inner)

	config, err := parseSSRURL(url)
	require.NoError(t, err)
	assert.Equal(t, "192.168.1.1", config.Host)
	assert.Equal(t, "8388", config.Port)
	assert.Equal(t, "auth_aes128_sha1", config.Protocol)
	assert.Equal(t, "aes-256-cfb", config.Method)
	assert.Equal(t, "tls1.2_ticket_auth", config.Obfs)
	assert.Equal(t, "mypassword", config.Password)
	assert.Equal(t, "bing.com", config.ObfsParam)
	assert.Equal(t, "12345:auth", config.ProtoParam)
}

func TestParseSSRURLNoParams(t *testing.T) {
	inner := "10.0.0.1:443:origin:aes-128-cfb:plain:" + ssrEncode("pass123")
	url := "ssr://" + ssrEncode(inner)

	config, err := parseSSRURL(url)
	require.NoError(t, err)
	assert.Equal(t, "10.0.0.1", config.Host)
	assert.Equal(t, "443", config.Port)
	assert.Equal(t, "origin", config.Protocol)
	assert.Equal(t, "aes-128-cfb", config.Method)
	assert.Equal(t, "plain", config.Obfs)
	assert.Equal(t, "pass123", config.Password)
	assert.Empty(t, config.ObfsParam)
	assert.Empty(t, config.ProtoParam)
}

func TestParseSSRURLIPv6Host(t *testing.T) {
	inner := "::1:8388:origin:aes-256-cfb:plain:" + ssrEncode("password")
	url := "ssr://" + ssrEncode(inner)

	config, err := parseSSRURL(url)
	require.NoError(t, err)
	assert.Equal(t, "::1", config.Host)
	assert.Equal(t, "8388", config.Port)
	assert.Equal(t, "password", config.Password)
}

func TestParseSSRURLMissingPrefix(t *testing.T) {
	_, err := parseSSRURL("ss://something")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ssr://")
}

func TestParseSSRURLBadBase64(t *testing.T) {
	_, err := parseSSRURL("ssr://!invalid!base64!")
	assert.Error(t, err)
}

func TestParseSSRURLTooFewFields(t *testing.T) {
	inner := "host:port:only:three"
	url := "ssr://" + ssrEncode(inner)

	_, err := parseSSRURL(url)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "6 colon-separated fields")
}

func TestParseSSRURLWithSlashSeparator(t *testing.T) {
	// Some SSR URLs use "/" instead of "/?" as separator
	inner := "10.0.0.1:443:origin:aes-128-cfb:plain:" + ssrEncode("pass123") + "/"
	url := "ssr://" + ssrEncode(inner)

	config, err := parseSSRURL(url)
	require.NoError(t, err)
	assert.Equal(t, "pass123", config.Password)
}

func TestBase64DecodeSSRURLSafe(t *testing.T) {
	// Test that URL-safe base64 characters (- and _) are handled
	original := "test+data/with=special"
	encoded := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte(original))

	decoded, err := base64DecodeSSR(encoded)
	require.NoError(t, err)
	assert.Equal(t, original, decoded)
}

func TestBase64DecodeSSRPaddingNormalization(t *testing.T) {
	// Test that base64DecodeSSR handles strings with stripped padding
	tests := []string{"a", "ab", "abc", "abcd", "abcde"}
	for _, orig := range tests {
		// Encode without padding (as SSR URLs do in practice)
		stripped := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte(orig))

		decoded, err := base64DecodeSSR(stripped)
		require.NoError(t, err)
		assert.Equal(t, orig, decoded)
	}
}
