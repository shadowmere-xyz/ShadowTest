package ssproxy

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
)

// SSRConfig holds the parsed configuration from an SSR URL.
type SSRConfig struct {
	Host       string
	Port       string
	Protocol   string
	Method     string
	Obfs       string
	Password   string
	ObfsParam  string
	ProtoParam string
}

// parseSSRURL parses an SSR URL of the format:
// ssr://base64(host:port:protocol:method:obfs:base64(password)/?params)
func parseSSRURL(s string) (SSRConfig, error) {
	if !strings.HasPrefix(s, "ssr://") {
		return SSRConfig{}, fmt.Errorf("address does not have ssr:// prefix")
	}

	encoded := s[6:]
	decoded, err := base64DecodeSSR(encoded)
	if err != nil {
		return SSRConfig{}, fmt.Errorf("failed to base64 decode SSR URL: %w", err)
	}

	// Split params from the main part: host:port:protocol:method:obfs:base64pass/?params
	mainPart := decoded
	paramsPart := ""
	if idx := strings.Index(decoded, "/?"); idx != -1 {
		mainPart = decoded[:idx]
		paramsPart = decoded[idx+2:]
	} else {
		// Trim a bare trailing "/" if present (some SSR URLs end with one)
		mainPart = strings.TrimRight(decoded, "/")
	}

	// Split main part on ":"
	// Format: host:port:protocol:method:obfs:base64pass
	// IPv6 hosts may contain colons, so we split from the right
	parts := strings.Split(mainPart, ":")
	if len(parts) < 6 {
		return SSRConfig{}, fmt.Errorf("invalid SSR URL format: expected at least 6 colon-separated fields, got %d", len(parts))
	}

	// The last 5 fields are: port, protocol, method, obfs, base64pass
	// Everything before that is the host (handles IPv6)
	n := len(parts)
	base64Pass := parts[n-1]
	obfs := parts[n-2]
	method := parts[n-3]
	protocol := parts[n-4]
	port := parts[n-5]
	host := strings.Join(parts[:n-5], ":")

	password, err := base64DecodeSSR(base64Pass)
	if err != nil {
		return SSRConfig{}, fmt.Errorf("failed to decode SSR password: %w", err)
	}

	config := SSRConfig{
		Host:     host,
		Port:     port,
		Protocol: protocol,
		Method:   method,
		Obfs:     obfs,
		Password: password,
	}

	// Parse optional query parameters
	if paramsPart != "" {
		params, parseErr := url.ParseQuery(paramsPart)
		if parseErr != nil {
			return SSRConfig{}, fmt.Errorf("failed to parse SSR query parameters: %w", parseErr)
		}
		if v := params.Get("obfsparam"); v != "" {
			decoded, decErr := base64DecodeSSR(v)
			if decErr != nil {
				return SSRConfig{}, fmt.Errorf("failed to decode obfsparam: %w", decErr)
			}
			config.ObfsParam = decoded
		}
		if v := params.Get("protoparam"); v != "" {
			decoded, decErr := base64DecodeSSR(v)
			if decErr != nil {
				return SSRConfig{}, fmt.Errorf("failed to decode protoparam: %w", decErr)
			}
			config.ProtoParam = decoded
		}
	}

	if config.Host == "" || config.Port == "" || config.Method == "" || config.Password == "" {
		return SSRConfig{}, fmt.Errorf("invalid SSR URL: missing required fields")
	}

	return config, nil
}

// base64DecodeSSR decodes a base64 string using URL-safe encoding with padding normalization.
// SSR URLs use URL-safe base64 (- and _ instead of + and /) and may omit padding.
func base64DecodeSSR(s string) (string, error) {
	// Replace URL-safe characters with standard base64
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")

	// Add padding if needed
	if m := len(s) % 4; m != 0 {
		s += strings.Repeat("=", 4-m)
	}

	decoded, err := base64.StdEncoding.DecodeString(s)
	return string(decoded), err
}
