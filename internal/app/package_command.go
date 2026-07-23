package app

import (
	"fmt"
	"strings"

	"one-cli/internal/bundle"
	outjson "one-cli/internal/output"

	"github.com/spf13/cobra"
)

func NewPackageCommand() *cobra.Command {
	var projectDir string
	var outputDir string
	var binaryPath string

	cmd := &cobra.Command{
		Use:   "package",
		Short: "Package a generated project as an installable grouped Skills bundle",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := bundle.Package(bundle.Options{
				ProjectDir: strings.TrimSpace(projectDir),
				OutputDir:  strings.TrimSpace(outputDir),
				BinaryPath: strings.TrimSpace(binaryPath),
			})
			if err != nil {
				return err
			}
			if !JSONEnabled(cmd) {
				return nil
			}
			rendered, err := outjson.JSONSuccess(cmd.CommandPath(), "packaged Skills bundle", map[string]any{
				"app":    result.AppName,
				"output": result.OutputDir,
				"groups": result.Groups,
			})
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), rendered)
			return err
		},
	}
	cmd.Flags().StringVar(&projectDir, "project", "", "Generated OpenCLI project directory")
	cmd.Flags().StringVar(&outputDir, "output", "", "Installable Skills bundle output directory")
	cmd.Flags().StringVar(&binaryPath, "binary", "", "Prebuilt CLI binary; omit to build the generated project")
	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("output")
	return cmd
}
