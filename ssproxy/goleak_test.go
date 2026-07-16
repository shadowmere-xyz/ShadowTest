package ssproxy

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain runs the ssproxy test suite under goleak so that any goroutine
// leaked by the proxy-testing code (the SOCKS listener started by
// ListenForOneConnection and the bidirectional copy goroutines started by
// relay) fails the package instead of silently piling up.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
