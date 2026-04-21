package unit_test

import (
	"path/filepath"
	"strings"
	"testing"

	"one-cli/internal/app"
)

func TestRunGenerateRejectsRustMCPStdio(t *testing.T) {
	err := app.RunGenerate(
		"",
		filepath.Join("..", "command", "testdata", "mcp_stdio.json"),
		t.TempDir(),
		"petcli",
		"petcli",
		"",
		"rust",
	)
	if err == nil || !strings.Contains(err.Error(), "streamable_http") {
		t.Fatalf("expected MCP transport error, got %v", err)
	}
}
