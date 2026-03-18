package main

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	testsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "shadowtest_tests_total",
		Help: "The total number of tests performed",
	}, []string{"proxy_type"})

	failuresTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "shadowtest_failures_total",
		Help: "The total number of failed tests",
	}, []string{"proxy_type"})
)

func proxyTypeFromAddress(address string) string {
	if strings.HasPrefix(address, "ssr://") {
		return "ssr"
	}
	return "ss"
}
