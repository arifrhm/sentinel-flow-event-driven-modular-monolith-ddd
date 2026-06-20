package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"

	"sentinel-flow/pkg/broker"
)

func TestSimulatorMain(t *testing.T) {
	// Spin up test server for Ingest Gateway
	ingestMux := http.NewServeMux()
	ingestMux.HandleFunc("/v1/events", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})
	ingestSrv := httptest.NewServer(ingestMux)
	defer ingestSrv.Close()

	// Spin up test server for Marketing Compliance
	mktMux := http.NewServeMux()
	mktMux.HandleFunc("/privacy/ccpa-export", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"user_id":"test"}`))
	})
	mktMux.HandleFunc("/privacy/gdpr-delete", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"purged"}`))
	})
	mktSrv := httptest.NewServer(mktMux)
	defer mktSrv.Close()

	// Spin up test server for Analytics dashboard
	analyticsMux := http.NewServeMux()
	analyticsMux.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`dashboard telemetry`))
	})
	analyticsSrv := httptest.NewServer(analyticsMux)
	defer analyticsSrv.Close()

	// Override URL vars
	oldIngest := ingestURL
	oldMkt := marketingURL
	oldAnalytics := analyticsURL
	defer func() {
		ingestURL = oldIngest
		marketingURL = oldMkt
		analyticsURL = oldAnalytics
	}()

	ingestURL = ingestSrv.URL + "/v1/events"
	marketingURL = mktSrv.URL
	analyticsURL = analyticsSrv.URL

	// Run main function (runs steps 1-5, wait 2s, and step 6)
	go main()

	// Wait 2.5 seconds for main to complete its simulation steps
	time.Sleep(2500 * time.Millisecond)
}

func TestPingClusterFailure(t *testing.T) {
	client := &http.Client{Timeout: 10 * time.Millisecond}
	err := pingCluster(client)
	if err == nil {
		t.Error("expected error from pingCluster with offline URL, got nil")
	}
}

func TestPostEventFailure(t *testing.T) {
	client := &http.Client{Timeout: 10 * time.Millisecond}
	status := postEvent(client, &broker.TrackingEvent{})
	if status != 500 {
		t.Errorf("expected 500, got %d", status)
	}
}

func TestPrintDashboardFailure(t *testing.T) {
	oldAnalytics := analyticsURL
	analyticsURL = "http://invalid-url"
	defer func() { analyticsURL = oldAnalytics }()
	
	client := &http.Client{Timeout: 10 * time.Millisecond}
	printDashboard(client)
}

func TestRunLegitimateTrafficFailure(t *testing.T) {
	oldIngest := ingestURL
	ingestURL = "http://invalid-url"
	defer func() { ingestURL = oldIngest }()
	
	client := &http.Client{Timeout: 10 * time.Millisecond}
	runLegitimateTraffic(client)
}

func TestSimulatorMainExit(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		ingestURL = "http://invalid-url"
		marketingURL = "http://invalid-url"
		analyticsURL = "http://invalid-url"
		main()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestSimulatorMainExit")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}
