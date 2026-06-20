package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"sentinel-flow/pkg/broker"
	"sentinel-flow/pkg/config"
	"sentinel-flow/pkg/domain/analytics"
	"sentinel-flow/pkg/domain/fraud"
	"sentinel-flow/pkg/domain/marketing"
	"sentinel-flow/pkg/resilience"
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

	memMktRepo := marketing.NewMemoryMarketingRepository()
	memFraudRepo := fraud.NewMemoryFraudRepository(func(userID string) {
		_ = memMktRepo.DeleteCRMContact(userID)
	})
	memTracker := analytics.NewMemoryTelemetryTracker()
	memBroker := broker.NewInMemoryBroker(10000)
	defer memBroker.Close()

	cbTransition := func(from, to resilience.CircuitBreakerState) {
		slog.Warn("[Marketing] Standalone CB Transition", "from", from.String(), "to", to.String())
	}

	if os.Getenv("BE_CRASHER_CB") == "1" {
		cbTransition(resilience.StateClosed, resilience.StateOpen)
	}

	rc := resilience.NewResilientClient(
		200*time.Millisecond,
		3,
		5*time.Second,
		2,
		20*time.Millisecond,
		cbTransition,
	)

	crmAdapter := marketing.NewHTTPCRMAdapter(cfg.CRMURL, rc)
	service := marketing.NewMarketingService(memBroker, memMktRepo, memTracker, memFraudRepo, crmAdapter)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if os.Getenv("BE_CRASHER_CONSUMER") == "1" {
			_ = memBroker.Close()
		}
		if err := service.StartConsumer(ctx); err != nil {
			slog.Error("[Marketing] Consumer failed", "error", err)
			os.Exit(1)
		}
	}()

	if os.Getenv("BE_CRASHER_CONSUMER") == "1" {
		time.Sleep(200 * time.Millisecond)
	}

	mux := http.NewServeMux()
	service.SetupRoutes(mux)

	slog.Info("[Marketing] Compliance API starting", "port", cfg.PortMarketing)
	if err := http.ListenAndServe(":"+cfg.PortMarketing, mux); err != nil {
		slog.Error("[Marketing] Server failed", "error", err)
		os.Exit(1)
	}
}
