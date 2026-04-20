package command_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"one-cli/internal/app"
)

func TestOneLeaveHelp(t *testing.T) {
	cmd := newGoRunCommand(t, "./examples/one-leave/cmd/one-leave", "--help")
	out, err := cmd.CombinedOutput()
	if err == nil {
		if !strings.Contains(string(out), "one-leave") {
			t.Fatalf("expected help output to mention one-leave, got: %s", string(out))
		}
		return
	}

	t.Fatalf("expected help command to succeed, got error: %v, output: %s", err, string(out))
}

func TestGeneratedRootHelpIncludesTraceFlag(t *testing.T) {
	dir := t.TempDir()
	if err := app.RunGenerate(filepath.Join("..", "..", "examples", "openapi.json"), dir, "github.com/acme/generated", "openapi-cli", ""); err != nil {
		t.Fatalf("generate: %v", err)
	}

	cmd := exec.Command("./bin/openapi-cli", "--help")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GOCACHE="+filepath.Join(t.TempDir(), "gocache"),
		"GOTOOLCHAIN=local",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("help command failed: %v output=%s", err, out)
	}
	if !strings.Contains(string(out), "--trace") {
		t.Fatalf("expected --trace in root help, got %s", out)
	}
}

func newGoRunCommand(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()

	cacheDir := t.TempDir()
	cmd := exec.Command("go", append([]string{"run"}, args...)...)
	cmd.Dir = "../.."
	cmd.Env = append(cmd.Environ(),
		"GOCACHE="+filepath.Join(cacheDir, "gocache"),
		"GOTOOLCHAIN=local",
	)

	return cmd
}
