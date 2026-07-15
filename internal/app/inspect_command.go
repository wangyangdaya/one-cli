package app

import (
	"context"
	"fmt"
	"os"
	"strings"

	"one-cli/internal/ainormalize"
	"one-cli/internal/loaders"
	"one-cli/internal/openapi"
	outjson "one-cli/internal/output"

	"github.com/spf13/cobra"
)

func NewInspectCommand() *cobra.Command {
	var input string
	var aiSuggestConfig bool
	var output string

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

			if aiSuggestConfig {
				if JSONEnabled(cmd) {
					return fmt.Errorf("--ai-suggest-config does not support --json")
				}
				return runAISuggestConfig(cmd, doc, strings.TrimSpace(output))
			}

			if JSONEnabled(cmd) {
				operations := make([]inspectOperation, 0, len(doc.Operations))
				for _, op := range doc.Operations {
					operations = append(operations, inspectOperation{
						Tag:         op.Tag,
						Method:      op.Method,
						Path:        op.Path,
						OperationID: op.OperationID,
					})
				}
				rendered, err := outjson.JSONSuccess(cmd.CommandPath(), "inspected document", map[string]any{
					"operations": operations,
				})
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(cmd.OutOrStdout(), rendered)
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
	cmd.Flags().BoolVar(&aiSuggestConfig, "ai-suggest-config", false, "Use AI to suggest an opencli.yaml naming config")
	cmd.Flags().StringVar(&output, "output", "", "Write AI suggestion YAML to a file")
	_ = cmd.MarkFlagRequired("input")
	return cmd
}

func runAISuggestConfig(cmd *cobra.Command, doc openapi.Document, output string) error {
	inventory := ainormalize.BuildInventory(doc)
	client, err := aiSuggestClient()
	if err != nil {
		return err
	}
	suggestion, err := client.SuggestConfig(context.Background(), inventory)
	if err != nil {
		return err
	}
	cfg, diagnostics := ainormalize.ValidateSuggestion(inventory, suggestion)
	if cfg.Naming.TagAlias == nil && cfg.Naming.OperationAlias == nil {
		return fmt.Errorf("AI suggestion did not contain any valid aliases")
	}
	rendered, err := ainormalize.RenderConfigYAML(cfg)
	if err != nil {
		return err
	}
	for _, rejected := range diagnostics.Rejected {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "rejected %s %q -> %q: %s\n", rejected.Kind, rejected.Key, rejected.Alias, rejected.Reason)
	}
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "analyzed %d operations\n", len(inventory.Operations))
	if output != "" {
		return os.WriteFile(output, rendered, 0o644)
	}
	_, err = cmd.OutOrStdout().Write(rendered)
	return err
}

var aiSuggestClientOverride ainormalize.Client

func aiSuggestClient() (ainormalize.Client, error) {
	if aiSuggestClientOverride != nil {
		return aiSuggestClientOverride, nil
	}
	return ainormalize.NewOpenAICompatibleClientFromEnv()
}

func SetAISuggestClientForTest(client ainormalize.Client) func() {
	previous := aiSuggestClientOverride
	aiSuggestClientOverride = client
	return func() {
		aiSuggestClientOverride = previous
	}
}

type inspectOperation struct {
	Tag         string `json:"tag"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	OperationID string `json:"operation_id"`
}
