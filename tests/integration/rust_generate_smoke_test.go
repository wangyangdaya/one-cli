package integration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"one-cli/internal/app"
)

func TestGenerateRustOpenAPISmoke(t *testing.T) {
	dir := t.TempDir()
	if err := app.RunGenerate(filepath.Join("..", "..", "examples", "petstore.yaml"), "", dir, "petcli", "petcli", "", "rust"); err != nil {
		t.Fatalf("run rust generate: %v", err)
	}

	for _, rel := range []string{
		"Cargo.toml",
		"README.md",
		"skills/pet/SKILL.md",
		"src/main.rs",
		"src/cli.rs",
		"src/client.rs",
		"src/output.rs",
		"src/types.rs",
		"src/commands/mod.rs",
		"src/commands/pet.rs",
	} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
	}

	tryCargoBuild(t, dir)
}

func TestGenerateRustMCPSmoke(t *testing.T) {
	const sessionID = "session-rust-123"
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Skipf("skipping MCP smoke test in restricted environment: %v", recovered)
		}
	}()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Method string `json:"method"`
			ID     any    `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		switch request.Method {
		case "initialize":
			w.Header().Set("Mcp-Session-Id", sessionID)
			writeSSEJSONRPC(t, w, map[string]any{
				"jsonrpc": "2.0",
				"id":      request.ID,
				"result": map[string]any{
					"protocolVersion": "2025-03-26",
					"capabilities":    map[string]any{},
					"serverInfo":      map[string]any{"name": "fake-search", "version": "1.0.0"},
				},
			})
		case "tools/list":
			writeSSEJSONRPC(t, w, map[string]any{
				"jsonrpc": "2.0",
				"id":      request.ID,
				"result": map[string]any{
					"tools": []any{
						map[string]any{
							"name":        "search_tool",
							"description": "Search content",
							"inputSchema": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"query": map[string]any{"type": "string"},
								},
								"required": []any{"query"},
							},
						},
					},
				},
			})
		default:
			t.Fatalf("unexpected MCP method %q", request.Method)
		}
	}))
	defer server.Close()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "mcp.json")
	if err := os.WriteFile(configPath, []byte(fmt.Sprintf(`{
		"servers": {
			"search": {
				"transport": "streamable_http",
				"url": %q
			}
		}
	}`, server.URL)), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	outDir := filepath.Join(dir, "generated")
	if err := app.RunGenerate("", configPath, outDir, "quark", "quark", "", "rust"); err != nil {
		t.Fatalf("run rust MCP generate: %v", err)
	}

	for _, rel := range []string{
		"Cargo.toml",
		"README.md",
		"skills/search/SKILL.md",
		"src/main.rs",
		"src/client.rs",
		"src/commands/mod.rs",
		"src/commands/search.rs",
	} {
		if _, err := os.Stat(filepath.Join(outDir, rel)); err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
	}

	commandContent, err := os.ReadFile(filepath.Join(outDir, "src", "commands", "search.rs"))
	if err != nil {
		t.Fatalf("read generated MCP command: %v", err)
	}
	commandText := string(commandContent)
	for _, want := range []string{
		"call_mcp_tool",
		"search_tool",
	} {
		if !strings.Contains(commandText, want) {
			t.Fatalf("generated MCP command missing %q:\n%s", want, commandText)
		}
	}

	tryCargoBuild(t, outDir)
}

func tryCargoBuild(t *testing.T, dir string) {
	t.Helper()

	if _, err := exec.LookPath("cargo"); err != nil {
		t.Skip("cargo not installed")
	}

	cmd := exec.Command("cargo", "build")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		output := string(out)
		if strings.Contains(output, "Could not resolve host: index.crates.io") ||
			strings.Contains(output, "failed to download from `https://index.crates.io/") {
			t.Skipf("cargo build skipped due to network restrictions:\n%s", output)
		}
		t.Fatalf("cargo build failed: %v\n%s", err, string(out))
	}
}
