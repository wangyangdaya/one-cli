package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Servers map[string]ServerConfig `json:"servers"`
}

type ServerConfig struct {
	Transport string            `json:"transport"`
	URL       string            `json:"url,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
}

func LoadConfig(raw []byte) (Config, error) {
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, err
	}
	if len(cfg.Servers) == 0 {
		return Config{}, fmt.Errorf("mcp config must define at least one server")
	}
	for name, server := range cfg.Servers {
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
