package app

import "github.com/spf13/cobra"

func JSONEnabled(cmd *cobra.Command) bool {
	enabled, err := cmd.Root().PersistentFlags().GetBool("json")
	return err == nil && enabled
}
