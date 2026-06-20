package main

import (
	"os"
	"os/exec"
	"testing"
)

func TestMainExit(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		os.Setenv("PORT_INGEST", "-1")
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
