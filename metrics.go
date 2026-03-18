package main

import (
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
	if len(address) >= 6 && address[:6] == "ssr://" {
		return "ssr"
	}
	return "ss"
}
