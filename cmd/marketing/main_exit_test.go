package main

import (
	"os"
	"os/exec"
	"testing"
)

func TestMainExit(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		os.Setenv("PORT_MARKETING", "-1")
		main()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestMainExit")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}

func TestMainExitConsumer(t *testing.T) {
	if os.Getenv("BE_CRASHER_CONSUMER") == "1" {
		main()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestMainExitConsumer")
	cmd.Env = append(os.Environ(), "BE_CRASHER_CONSUMER=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}

func TestMainExitCB(t *testing.T) {
	if os.Getenv("BE_CRASHER_CB") == "1" {
		main()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestMainExitCB")
	cmd.Env = append(os.Environ(), "BE_CRASHER_CB=1", "PORT_MARKETING=-1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}
