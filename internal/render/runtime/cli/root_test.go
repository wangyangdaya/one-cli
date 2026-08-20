package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewRootCommandVersionFlagSupportsJSON(t *testing.T) {
	cmd := NewRootCommand("petcli", "petcli CLI", "0.1.0")
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"--version", "--json"})

	if code := ExecuteRoot(cmd); code != 0 {
		t.Fatalf("ExecuteRoot() = %d, want 0", code)
	}
	if got := strings.TrimSpace(stdout.String()); got != `{"ok":true,"command":"petcli version","message":"ok","data":{"version":"0.1.0"}}` {
		t.Fatalf("version output = %q", got)
	}
}

func TestRootHelpRendersFlagsOnce(t *testing.T) {
	cmd := NewRootCommand("petcli", "petcli CLI", "0.1.0")
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	ApplyRootHelp(cmd)

	if code := ExecuteRoot(cmd); code != 0 {
		t.Fatalf("ExecuteRoot() = %d, want 0", code)
	}
	help := stdout.String()
	if strings.Contains(help, "-v, --version") {
		t.Fatalf("help output unexpectedly contains short version flag:\n%s", help)
	}
	for _, want := range []string{"-H, --header", "--json", "--trace", "--version"} {
		if count := strings.Count(help, want); count != 1 {
			t.Fatalf("%q appears %d times in help output:\n%s", want, count, help)
		}
	}
}
