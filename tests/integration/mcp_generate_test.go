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

func TestGenerateFromMCPConfig(t *testing.T) {
	const sessionID = "session-123"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept"); got != "application/json, text/event-stream" {
			t.Fatalf("accept header = %q", got)
		}

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
					"serverInfo": map[string]any{
						"name":    "fake-search",
						"version": "1.0.0",
					},
				},
			})
		case "tools/list":
			if got := r.Header.Get("Mcp-Session-Id"); got != sessionID {
				t.Fatalf("session header = %q", got)
			}
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

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcp.json")
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

	outDir := filepath.Join(tempDir, "generated")
	if err := app.RunGenerate("", configPath, outDir, "github.com/acme/generated", "searchcli", ""); err != nil {
		t.Fatalf("run generate: %v", err)
	}

	for _, rel := range []string{
		"cmd/searchcli/main.go",
		"internal/search/command.go",
		"internal/search/service.go",
		"skills/search/SKILL.md",
		"README.md",
	} {
		if _, err := os.Stat(filepath.Join(outDir, rel)); err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
	}

	content, err := os.ReadFile(filepath.Join(outDir, "internal", "search", "command.go"))
	if err != nil {
		t.Fatalf("read command: %v", err)
	}
	text := string(content)
	for _, want := range []string{
		`cmd.Flags().StringVar(&bodyQuery, "query", "", "JSON body field: query")`,
		`Body input: --query, --data, or --file`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("generated command missing %q", want)
		}
	}

	skillContent, err := os.ReadFile(filepath.Join(outDir, "skills", "search", "SKILL.md"))
	if err != nil {
		t.Fatalf("read skill: %v", err)
	}
	for _, want := range []string{
		"`searchcli search search-tool`",
		"MCP tool",
		"`--query`",
	} {
		if !strings.Contains(string(skillContent), want) {
			t.Fatalf("generated skill missing %q\n%s", want, string(skillContent))
		}
	}

	readmeContent, err := os.ReadFile(filepath.Join(outDir, "README.md"))
	if err != nil {
		t.Fatalf("read readme: %v", err)
	}
	for _, want := range []string{
		"MCP Transport",
		"tools/call",
		"./bin/searchcli search search-tool --query",
	} {
		if !strings.Contains(string(readmeContent), want) {
			t.Fatalf("generated README missing %q\n%s", want, string(readmeContent))
		}
	}
}

func writeSSEJSONRPC(t *testing.T, w http.ResponseWriter, payload map[string]any) {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal sse payload: %v", err)
	}
	w.Header().Set("Content-Type", "text/event-stream")
	if _, err := fmt.Fprintf(w, "event: message\ndata: %s\n\n", body); err != nil {
		t.Fatalf("write sse payload: %v", err)
	}
}

func TestGeneratedMCPCLIInvokesToolOverStreamableHTTP(t *testing.T) {
	toolCalled := false
	initialized := false
	const sessionID = "runtime-session-123"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept"); got != "application/json, text/event-stream" {
			t.Fatalf("accept header = %q", got)
		}

		var request struct {
			Method string         `json:"method"`
			ID     any            `json:"id"`
			Params map[string]any `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		switch request.Method {
		case "initialize":
			initialized = true
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
			if got := r.Header.Get("Mcp-Session-Id"); got != sessionID {
				t.Fatalf("tools/list session header = %q", got)
			}
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
		case "tools/call":
			if !initialized {
				t.Fatal("expected initialize before tools/call")
			}
			if got := r.Header.Get("Mcp-Session-Id"); got != sessionID {
				t.Fatalf("tools/call session header = %q", got)
			}
			toolCalled = true
			if got := request.Params["name"]; got != "search_tool" {
				t.Fatalf("tool name = %#v", got)
			}
			args, _ := request.Params["arguments"].(map[string]any)
			if got := args["query"]; got != "golang" {
				t.Fatalf("query = %#v", got)
			}
			writeSSEJSONRPC(t, w, map[string]any{
				"jsonrpc": "2.0",
				"id":      request.ID,
				"result": map[string]any{
					"content": []any{
						map[string]any{"type": "text", "text": "ok"},
					},
				},
			})
		default:
			t.Fatalf("unexpected MCP method %q", request.Method)
		}
	}))
	defer server.Close()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcp.json")
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

	outDir := filepath.Join(tempDir, "generated")
	if err := app.RunGenerate("", configPath, outDir, "github.com/acme/generated", "searchcli", ""); err != nil {
		t.Fatalf("run generate: %v", err)
	}

	cmd := exec.Command("go", "run", "./cmd/searchcli", "search", "search-tool", "--query", "golang")
	cmd.Dir = outDir
	cmd.Env = append(os.Environ(),
		"GOCACHE="+filepath.Join(tempDir, "gocache"),
		"GOTOOLCHAIN=local",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated mcp cli failed: %v output=%s", err, out)
	}
	if !toolCalled {
		t.Fatal("expected generated MCP CLI to call tools/call")
	}
	if !strings.Contains(string(out), "ok") {
		t.Fatalf("expected tool output, got %s", out)
	}
}
