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

func TestLoadConfigAcceptsMcpServersKey(t *testing.T) {
	raw := []byte(`{
		"mcpServers": {
			"deepwiki": {
				"command": "npx",
				"args": ["-y", "mcp-deepwiki@latest"]
			}
		}
	}`)

	cfg, err := mcp.LoadConfig(raw)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := cfg.Servers["deepwiki"].Transport; got != "stdio" {
		t.Fatalf("expected inferred stdio transport, got %q", got)
	}
	if got := cfg.Servers["deepwiki"].Command; got != "npx" {
		t.Fatalf("command = %q want npx", got)
	}
}

func TestLoadConfigAcceptsTypeFieldAsTransport(t *testing.T) {
	raw := []byte(`{
		"mcpServers": {
			"fetch": {
				"type": "streamable_http",
				"url": "https://example.com/mcp"
			}
		}
	}`)

	cfg, err := mcp.LoadConfig(raw)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := cfg.Servers["fetch"].Transport; got != "streamable_http" {
		t.Fatalf("transport = %q want streamable_http", got)
	}
}

func TestLoadConfigNormalizesSSEToStreamableHTTP(t *testing.T) {
	raw := []byte(`{
		"mcpServers": {
			"fetch": {
				"type": "sse",
				"url": "https://example.com/sse"
			}
		}
	}`)

	cfg, err := mcp.LoadConfig(raw)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := cfg.Servers["fetch"].Transport; got != "streamable_http" {
		t.Fatalf("transport = %q want streamable_http (normalized from sse)", got)
	}
}

func TestLoadConfigNormalizesStudioToStdio(t *testing.T) {
	raw := []byte(`{
		"mcpServers": {
			"deepwiki": {
				"transport": "studio",
				"command": "npx",
				"args": ["-y", "mcp-deepwiki@latest"]
			}
		}
	}`)

	cfg, err := mcp.LoadConfig(raw)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := cfg.Servers["deepwiki"].Transport; got != "stdio" {
		t.Fatalf("transport = %q want stdio (normalized from studio)", got)
	}
}

func TestLoadConfigInfersTransportFromURL(t *testing.T) {
	raw := []byte(`{
		"mcpServers": {
			"remote": {
				"url": "https://example.com/mcp"
			}
		}
	}`)

	cfg, err := mcp.LoadConfig(raw)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := cfg.Servers["remote"].Transport; got != "streamable_http" {
		t.Fatalf("transport = %q want streamable_http (inferred from url)", got)
	}
}

func TestLoadConfigInfersTransportFromCommand(t *testing.T) {
	raw := []byte(`{
		"mcpServers": {
			"local": {
				"command": "python",
				"args": ["server.py"]
			}
		}
	}`)

	cfg, err := mcp.LoadConfig(raw)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := cfg.Servers["local"].Transport; got != "stdio" {
		t.Fatalf("transport = %q want stdio (inferred from command)", got)
	}
}

func TestLoadConfigPrefersServersOverMcpServers(t *testing.T) {
	raw := []byte(`{
		"servers": {
			"a": {
				"transport": "stdio",
				"command": "python",
				"args": ["a.py"]
			}
		},
		"mcpServers": {
			"b": {
				"command": "python",
				"args": ["b.py"]
			}
		}
	}`)

	cfg, err := mcp.LoadConfig(raw)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if _, ok := cfg.Servers["a"]; !ok {
		t.Fatal("expected server 'a' from 'servers' key")
	}
	if _, ok := cfg.Servers["b"]; ok {
		t.Fatal("expected 'mcpServers' to be ignored when 'servers' is present")
	}
}
