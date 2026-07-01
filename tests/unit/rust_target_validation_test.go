package unit_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"one-cli/internal/model"
	"one-cli/internal/render"
)

func TestRenderRustProjectWithMCPStdioGroup(t *testing.T) {
	dir := t.TempDir()
	app := model.App{
		Name:    "testcli",
		Version: "0.0.1",
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
	if !contains(string(cargo), `version = "0.0.1"`) {
		t.Fatalf("expected Cargo.toml to include app version, got:\n%s", string(cargo))
	}

	readme, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	for _, want := range []string{
		"## Version",
		"Generated Rust CLIs use the package version `0.0.1` from `Cargo.toml` for `--version`.",
		"target/release/testcli --version",
	} {
		if !strings.Contains(string(readme), want) {
			t.Fatalf("generated Rust README missing %q\n%s", want, string(readme))
		}
	}
	if strings.Contains(string(readme), "OPENCLI_VERSION") {
		t.Fatalf("generated Rust README should not mention build-time version override:\n%s", string(readme))
	}

	cli, err := os.ReadFile(filepath.Join(dir, "src", "cli.rs"))
	if err != nil {
		t.Fatalf("read src/cli.rs: %v", err)
	}
	for _, want := range []string{
		`env!("CARGO_PKG_VERSION")`,
		`println!("testcli version {}", version())`,
		`testcli CLI`,
		`USAGE:`,
		`testcli [options] [command]`,
		`EXAMPLES:`,
		`testcli deepwiki read-wiki`,
		`More help: testcli <command> --help`,
	} {
		if !strings.Contains(string(cli), want) {
			t.Fatalf("generated Rust CLI missing %q\n%s", want, string(cli))
		}
	}
	if strings.Contains(string(cli), "OPENCLI_VERSION") {
		t.Fatalf("generated Rust CLI should not support OPENCLI_VERSION override:\n%s", string(cli))
	}
}
