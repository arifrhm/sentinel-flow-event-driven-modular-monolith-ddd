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
	os.Setenv("LOG_FORMAT", "text")
	defer os.Unsetenv("LOG_FORMAT")

	// Start orchestrator in a goroutine
	go main()

	// Wait briefly for all services to boot
	time.Sleep(400 * time.Millisecond)

	// Send GDPR purge request to the running Marketing Compliance API
	body := []byte(`{"user_id":"test_orchestrator_purge"}`)
	resp, err := http.Post("http://localhost:8082/v1/privacy/gdpr-delete", "application/json", bytes.NewBuffer(body))
	if err == nil {
		resp.Body.Close()
	}

	// Send SIGINT to own process to trigger graceful shutdown flow
	pid := os.Getpid()
	_ = syscall.Kill(pid, syscall.SIGINT)

	// Wait for teardown
	time.Sleep(400 * time.Millisecond)
}

func TestMainExecutionCBTransition(t *testing.T) {
	// Start our own server on 8084 to occupy it and always return 503
	crmTestSrv := &http.Server{
		Addr: ":8084",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}),
	}
	go func() {
		_ = crmTestSrv.ListenAndServe()
	}()
	defer crmTestSrv.Close()

	os.Setenv("LOG_FORMAT", "text")
	defer os.Unsetenv("LOG_FORMAT")

	go main()
	time.Sleep(400 * time.Millisecond)

	// Send 4 ingest requests to trigger 3+ consecutive CRM failures and open the CB
	client := &http.Client{Timeout: 1 * time.Second}
	for i := 0; i < 4; i++ {
		ingestBody := []byte(`{"event_id": "ev_cb", "user_id": "usr_cb", "event_type": "signup", "ip_address": "1.1.1.1", "user_agent": "Mozilla", "payload": {"email": "cb@example.com"}, "gdpr_consent": true}`)
		req, _ := http.NewRequest("POST", "http://localhost:8081/v1/events", bytes.NewBuffer(ingestBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Send SIGINT to own process to trigger graceful shutdown
	pid := os.Getpid()
	_ = syscall.Kill(pid, syscall.SIGINT)
	time.Sleep(400 * time.Millisecond)
}

func TestMainExecutionJSON(t *testing.T) {
	os.Setenv("LOG_FORMAT", "json")
	defer os.Unsetenv("LOG_FORMAT")

	go main()
	time.Sleep(400 * time.Millisecond)

	pid := os.Getpid()
	_ = syscall.Kill(pid, syscall.SIGINT)
	time.Sleep(400 * time.Millisecond)
}
