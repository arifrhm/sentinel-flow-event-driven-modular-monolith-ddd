package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sentinel-flow/pkg/config"
	"sentinel-flow/pkg/domain/analytics"
	"sentinel-flow/pkg/domain/fraud"
	"sentinel-flow/pkg/domain/ingest"
	"sentinel-flow/pkg/domain/marketing"
	"sentinel-flow/pkg/resilience"
)

func main() {
	cfg := config.LoadConfig()

	logger := setupLogger(cfg.LogFormat)
	slog.SetDefault(logger)

	slog.Info("===================================================")
	slog.Info("     BOOTING SENTINEL-FLOW MODULAR MONOLITH       ")
	slog.Info("===================================================")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sqlDB := initDatabase(cfg)
	eventBroker := initBroker(cfg)
	defer eventBroker.Close()

	// Instantiate repositories/trackers depending on configuration
	var ingestTracker ingest.TelemetryTracker
	var fraudTracker fraud.TelemetryTracker
	var mktTracker marketing.TelemetryTracker

	var fraudRepo fraud.FraudRepository
	var mktRepo marketing.MarketingRepository
	var telemetryTracker analytics.TelemetryTracker

	if sqlDB != nil {
		fraudRepo = fraud.NewPostgresFraudRepository(sqlDB)
		mktRepo = marketing.NewPostgresMarketingRepository(sqlDB)
		pgTracker := analytics.NewPostgresTelemetryTracker(sqlDB)
		telemetryTracker = pgTracker
		ingestTracker = pgTracker
		fraudTracker = pgTracker
		mktTracker = pgTracker
	} else {
		memMktRepo := marketing.NewMemoryMarketingRepository()
		mktRepo = memMktRepo
		fraudRepo = fraud.NewMemoryFraudRepository(func(userID string) {
			_ = memMktRepo.DeleteCRMContact(userID)
		})
		memTracker := analytics.NewMemoryTelemetryTracker()
		telemetryTracker = memTracker
		ingestTracker = memTracker
		fraudTracker = memTracker
		mktTracker = memTracker
	}

	// Launch Mock CRM Server
	crmSrv := &http.Server{Addr: ":" + cfg.PortCRM, Handler: resilience.SetupMockCRMServer()}
	go func() {
		slog.Info("[Cluster] Mock CRM Server starting", "port", cfg.PortCRM)
		_ = crmSrv.ListenAndServe()
	}()

	// Launch Ingest Gateway Module
	ingestServer := ingest.NewIngestServer(eventBroker, ingestTracker)
	ingestMux := http.NewServeMux()
	ingestServer.SetupRoutes(ingestMux)
	ingestSrv := &http.Server{Addr: ":" + cfg.PortIngest, Handler: ingestMux}
	go func() {
		slog.Info("[Cluster] Ingest Gateway Service starting", "port", cfg.PortIngest)
		_ = ingestSrv.ListenAndServe()
	}()

	// Launch Fraud Operations Module
	fraudService := fraud.NewFraudService(eventBroker, fraudRepo, fraudTracker)
	go func() {
		slog.Info("[Cluster] Fraud Operations Service daemon starting...")
		_ = fraudService.Start(ctx)
	}()

	// Launch Marketing Automation Module
	rc := resilience.NewResilientClient(
		250*time.Millisecond,
		3,
		5*time.Second,
		2,
		20*time.Millisecond,
		func(from, to resilience.CircuitBreakerState) {
			slog.Warn("[Marketing] CIRCUIT BREAKER STATE TRANSITION", "from", from.String(), "to", to.String())
		},
	)
	crmAdapter := marketing.NewHTTPCRMAdapter(cfg.CRMURL, rc)
	mktService := marketing.NewMarketingService(eventBroker, mktRepo, mktTracker, fraudService, crmAdapter)
	go func() {
		slog.Info("[Cluster] Marketing Automation worker daemon starting...")
		_ = mktService.StartConsumer(ctx)
	}()

	mktMux := http.NewServeMux()
	mktService.SetupRoutes(mktMux)
	mktSrv := &http.Server{Addr: ":" + cfg.PortMarketing, Handler: mktMux}
	go func() {
		slog.Info("[Cluster] Marketing Compliance API starting", "port", cfg.PortMarketing)
		_ = mktSrv.ListenAndServe()
	}()

	// Launch Analytics Module
	analyticsService := analytics.NewAnalyticsService(telemetryTracker)
	analyticsMux := http.NewServeMux()
	analyticsService.SetupRoutes(analyticsMux)
	analyticsSrv := &http.Server{Addr: ":" + cfg.PortAnalytics, Handler: analyticsMux}
	go func() {
		slog.Info("[Cluster] Analytics & Dashboard API starting", "port", cfg.PortAnalytics)
		_ = analyticsSrv.ListenAndServe()
	}()

	slog.Info("[Cluster] All monolith modules online. Press Ctrl+C to terminate.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	slog.Info("\n[Cluster] Shutting down modular monolith services gracefully...")
	cancel()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutCancel()

	_ = crmSrv.Shutdown(shutCtx)
	_ = ingestSrv.Shutdown(shutCtx)
	_ = mktSrv.Shutdown(shutCtx)
	_ = analyticsSrv.Shutdown(shutCtx)

	if sqlDB != nil {
		slog.Info("[Cluster] Closing PostgreSQL connection pool...")
		_ = sqlDB.Close()
	}
	slog.Info("[Cluster] All monolith modules terminated successfully.")
}
