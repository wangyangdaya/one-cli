package app

import (
	"fmt"
	"os"

	"one-cli/internal/output"

	"github.com/spf13/cobra"
)

func ExecuteRoot(cmd *cobra.Command) int {
	if err := cmd.Execute(); err != nil {
		if JSONEnabled(cmd) {
			rendered, renderErr := output.JSONError(cmd.CommandPath(), "command_error", err.Error())
			if renderErr == nil {
				fmt.Fprintln(os.Stderr, rendered)
				return 1
			}
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}
