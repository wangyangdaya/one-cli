package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func ExecuteRoot(cmd *cobra.Command) int {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
