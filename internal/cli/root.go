package cli

import (
	"strings"

	"github.com/spf13/cobra"
)

func NewRootCommand(use, short, version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:           use,
		Short:         short,
		Version:       version,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.CompletionOptions.DisableDefaultCmd = true
	return cmd
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
{{end}}
More help: {{.CommandPath}} <command> --help
`
