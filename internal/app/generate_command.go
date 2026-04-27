package app

import (
	"fmt"
	"strings"

	"one-cli/internal/configgen"
	"one-cli/internal/loaders"
	"one-cli/internal/mcp"
	"one-cli/internal/openapi"
	"one-cli/internal/planner"
	"one-cli/internal/render"

	"github.com/spf13/cobra"
)

func NewGenerateCommand() *cobra.Command {
	var input string
	var mcpConfig string
	var output string
	var module string
	var appName string
	var configPath string
	var target string

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate a Go CLI project from Swagger/OpenAPI or MCP",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunGenerate(input, mcpConfig, output, module, appName, configPath, target)
		},
	}

	cmd.Flags().StringVar(&input, "input", "", "Path or URL to the OpenAPI document")
	cmd.Flags().StringVar(&mcpConfig, "mcp-config", "", "Path to the MCP server config file")
	cmd.Flags().StringVar(&output, "output", "", "Output directory")
	cmd.Flags().StringVar(&module, "module", "", "Go module path for the generated project")
	cmd.Flags().StringVar(&appName, "app", "", "Binary/root command name for the generated project")
	cmd.Flags().StringVar(&configPath, "config", "", "Path to opencli YAML config")
	cmd.Flags().StringVar(&target, "target", "go", "Generation target: go or rust")
	_ = cmd.MarkFlagRequired("output")
	_ = cmd.MarkFlagRequired("module")
	_ = cmd.MarkFlagRequired("app")
	return cmd
}

func validateGenerateSources(input, mcpConfig string) error {
	hasInput := strings.TrimSpace(input) != ""
	hasMCPConfig := strings.TrimSpace(mcpConfig) != ""
	if hasInput == hasMCPConfig {
		return fmt.Errorf("exactly one of --input or --mcp-config is required")
	}
	return nil
}

func normalizeTarget(values []string) (string, error) {
	target := "go"
	if len(values) > 0 {
		target = strings.TrimSpace(values[0])
	}

	switch strings.ToLower(target) {
	case "", "go":
		return "go", nil
	case "rust":
		return "rust", nil
	default:
		return "", fmt.Errorf("unsupported target %q: expected go or rust", target)
	}
}

func validateRustMCPConfig(path string) error {
	raw, err := loaders.Load(strings.TrimSpace(path))
	if err != nil {
		return err
	}

	cfg, err := mcp.LoadConfig(raw)
	if err != nil {
		return err
	}

	for name, server := range cfg.Servers {
		if strings.TrimSpace(server.Transport) != "streamable_http" {
			return fmt.Errorf("rust target only supports MCP streamable_http transport; server %q uses %q", name, server.Transport)
		}
	}

	return nil
}

func RunGenerate(input, mcpConfig, output, module, appName, configPath string, targets ...string) error {
	target, err := normalizeTarget(targets)
	if err != nil {
		return err
	}

	if err := validateGenerateSources(input, mcpConfig); err != nil {
		return err
	}

	if target == "rust" && strings.TrimSpace(mcpConfig) != "" {
		// Rust target now supports both streamable_http and stdio MCP transports.
	}

	cfg, err := configgen.Load(strings.TrimSpace(configPath))
	if err != nil {
		return err
	}

	var doc openapi.Document
	if strings.TrimSpace(mcpConfig) != "" {
		doc, err = mcp.DiscoverDocument(strings.TrimSpace(mcpConfig))
		if err != nil {
			return err
		}
	} else {
		raw, err := loaders.Load(strings.TrimSpace(input))
		if err != nil {
			return err
		}

		doc, err = openapi.Parse(raw)
		if err != nil {
			return err
		}
	}

	plan, err := planner.Build(doc, cfg)
	if err != nil {
		return err
	}
	plan.Name = strings.TrimSpace(appName)
	return render.Project(strings.TrimSpace(output), strings.TrimSpace(module), plan, target)
}
