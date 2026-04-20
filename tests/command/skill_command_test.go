package command_test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOneLeaveSkill(t *testing.T) {
	cmd := newGoRunCommand(t, "./examples/one-leave/cmd/one-leave", "skill")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected skill command to succeed, got error: %v, output: %s", err, string(out))
	}

	expected := readFile(t, filepath.Join("..", "..", "examples", "one-leave", "skills", "leave", "SKILL.md"))
	if string(out) != expected {
		t.Fatalf("leave skill output mismatch\nexpected:\n%s\nactual:\n%s", expected, string(out))
	}
}
func readFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return string(data)
}
