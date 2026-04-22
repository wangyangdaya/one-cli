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
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), "opencli init: not yet implemented."); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), "See https://github.com/yourusername/opencli for documentation."); err != nil {
				return err
			}
			return nil
		},
	}
}
