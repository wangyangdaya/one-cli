package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func ExecuteRoot(cmd *cobra.Command) int {
	if err := cmd.Execute(); err != nil {
		if errors.Is(err, errVersionPrinted) {
			return 0
		}
		if JSONEnabled(cmd) {
			rendered, renderErr := JSONError(cmd.CommandPath(), "command_error", err.Error())
			if renderErr == nil {
				_, _ = fmt.Fprintln(os.Stderr, rendered)
				return 1
			}
		}
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
