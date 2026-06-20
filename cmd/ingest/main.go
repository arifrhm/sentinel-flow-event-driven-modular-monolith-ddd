package main

import (
	"log/slog"
	"net/http"
	"os"

	"sentinel-flow/pkg/broker"
	"sentinel-flow/pkg/config"
	"sentinel-flow/pkg/domain/analytics"
	"sentinel-flow/pkg/domain/ingest"
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

	memTracker := analytics.NewMemoryTelemetryTracker()
	memBroker := broker.NewInMemoryBroker(10000)
	defer memBroker.Close()

	server := ingest.NewIngestServer(memBroker, memTracker)
	mux := http.NewServeMux()
	server.SetupRoutes(mux)

	slog.Info("[Ingest] Gateway starting", "port", cfg.PortIngest)
	if err := http.ListenAndServe(":"+cfg.PortIngest, mux); err != nil {
		slog.Error("[Ingest] Server failed", "error", err)
		os.Exit(1)
	}
}
