package mcp

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"one-cli/internal/openapi"
)

func DiscoverDocument(configPath string) (openapi.Document, error) {
	raw, err := os.ReadFile(strings.TrimSpace(configPath))
	if err != nil {
		return openapi.Document{}, fmt.Errorf("read mcp config: %w", err)
	}

	cfg, err := LoadConfig(raw)
	if err != nil {
		return openapi.Document{}, err
	}

	doc := openapi.Document{Title: "mcp"}
	serverNames := make([]string, 0, len(cfg.Servers))
	for name := range cfg.Servers {
		serverNames = append(serverNames, name)
	}
	sort.Strings(serverNames)

	for _, name := range serverNames {
		tools, err := discoverTools(name, cfg.Servers[name])
		if err != nil {
			return openapi.Document{}, err
		}
		serverDoc, err := ConvertServer(name, cfg.Servers[name], tools)
		if err != nil {
			return openapi.Document{}, err
		}
		doc.Tags = append(doc.Tags, serverDoc.Tags...)
		doc.Operations = append(doc.Operations, serverDoc.Operations...)
	}

	return doc, nil
}

func discoverTools(name string, server ServerConfig) ([]Tool, error) {
	switch strings.TrimSpace(server.Transport) {
	case "streamable_http":
		return discoverHTTPTools(name, server)
	case "stdio":
		return discoverStdioTools(name, server)
	default:
		return nil, fmt.Errorf("server %q uses unsupported transport %q", name, server.Transport)
	}
}
