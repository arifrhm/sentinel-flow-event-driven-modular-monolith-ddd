package main

import (
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"

	"sentinel-flow/pkg/broker"
	"sentinel-flow/pkg/mockdb"
)

func TestMainExitDB(t *testing.T) {
	if os.Getenv("BE_CRASHER_DB") == "1" {
		os.Setenv("DATABASE_TYPE", "postgres")
		os.Setenv("DATABASE_URL", "postgres://invalid:invalid@localhost:9999/invalid")
		main()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestMainExitDB")
	cmd.Env = append(os.Environ(), "BE_CRASHER_DB=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}

func TestMainExitBroker(t *testing.T) {
	if os.Getenv("BE_CRASHER_BROKER") == "1" {
		os.Setenv("DATABASE_TYPE", "memory")
		os.Setenv("BROKER_TYPE", "redis")
		os.Setenv("REDIS_URL", "redis://localhost:9999")
		broker.PingRetries = 1
		broker.PingDelay = 1 * time.Millisecond
		main()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestMainExitBroker")
	cmd.Env = append(os.Environ(), "BE_CRASHER_BROKER=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}

func TestMainExitMigrationError(t *testing.T) {
	if os.Getenv("BE_CRASHER_MIG") == "1" {
		os.Setenv("DATABASE_TYPE", "postgres-mock")
		main()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestMainExitMigrationError")
	cmd.Env = append(os.Environ(), "BE_CRASHER_MIG=1")
	cmd.Dir = os.TempDir()
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}

func TestMainExitMigrationExecError(t *testing.T) {
	if os.Getenv("BE_CRASHER_MIG_EXEC") == "1" {
		os.Setenv("DATABASE_TYPE", "postgres-mock")
		mockdb.SimError = errors.New("exec migration failed")
		main()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestMainExitMigrationExecError")
	cmd.Env = append(os.Environ(), "BE_CRASHER_MIG_EXEC=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}

func TestMainExitDBOpenError(t *testing.T) {
	if os.Getenv("BE_CRASHER_DB_OPEN") == "1" {
		os.Setenv("DATABASE_TYPE", "postgres")
		main()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestMainExitDBOpenError")
	cmd.Env = append(os.Environ(), "BE_CRASHER_DB_OPEN=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}
