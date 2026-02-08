package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

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
			Release:          fmt.Sprintf("shadowtest@%s", Version),
			Environment:      getEnvironment(),
		})
		if err != nil {
			log.Fatalf("sentry.Init: %s", err)
		}
		defer sentry.Flush(2 * time.Second)
		log.Info("Sentry initialized successfully")
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

	// Wrap router with Sentry middleware
	handlerWithSentry := sentryMiddleware(router)

	mdlw := middleware.New(middleware.Config{
		Recorder: metrics.NewRecorder(metrics.Config{
			Prefix: "shadowtest",
		}),
	})
	routerWithMetrics := std.Handler("", mdlw, handlerWithSentry)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: routerWithMetrics,
	}

	go func() {
		log.Infof("Starting server at port %s", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down server...")

	shutdownTimeout := 60 * time.Second
	if envTimeout := os.Getenv("SHUTDOWN_TIMEOUT"); envTimeout != "" {
		if t, err := strconv.Atoi(envTimeout); err == nil {
			shutdownTimeout = time.Duration(t) * time.Second
		} else {
			log.Warnf("Invalid SHUTDOWN_TIMEOUT value '%s', using default 60s", envTimeout)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Info("Server exiting")
}

func getEnvironment() string {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "production"
	}
	return env
}

func sentryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub := sentry.CurrentHub().Clone()
		hub.Scope().SetRequest(r)
		hub.Scope().SetTag("path", r.URL.Path)
		hub.Scope().SetTag("method", r.Method)
		ctx := sentry.SetHubOnContext(r.Context(), hub)

		defer func() {
			if err := recover(); err != nil {
				hub.RecoverWithContext(ctx, err)
				log.Errorf("Panic recovered: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
