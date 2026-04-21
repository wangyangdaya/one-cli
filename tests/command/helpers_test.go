package command_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func newGoRunCommand(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()

	cacheDir := t.TempDir()
	cmd := exec.Command("go", append([]string{"run"}, args...)...)
	cmd.Dir = "../.."
	cmd.Env = append(os.Environ(),
		"GOCACHE="+filepath.Join(cacheDir, "gocache"),
		"GOTOOLCHAIN=local",
	)

	return cmd
}
