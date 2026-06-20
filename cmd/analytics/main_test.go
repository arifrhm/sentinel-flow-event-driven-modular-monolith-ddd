package main

import (
	"os"
	"testing"
	"time"
)

func TestMainExecution(t *testing.T) {
	os.Setenv("PORT_ANALYTICS", "9083")
	os.Setenv("LOG_FORMAT", "json")
	defer func() {
		os.Unsetenv("PORT_ANALYTICS")
		os.Unsetenv("LOG_FORMAT")
	}()

	go main()
	time.Sleep(200 * time.Millisecond)
}

func TestMainExecutionText(t *testing.T) {
	os.Setenv("PORT_ANALYTICS", "9085")
	os.Setenv("LOG_FORMAT", "text")
	defer func() {
		os.Unsetenv("PORT_ANALYTICS")
		os.Unsetenv("LOG_FORMAT")
	}()

	go main()
	time.Sleep(200 * time.Millisecond)
}
