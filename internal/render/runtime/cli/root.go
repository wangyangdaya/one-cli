package cli

import "github.com/spf13/cobra"

var traceSetter func(bool)
var outputFormat string

func BindTrace(setter func(bool)) {
	traceSetter = setter
}

// OutputFormat returns the current output format ("json" or "text").
func OutputFormat() string {
	if outputFormat == "json" {
		return "json"
	}
	return "text"
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
	cmd.PersistentFlags().StringVar(&outputFormat, "output", "text", "Output format: text or json")
	return cmd
}
