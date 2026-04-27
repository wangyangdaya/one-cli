package unit_test

import (
	"os"
	"path/filepath"
	"testing"

	"one-cli/internal/model"
	"one-cli/internal/render"
)

func TestRenderRustProjectWithMCPStdioGroup(t *testing.T) {
	dir := t.TempDir()
	app := model.App{
		Name: "testcli",
		Groups: []model.Group{
			{
				Name:        "deepwiki",
				PackageName: "deepwiki",
				Backend:     "mcp-stdio",
				Command:     "npx",
				Args:        []string{"-y", "mcp-deepwiki@latest"},
				Operations: []model.Operation{
					{
						CommandName: "read-wiki",
						RemoteName:  "read_wiki_structure",
						Method:      "MCP",
						Path:        "/read_wiki_structure",
						Summary:     "Read wiki structure",
					},
				},
			},
		},
	}

	if err := render.Project(dir, "testcli", app, "rust"); err != nil {
		t.Fatalf("render rust project with MCP stdio: %v", err)
	}

	// Verify key files exist
	for _, path := range []string{"Cargo.toml", "src/main.rs", "src/client.rs", "src/commands/deepwiki.rs"} {
		if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
			t.Fatalf("expected %s in output, got %v", path, err)
		}
	}

	// Verify Cargo.toml includes process feature for tokio
	cargo, err := os.ReadFile(filepath.Join(dir, "Cargo.toml"))
	if err != nil {
		t.Fatalf("read Cargo.toml: %v", err)
	}
	if !contains(string(cargo), "process") {
		t.Fatal("expected Cargo.toml to include tokio process feature for MCP stdio")
	}
}
