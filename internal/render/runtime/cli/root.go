package cli

import "github.com/spf13/cobra"

var traceSetter func(bool)

func BindTrace(setter func(bool)) {
	traceSetter = setter
}

func NewRootCommand(use, short string) *cobra.Command {
	var trace bool

	cmd := &cobra.Command{
		Use:           use,
		Short:         short,
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if traceSetter != nil {
				traceSetter(trace)
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().BoolVar(&trace, "trace", false, "Print HTTP request and response trace logs")
	return cmd
}
