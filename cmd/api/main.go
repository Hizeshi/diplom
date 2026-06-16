package main

import (
	"context"
	"log"
	"time"

	"iq-home/backend/internal/app"
	"iq-home/backend/internal/config"
	"iq-home/backend/internal/observability"
	"iq-home/backend/pkg/logger"
)

func main() {
	cfg := config.MustLoad()

	// Optional distributed tracing. Never fatal: if the collector is missing or
	// tracing is disabled, the app runs exactly as before.
	logr := logger.New()
	shutdown, err := observability.InitTracer(context.Background(), logr)
	if err != nil {
		logr.Warn("tracing init failed, continuing without tracing", "err", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()
		if err := shutdown(ctx); err != nil {
			logr.Warn("tracer shutdown error", "err", err)
		}
	}()

	if err := app.New(cfg).Run(); err != nil {
		log.Fatal(err)
	}
}
