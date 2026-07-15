package cli

import (
	"strings"

	"github.com/spf13/cobra"
)

var traceSetter func(bool)
var requestHeaders []string

func BindTrace(setter func(bool)) {
	traceSetter = setter
}

func NewRootCommand(use, short, version string) *cobra.Command {
	var trace bool
	requestHeaders = nil

	cmd := &cobra.Command{
		Use:           use,
		Short:         short,
		Version:       version,
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

	cmd.CompletionOptions.DisableDefaultCmd = true
	cmd.PersistentFlags().StringArrayVarP(&requestHeaders, "header", "H", nil, "Request header in 'Name: Value' or 'Name=Value' format; repeatable")
	cmd.PersistentFlags().BoolVar(&trace, "trace", false, "Print HTTP request and response trace logs")
	cmd.PersistentFlags().Bool("json", false, "Print command output as JSON")
	return cmd
}

func RequestHeaders() []string {
	if len(requestHeaders) == 0 {
		return nil
	}
	return append([]string(nil), requestHeaders...)
}

func ApplyRootHelp(cmd *cobra.Command) {
	if strings.TrimSpace(cmd.Example) == "" {
		cmd.Example = rootExamples(cmd)
	}
	cmd.SetHelpTemplate(rootHelpTemplate)
}

func rootExamples(cmd *cobra.Command) string {
	root := firstWord(cmd.Use)
	if root == "" {
		root = cmd.Name()
	}

	examples := make([]string, 0, 3)
	for _, child := range cmd.Commands() {
		if !child.IsAvailableCommand() {
			continue
		}
		command := firstWord(child.Use)
		if command == "" {
			command = child.Name()
		}

		subcommand := ""
		for _, grandchild := range child.Commands() {
			if !grandchild.IsAvailableCommand() {
				continue
			}
			subcommand = firstWord(grandchild.Use)
			if subcommand == "" {
				subcommand = grandchild.Name()
			}
			break
		}

		if subcommand != "" {
			examples = append(examples, "  "+root+" "+command+" "+subcommand)
		} else {
			examples = append(examples, "  "+root+" "+command)
		}
		if len(examples) == 3 {
			break
		}
	}

	if len(examples) == 0 {
		return "  " + root + " --help"
	}
	return strings.Join(examples, "\n")
}

func firstWord(value string) string {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

const rootHelpTemplate = `{{with (or .Long .Short)}}{{.}}

{{end}}USAGE:
  {{.CommandPath}} [options] [command]

{{if .Example}}EXAMPLES:
{{.Example}}

{{end}}{{if .HasAvailableSubCommands}}Available Commands:
{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}  {{rpad .Name .NamePadding }} {{.Short}}
{{end}}{{end}}
{{end}}Options:
{{if .HasAvailableLocalFlags}}{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}
{{end}}{{if .HasAvailablePersistentFlags}}{{.PersistentFlags.FlagUsages | trimTrailingWhitespaces}}
{{end}}{{if .HasAvailableInheritedFlags}}{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}
{{end}}
More help: {{.CommandPath}} <command> --help
`
