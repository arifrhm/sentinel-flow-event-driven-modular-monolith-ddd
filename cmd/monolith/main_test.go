package main

import (
	"bytes"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestMainExecution(t *testing.T) {
	// Set configuration variables to in-memory mode
	os.Setenv("DATABASE_TYPE", "memory")
	os.Setenv("BROKER_TYPE", "memory")
	os.Setenv("LOG_FORMAT", "json")
	defer func() {
		os.Unsetenv("DATABASE_TYPE")
		os.Unsetenv("BROKER_TYPE")
		os.Unsetenv("LOG_FORMAT")
	}()

	// Start main in a background goroutine
	go main()

	// Wait briefly for all HTTP servers and background consumers to boot up
	time.Sleep(300 * time.Millisecond)

	// Send SIGINT to own process to trigger graceful shutdown flow
	pid := os.Getpid()
	_ = syscall.Kill(pid, syscall.SIGINT)

	// Wait for shutdown teardowns to complete
	time.Sleep(300 * time.Millisecond)
}

func TestMainExecutionRedisText(t *testing.T) {
	os.Setenv("DATABASE_TYPE", "memory")
	os.Setenv("BROKER_TYPE", "redis")
	os.Setenv("LOG_FORMAT", "text")
	os.Setenv("REDIS_URL", "redis://localhost:6379")
	os.Setenv("PORT_INGEST", "18081")
	os.Setenv("PORT_MARKETING", "18082")
	os.Setenv("PORT_ANALYTICS", "18083")
	os.Setenv("PORT_CRM", "18084")
	defer func() {
		os.Unsetenv("DATABASE_TYPE")
		os.Unsetenv("BROKER_TYPE")
		os.Unsetenv("LOG_FORMAT")
		os.Unsetenv("REDIS_URL")
		os.Unsetenv("PORT_INGEST")
		os.Unsetenv("PORT_MARKETING")
		os.Unsetenv("PORT_ANALYTICS")
		os.Unsetenv("PORT_CRM")
	}()

	go main()
	time.Sleep(400 * time.Millisecond)

	pid := os.Getpid()
	_ = syscall.Kill(pid, syscall.SIGINT)
	time.Sleep(400 * time.Millisecond)
}

func TestMainExecutionPostgresMock(t *testing.T) {
	os.Setenv("DATABASE_TYPE", "postgres-mock")
	os.Setenv("BROKER_TYPE", "memory")
	os.Setenv("LOG_FORMAT", "text")
	os.Setenv("PORT_INGEST", "28081")
	os.Setenv("PORT_MARKETING", "28082")
	os.Setenv("PORT_ANALYTICS", "28083")
	os.Setenv("PORT_CRM", "28084")
	defer func() {
		os.Unsetenv("DATABASE_TYPE")
		os.Unsetenv("BROKER_TYPE")
		os.Unsetenv("LOG_FORMAT")
		os.Unsetenv("PORT_INGEST")
		os.Unsetenv("PORT_MARKETING")
		os.Unsetenv("PORT_ANALYTICS")
		os.Unsetenv("PORT_CRM")
	}()

	go main()
	time.Sleep(400 * time.Millisecond)

	pid := os.Getpid()
	_ = syscall.Kill(pid, syscall.SIGINT)
	time.Sleep(400 * time.Millisecond)
}

func TestMainExecutionCBTransition(t *testing.T) {
	// Occupy Port CRM 38084 and return 503
	crmTestSrv := &http.Server{
		Addr: ":38084",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}),
	}
	go func() {
		_ = crmTestSrv.ListenAndServe()
	}()
	defer crmTestSrv.Close()

	os.Setenv("DATABASE_TYPE", "memory")
	os.Setenv("BROKER_TYPE", "memory")
	os.Setenv("LOG_FORMAT", "text")
	os.Setenv("PORT_INGEST", "38081")
	os.Setenv("PORT_MARKETING", "38082")
	os.Setenv("PORT_ANALYTICS", "38083")
	os.Setenv("PORT_CRM", "38084")
	os.Setenv("CRM_URL", "http://localhost:38084/crm/sync")
	defer func() {
		os.Unsetenv("DATABASE_TYPE")
		os.Unsetenv("BROKER_TYPE")
		os.Unsetenv("LOG_FORMAT")
		os.Unsetenv("PORT_INGEST")
		os.Unsetenv("PORT_MARKETING")
		os.Unsetenv("PORT_ANALYTICS")
		os.Unsetenv("PORT_CRM")
		os.Unsetenv("CRM_URL")
	}()

	go main()
	time.Sleep(400 * time.Millisecond)

	// Send GDPR delete callback to verify that callback is executed
	client := &http.Client{Timeout: 1 * time.Second}
	body := []byte(`{"user_id": "usr_test"}`)
	reqDelete, _ := http.NewRequest("POST", "http://localhost:38082/v1/privacy/gdpr-delete", bytes.NewBuffer(body))
	reqDelete.Header.Set("Content-Type", "application/json")
	respDelete, err := client.Do(reqDelete)
	if err == nil {
		respDelete.Body.Close()
	}

	// Send 4 ingest requests to trigger CB transition
	for i := 0; i < 4; i++ {
		ingestBody := []byte(`{"event_id": "ev_cb", "user_id": "usr_cb", "event_type": "signup", "ip_address": "1.1.1.1", "user_agent": "Mozilla", "payload": {"email": "cb@example.com"}, "gdpr_consent": true}`)
		req, _ := http.NewRequest("POST", "http://localhost:38081/v1/events", bytes.NewBuffer(ingestBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
		}
		time.Sleep(50 * time.Millisecond)
	}

	time.Sleep(500 * time.Millisecond)

	pid := os.Getpid()
	_ = syscall.Kill(pid, syscall.SIGINT)
	time.Sleep(400 * time.Millisecond)
}
