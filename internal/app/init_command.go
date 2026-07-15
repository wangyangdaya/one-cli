package app

import (
	"fmt"

	outjson "one-cli/internal/output"

	"github.com/spf13/cobra"
)

func NewInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize an opencli configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			if JSONEnabled(cmd) {
				rendered, err := outjson.JSONSuccess(cmd.CommandPath(), "init is not yet implemented", map[string]bool{
					"implemented": false,
				})
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(cmd.OutOrStdout(), rendered)
				return err
			}
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
