package app

import (
	"fmt"
	"strings"

	"one-cli/internal/loaders"
	"one-cli/internal/openapi"

	"github.com/spf13/cobra"
)

func NewInspectCommand() *cobra.Command {
	var input string

	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Inspect an OpenAPI document",
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, err := loaders.Load(strings.TrimSpace(input))
			if err != nil {
				return err
			}

			doc, err := openapi.Parse(raw)
			if err != nil {
				return err
			}

			for _, op := range doc.Operations {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s %s %s %s\n", op.Tag, op.Method, op.Path, op.OperationID); err != nil {
					return err
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&input, "input", "", "Path or URL to the OpenAPI document")
	_ = cmd.MarkFlagRequired("input")
	return cmd
}
