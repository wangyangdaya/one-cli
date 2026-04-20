package app

import (
	"strings"

	"one-cli/internal/configgen"
	"one-cli/internal/loaders"
	"one-cli/internal/openapi"
	"one-cli/internal/planner"
	"one-cli/internal/render"

	"github.com/spf13/cobra"
)

func NewGenerateCommand() *cobra.Command {
	var input string
	var output string
	var module string
	var appName string
	var configPath string

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate a Go CLI project from Swagger/OpenAPI",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunGenerate(input, output, module, appName, configPath)
		},
	}

	cmd.Flags().StringVar(&input, "input", "", "Path or URL to the OpenAPI document")
	cmd.Flags().StringVar(&output, "output", "", "Output directory")
	cmd.Flags().StringVar(&module, "module", "", "Go module path for the generated project")
	cmd.Flags().StringVar(&appName, "app", "", "Binary/root command name for the generated project")
	cmd.Flags().StringVar(&configPath, "config", "", "Path to opencli YAML config")
	_ = cmd.MarkFlagRequired("input")
	_ = cmd.MarkFlagRequired("output")
	_ = cmd.MarkFlagRequired("module")
	_ = cmd.MarkFlagRequired("app")
	return cmd
}

func RunGenerate(input, output, module, appName, configPath string) error {
	cfg, err := configgen.Load(strings.TrimSpace(configPath))
	if err != nil {
		return err
	}

	raw, err := loaders.Load(strings.TrimSpace(input))
	if err != nil {
		return err
	}

	doc, err := openapi.Parse(raw)
	if err != nil {
		return err
	}

	plan, err := planner.Build(doc, cfg)
	if err != nil {
		return err
	}
	plan.Name = strings.TrimSpace(appName)
	return render.Project(strings.TrimSpace(output), strings.TrimSpace(module), plan)
}
