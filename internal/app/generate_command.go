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
	var appVersion string
	var configPath string
	var target string
	var skillLang string

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate a Go CLI project from Swagger/OpenAPI or MCP",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunGenerateWithVersionAndSkillLang(input, mcpConfig, output, module, appName, appVersion, configPath, skillLang, target)
		},
	}

	cmd.Flags().StringVar(&input, "input", "", "Path or URL to the OpenAPI document")
	cmd.Flags().StringVar(&mcpConfig, "mcp-config", "", "Path to the MCP server config file")
	cmd.Flags().StringVar(&output, "output", "", "Output directory")
	cmd.Flags().StringVar(&module, "module", "", "Go module path for the generated project")
	cmd.Flags().StringVar(&appName, "app", "", "Binary/root command name for the generated project")
	cmd.Flags().StringVar(&appVersion, "app-version", "", "Version for the generated CLI project")
	cmd.Flags().StringVar(&configPath, "config", "", "Path to opencli YAML config")
	cmd.Flags().StringVar(&target, "target", "go", "Generation target: go or rust")
	cmd.Flags().StringVar(&skillLang, "skill-lang", "en", "Generated skill language: en or zh")
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

func RunGenerate(input, mcpConfig, output, module, appName, configPath string, targets ...string) error {
	return RunGenerateWithVersion(input, mcpConfig, output, module, appName, "", configPath, targets...)
}

func RunGenerateWithVersion(input, mcpConfig, output, module, appName, appVersion, configPath string, targets ...string) error {
	return RunGenerateWithVersionAndSkillLang(input, mcpConfig, output, module, appName, appVersion, configPath, "en", targets...)
}

func RunGenerateWithVersionAndSkillLang(input, mcpConfig, output, module, appName, appVersion, configPath, skillLang string, targets ...string) error {
	if err := validateGenerateSources(input, mcpConfig); err != nil {
		return err
	}

	cfg, err := configgen.Load(strings.TrimSpace(configPath))
	if err != nil {
		return err
	}
	if trimmed := strings.TrimSpace(appVersion); trimmed != "" {
		cfg.App.Version = trimmed
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

	plan := planner.Build(doc, cfg)
	plan.Name = strings.TrimSpace(appName)
	target := "go"
	if len(targets) > 0 {
		target = strings.TrimSpace(targets[0])
	}
	return render.Project(strings.TrimSpace(output), strings.TrimSpace(module), plan, target, strings.TrimSpace(skillLang))
}
