package main

import (
	"bytes"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestMainExecution(t *testing.T) {
	os.Setenv("PORT_MARKETING", "29082")
	os.Setenv("LOG_FORMAT", "text")
	defer func() {
		os.Unsetenv("PORT_MARKETING")
		os.Unsetenv("LOG_FORMAT")
	}()

	go main()
	time.Sleep(300 * time.Millisecond)

	// Call GDPR delete endpoint to trigger main's deleter callback
	body := []byte(`{"user_id":"test_deleter_usr"}`)
	resp, err := http.Post("http://localhost:29082/v1/privacy/gdpr-delete", "application/json", bytes.NewBuffer(body))
	if err == nil {
		resp.Body.Close()
	}
}

func TestMainExecutionJSON(t *testing.T) {
	os.Setenv("PORT_MARKETING", "29083")
	os.Setenv("LOG_FORMAT", "json")
	defer func() {
		os.Unsetenv("PORT_MARKETING")
		os.Unsetenv("LOG_FORMAT")
	}()

	go main()
	time.Sleep(300 * time.Millisecond)
}
