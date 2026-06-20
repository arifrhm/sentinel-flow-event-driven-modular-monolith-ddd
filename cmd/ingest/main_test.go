package main

import (
	"os"
	"testing"
	"time"
)

func TestMainExecution(t *testing.T) {
	os.Setenv("PORT_INGEST", "9086")
	os.Setenv("LOG_FORMAT", "json")
	defer func() {
		os.Unsetenv("PORT_INGEST")
		os.Unsetenv("LOG_FORMAT")
	}()

	go main()
	time.Sleep(200 * time.Millisecond)
}

func TestMainExecutionText(t *testing.T) {
	os.Setenv("PORT_INGEST", "9087")
	os.Setenv("LOG_FORMAT", "text")
	defer func() {
		os.Unsetenv("PORT_INGEST")
		os.Unsetenv("LOG_FORMAT")
	}()

	go main()
	time.Sleep(200 * time.Millisecond)
}
