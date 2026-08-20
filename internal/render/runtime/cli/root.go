package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var traceSetter func(bool)
var requestHeaders []string

var errVersionPrinted = errors.New("version printed")

func BindTrace(setter func(bool)) {
	traceSetter = setter
}

func NewRootCommand(use, short, version string) *cobra.Command {
	var trace bool
	var showVersion bool
	requestHeaders = nil

	cmd := &cobra.Command{
		Use:           use,
		Short:         short,
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				return printVersion(cmd, version)
			}
			if traceSetter != nil {
				traceSetter(trace)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.CompletionOptions.DisableDefaultCmd = true
	cmd.PersistentFlags().StringArrayVarP(&requestHeaders, "header", "H", nil, "Request header in 'Name: Value' or 'Name=Value' format; repeatable")
	cmd.PersistentFlags().BoolVar(&trace, "trace", false, "Print HTTP request and response trace logs")
	cmd.PersistentFlags().BoolVarP(&showVersion, "version", "v", false, "Print version information")
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
		if command == "skills" {
			continue
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

		if subcommand == "" {
			continue
		}
		examples = append(examples, "  "+root+" "+command+" "+subcommand)
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
{{end}}{{if and .HasParent .HasAvailableInheritedFlags}}{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}
{{end}}
More help: {{.CommandPath}} <command> --help
`

func printVersion(cmd *cobra.Command, version string) error {
	if JSONEnabled(cmd) {
		rendered, err := JSONSuccess(cmd.Root().Name()+" version", "ok", map[string]string{
			"version": version,
		})
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), rendered)
		if err != nil {
			return err
		}
		return errVersionPrinted
	}

	_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s version %s\n", cmd.Root().Name(), version)
	if err != nil {
		return err
	}
	return errVersionPrinted
}
