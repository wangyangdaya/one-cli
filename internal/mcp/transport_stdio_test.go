package mcp

import (
	"io"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestShutdownStdioProcessAllowsExitAfterEOF(t *testing.T) {
	cmd, stdin := startStdioShutdownHelper(t, "exit")

	shutdownStdioProcess(stdin, cmd, 100*time.Millisecond)

	if cmd.ProcessState == nil || !cmd.ProcessState.Success() {
		t.Fatalf("process state = %v, want successful exit after stdin EOF", cmd.ProcessState)
	}
}

func TestShutdownStdioProcessEscalatesWhenServerDoesNotExit(t *testing.T) {
	cmd, stdin := startStdioShutdownHelper(t, "linger")
	started := time.Now()

	shutdownStdioProcess(stdin, cmd, 25*time.Millisecond)

	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("shutdown took %v, want bounded escalation", elapsed)
	}
	if cmd.ProcessState == nil || cmd.ProcessState.Success() {
		t.Fatalf("process state = %v, want forced non-successful exit", cmd.ProcessState)
	}
}

func startStdioShutdownHelper(t *testing.T, mode string) (*exec.Cmd, io.WriteCloser) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestStdioShutdownHelperProcess$")
	cmd.Env = append(os.Environ(), "OPENCLI_STDIO_HELPER_MODE="+mode)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start helper: %v", err)
	}
	return cmd, stdin
}

func TestStdioShutdownHelperProcess(t *testing.T) {
	mode := os.Getenv("OPENCLI_STDIO_HELPER_MODE")
	if mode == "" {
		return
	}
	_, _ = io.Copy(io.Discard, os.Stdin)
	if mode == "linger" {
		time.Sleep(10 * time.Second)
	}
}
