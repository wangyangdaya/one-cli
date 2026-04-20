package app

import "github.com/spf13/cobra"

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "opencli",
		Short:         "Generate Go CLI projects from Swagger/OpenAPI",
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
