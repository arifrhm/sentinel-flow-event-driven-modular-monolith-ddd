package main

import (
	"log/slog"
	"net/http"
	"os"

	"sentinel-flow/pkg/config"
	"sentinel-flow/pkg/domain/analytics"
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
	server := analytics.NewAnalyticsService(memTracker)

	mux := http.NewServeMux()
	server.SetupRoutes(mux)

	slog.Info("[Analytics] Dashboard API starting", "port", cfg.PortAnalytics)
	if err := http.ListenAndServe(":"+cfg.PortAnalytics, mux); err != nil {
		slog.Error("[Analytics] Server failed", "error", err)
		os.Exit(1)
	}
}
