package main

import (
	"context"
	"log/slog"
	"os"

	"sentinel-flow/pkg/broker"
	"sentinel-flow/pkg/config"
	"sentinel-flow/pkg/domain/analytics"
	"sentinel-flow/pkg/domain/fraud"
)

func main() {
	cfg := config.LoadConfig()

	var logHandler slog.Handler
	if cfg.LogFormat == "json" {
		logHandler = slog.NewJSONHandler(os.Stdout, nil)
	} else {
		logHandler = slog.NewTextHandler(os.Stdout, nil)
	}
	slog.SetDefault(slog.New(logHandler))

	memBroker := broker.NewInMemoryBroker(10000)
	defer memBroker.Close()

	if os.Getenv("BE_CRASHER") == "1" {
		_ = memBroker.Close()
	}

	// In standalone, we don't have marketing context active. Set callback to nil
	memRepo := fraud.NewMemoryFraudRepository(nil)
	memTracker := analytics.NewMemoryTelemetryTracker()

	service := fraud.NewFraudService(memBroker, memRepo, memTracker)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	slog.Info("[Fraud] Standalone worker starting...")
	if err := service.Start(ctx); err != nil {
		slog.Error("[Fraud] Service crashed", "error", err)
		os.Exit(1)
	}
}
