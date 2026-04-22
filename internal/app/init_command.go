package app

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize an opencli configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "opencli init: not yet implemented.")
			fmt.Fprintln(cmd.OutOrStdout(), "See https://github.com/yourusername/opencli for documentation.")
			return nil
		},
	}
}
