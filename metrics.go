package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	testsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "shadowtest_tests_total",
		Help: "The total number of tests performed",
	})

	failuresTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "shadowtest_failures_total",
		Help: "The total number of failed tests",
	})
)
