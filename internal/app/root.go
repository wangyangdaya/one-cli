package app

import "github.com/spf13/cobra"

// Version is set at build time via -ldflags.
var Version = "dev"

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "opencli",
		Short:         "Generate CLI projects from OpenAPI/Swagger or MCP (Go & Rust)",
		Version:       Version,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(NewInitCommand())
	cmd.AddCommand(NewInspectCommand())
	cmd.AddCommand(NewGenerateCommand())

	return cmd
}
