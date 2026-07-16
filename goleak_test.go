package main

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain runs the package test suite under goleak. The /v3/test handler drives
// the real proxy-testing code (ssproxy.GetShadowsocksProxyDetails), which starts
// a SOCKS listener goroutine and bidirectional relay goroutines per request, so
// this guards the HTTP entry point against goroutine leaks.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
