package main

import (
	"os"
	"testing"
	"time"
)

func TestMainExecution(t *testing.T) {
	os.Setenv("LOG_FORMAT", "json")
	defer os.Unsetenv("LOG_FORMAT")

	go main()
	time.Sleep(200 * time.Millisecond)
}

func TestMainExecutionText(t *testing.T) {
	os.Setenv("LOG_FORMAT", "text")
	defer os.Unsetenv("LOG_FORMAT")

	go main()
	time.Sleep(200 * time.Millisecond)
}
