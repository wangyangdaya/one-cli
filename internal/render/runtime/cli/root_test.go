package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewRootCommandVersionFlagSupportsJSON(t *testing.T) {
	cmd := NewRootCommand("petcli", "petcli CLI", "0.1.0")
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"-v", "--json"})

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
	for _, want := range []string{"-H, --header", "--json", "--trace", "-v, --version"} {
		if count := strings.Count(help, want); count != 1 {
			t.Fatalf("%q appears %d times in help output:\n%s", want, count, help)
		}
	}
}

func TestApplyRootHelpExamplesSkipSkillsCommand(t *testing.T) {
	originalSorting := cobra.EnableCommandSorting
	cobra.EnableCommandSorting = false
	defer func() { cobra.EnableCommandSorting = originalSorting }()

	cmd := NewRootCommand("petcli", "petcli CLI", "0.1.0")
	pet := &cobra.Command{Use: "pet", Short: "Pet operations"}
	pet.AddCommand(&cobra.Command{Use: "list", Short: "GET /pets", Run: func(cmd *cobra.Command, args []string) {}})
	cmd.AddCommand(&cobra.Command{Use: "login", Short: "OAuth login", Run: func(cmd *cobra.Command, args []string) {}})
	cmd.AddCommand(pet)
	cmd.AddCommand(NewSkillsCommand())

	ApplyRootHelp(cmd)

	if got := cmd.Example; strings.Contains(got, "skills") {
		t.Fatalf("root examples should skip skills command, got:\n%s", got)
	}
	if got := cmd.Example; strings.Contains(got, "login") {
		t.Fatalf("root examples should skip top-level non-group commands, got:\n%s", got)
	}
	if got := strings.TrimSpace(cmd.Example); got != "petcli pet list" {
		t.Fatalf("root examples = %q, want petcli pet list", got)
	}
}
