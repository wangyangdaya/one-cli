package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Config is the normalized MCP configuration.
type Config struct {
	Servers map[string]ServerConfig `json:"servers"`
}

// ServerConfig describes a single MCP server.
type ServerConfig struct {
	Transport string            `json:"transport"`
	URL       string            `json:"url,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
}

// rawConfig captures both "servers" and "mcpServers" top-level keys,
// and each server entry accepts both "transport" and "type" fields.
type rawConfig struct {
	Servers    map[string]rawServerConfig `json:"servers"`
	MCPServers map[string]rawServerConfig `json:"mcpServers"`
}

type rawServerConfig struct {
	Transport string            `json:"transport"`
	Type      string            `json:"type"`
	URL       string            `json:"url,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
}

func LoadConfig(raw []byte) (Config, error) {
	var rc rawConfig
	if err := json.Unmarshal(raw, &rc); err != nil {
		return Config{}, err
	}

	// Merge: prefer "servers", fall back to "mcpServers".
	merged := rc.Servers
	if len(merged) == 0 {
		merged = rc.MCPServers
	}
	if len(merged) == 0 {
		return Config{}, fmt.Errorf("mcp config must define at least one server")
	}

	cfg := Config{Servers: make(map[string]ServerConfig, len(merged))}
	for name, raw := range merged {
		server := normalizeServerConfig(raw)
		server = expandServerConfig(server)

		switch strings.TrimSpace(server.Transport) {
		case "streamable_http":
			if strings.TrimSpace(server.URL) == "" {
				return Config{}, fmt.Errorf("server %q requires url for streamable_http transport", name)
			}
		case "stdio":
			if strings.TrimSpace(server.Command) == "" {
				return Config{}, fmt.Errorf("server %q requires command for stdio transport", name)
			}
		default:
			return Config{}, fmt.Errorf("server %q uses unsupported transport %q", name, server.Transport)
		}
		cfg.Servers[name] = server
	}
	return cfg, nil
}

// normalizeServerConfig converts a rawServerConfig into a ServerConfig,
// resolving the transport from either "transport" or "type" fields, and
// inferring stdio when command is present but no transport is specified.
func normalizeServerConfig(raw rawServerConfig) ServerConfig {
	transport := strings.TrimSpace(raw.Transport)
	if transport == "" {
		transport = strings.TrimSpace(raw.Type)
	}

	// Normalize transport aliases.
	switch transport {
	case "sse":
		// SSE endpoints typically also support streamable HTTP.
		transport = "streamable_http"
	case "studio":
		// "studio" is an IDE-specific marker; the actual transport is stdio.
		transport = "stdio"
	}

	// Infer transport when not specified.
	if transport == "" {
		if strings.TrimSpace(raw.Command) != "" {
			transport = "stdio"
		} else if strings.TrimSpace(raw.URL) != "" {
			transport = "streamable_http"
		}
	}

	return ServerConfig{
		Transport: transport,
		URL:       raw.URL,
		Headers:   raw.Headers,
		Command:   raw.Command,
		Args:      raw.Args,
		Env:       raw.Env,
	}
}

func expandServerConfig(server ServerConfig) ServerConfig {
	server.Transport = os.ExpandEnv(server.Transport)
	server.URL = os.ExpandEnv(server.URL)
	server.Command = os.ExpandEnv(server.Command)

	for i, arg := range server.Args {
		server.Args[i] = os.ExpandEnv(arg)
	}
	for key, value := range server.Headers {
		server.Headers[key] = os.ExpandEnv(value)
	}
	for key, value := range server.Env {
		server.Env[key] = os.ExpandEnv(value)
	}

	return server
}
