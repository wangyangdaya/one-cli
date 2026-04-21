package unit_test

import (
	"strings"
	"testing"

	"one-cli/internal/mcp"
)

func TestLoadConfigAcceptsStreamableHTTPAndStdio(t *testing.T) {
	raw := []byte(`{
		"servers": {
			"remote": {
				"transport": "streamable_http",
				"url": "https://example.com/mcp",
				"headers": {"Authorization": "Bearer token"}
			},
			"local": {
				"transport": "stdio",
				"command": "python",
				"args": ["server.py"],
				"env": {"DEBUG": "1"}
			}
		}
	}`)

	cfg, err := mcp.LoadConfig(raw)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := cfg.Servers["remote"].Transport; got != "streamable_http" {
		t.Fatalf("remote transport = %q", got)
	}
	if got := cfg.Servers["local"].Command; got != "python" {
		t.Fatalf("local command = %q", got)
	}
}

func TestLoadConfigRejectsMissingTransportFields(t *testing.T) {
	raw := []byte(`{
		"servers": {
			"broken": {
				"transport": "streamable_http"
			}
		}
	}`)

	_, err := mcp.LoadConfig(raw)
	if err == nil || !strings.Contains(err.Error(), "broken") {
		t.Fatalf("expected server-scoped validation error, got %v", err)
	}
}
