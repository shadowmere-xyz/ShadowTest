package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/getsentry/sentry-go"
	log "github.com/sirupsen/logrus"

	_ "embed"

	"github.com/slok/go-http-metrics/middleware"
	"github.com/slok/go-http-metrics/middleware/std"

	metrics "github.com/slok/go-http-metrics/metrics/prometheus"
)

var (
	// Version is set by the build system.
	Version = "dev"
	// GitCommit is set by the build system.
	GitCommit = "HEAD"
)

func main() {
	sentryDsn := os.Getenv("SENTRY_DSN")
	if sentryDsn != "" {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:              sentryDsn,
			TracesSampleRate: 0.1,
		})
		if err != nil {
			log.Fatalf("sentry.Init: %s", err)
		}
	} else {
		log.Warn("SENTRY_DSN was not provided. Running without sentry.")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	router, err := getRouter(true)
	if err != nil {
		log.Fatal(err)
	}

	mdlw := middleware.New(middleware.Config{
		Recorder: metrics.NewRecorder(metrics.Config{
			Prefix: "shadowtest",
		}),
	})
	routerWithMetrics := std.Handler("", mdlw, router)

	log.Infof("Starting server at port %s", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), routerWithMetrics); err != nil {
		log.Fatal(err)
	}
}
