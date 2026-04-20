package app

import (
	"errors"

	"github.com/spf13/cobra"
)

func NewInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize an opencli configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("init not implemented")
		},
	}
}
