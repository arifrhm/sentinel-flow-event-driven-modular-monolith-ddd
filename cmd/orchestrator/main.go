package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sentinel-flow/pkg/broker"
	"sentinel-flow/pkg/config"
	"sentinel-flow/pkg/domain/analytics"
	"sentinel-flow/pkg/domain/fraud"
	"sentinel-flow/pkg/domain/ingest"
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

	slog.Info("===================================================")
	slog.Info("     BOOTING SENTINEL-FLOW MICROSERVICES CLUSTER    ")
	slog.Info("===================================================")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	memBroker := broker.NewInMemoryBroker(10000)
	defer memBroker.Close()

	// In-memory repositories & trackers
	memMktRepo := marketing.NewMemoryMarketingRepository()
	memFraudRepo := fraud.NewMemoryFraudRepository(func(userID string) {
		_ = memMktRepo.DeleteCRMContact(userID)
	})
	memTracker := analytics.NewMemoryTelemetryTracker()

	// Launch Mock CRM Server (Port 8084)
	crmHandler := resilience.SetupMockCRMServer()
	crmSrv := &http.Server{Addr: ":8084", Handler: crmHandler}
	go func() {
		slog.Info("[Cluster] Mock CRM Server starting on port 8084...")
		_ = crmSrv.ListenAndServe()
	}()

	// Launch Ingest Gateway Service (Port 8081)
	ingestServer := ingest.NewIngestServer(memBroker, memTracker)
	ingestMux := http.NewServeMux()
	ingestServer.SetupRoutes(ingestMux)
	ingestSrv := &http.Server{Addr: ":8081", Handler: ingestMux}
	go func() {
		slog.Info("[Cluster] Ingest Gateway Service starting on port 8081...")
		_ = ingestSrv.ListenAndServe()
	}()

	// Launch Fraud Operations Service (Consumer daemon)
	fraudService := fraud.NewFraudService(memBroker, memFraudRepo, memTracker)
	go func() {
		slog.Info("[Cluster] Fraud Operations Service daemon starting...")
		_ = fraudService.Start(ctx)
	}()

	// Launch Marketing Automation & GDPR Compliance Service (Port 8082 + Consumer daemon)
	rc := resilience.NewResilientClient(
		250*time.Millisecond,
		3,
		5*time.Second,
		2,
		20*time.Millisecond,
		func(from, to resilience.CircuitBreakerState) {
			slog.Warn("[Marketing] CB Transition", "from", from.String(), "to", to.String())
		},
	)
	crmAdapter := marketing.NewHTTPCRMAdapter("http://localhost:8084/crm/sync", rc)
	mktService := marketing.NewMarketingService(memBroker, memMktRepo, memTracker, fraudService, crmAdapter)

	go func() {
		slog.Info("[Cluster] Marketing Automation worker daemon starting...")
		_ = mktService.StartConsumer(ctx)
	}()

	mktMux := http.NewServeMux()
	mktService.SetupRoutes(mktMux)
	mktSrv := &http.Server{Addr: ":8082", Handler: mktMux}
	go func() {
		slog.Info("[Cluster] Marketing Compliance API starting on port 8082...")
		_ = mktSrv.ListenAndServe()
	}()

	// Launch Analytics & Reporting Dashboard Service (Port 8083)
	analyticsService := analytics.NewAnalyticsService(memTracker)
	analyticsMux := http.NewServeMux()
	analyticsService.SetupRoutes(analyticsMux)
	analyticsSrv := &http.Server{Addr: ":8083", Handler: analyticsMux}
	go func() {
		slog.Info("[Cluster] Analytics & Dashboard API starting on port 8083...")
		_ = analyticsSrv.ListenAndServe()
	}()

	slog.Info("[Cluster] All services online. Press Ctrl+C to terminate.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	slog.Info("\n[Cluster] Shutting down services gracefully...")
	cancel()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutCancel()

	_ = crmSrv.Shutdown(shutCtx)
	_ = ingestSrv.Shutdown(shutCtx)
	_ = mktSrv.Shutdown(shutCtx)
	_ = analyticsSrv.Shutdown(shutCtx)

	slog.Info("[Cluster] All services terminated successfully.")
}
