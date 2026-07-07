package command_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"one-cli/internal/ainormalize"
	"one-cli/internal/app"
)

func TestInspectCommand(t *testing.T) {
	cmd := app.NewRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"inspect", "--input", filepath.Join("..", "..", "examples", "petstore.yaml")})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute inspect: %v", err)
	}

	out := buf.String()
	if !bytes.Contains([]byte(out), []byte("listPets")) {
		t.Fatalf("unexpected inspect output: %q", out)
	}
}

func TestInspectCommandJSONOutput(t *testing.T) {
	cmd := app.NewRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--json", "inspect", "--input", filepath.Join("..", "..", "examples", "petstore.yaml")})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute inspect: %v", err)
	}

	var envelope struct {
		OK      bool   `json:"ok"`
		Command string `json:"command"`
		Data    struct {
			Operations []struct {
				OperationID string `json:"operation_id"`
			} `json:"operations"`
		} `json:"data"`
	}
	if err := json.Unmarshal(buf.Bytes(), &envelope); err != nil {
		t.Fatalf("expected JSON inspect output, got error %v output=%s", err, buf.String())
	}
	if !envelope.OK || envelope.Command != "opencli inspect" || len(envelope.Data.Operations) == 0 {
		t.Fatalf("unexpected JSON envelope: %+v", envelope)
	}
}

func TestInspectCommandAISuggestConfig(t *testing.T) {
	restore := app.SetAISuggestClientForTest(ainormalize.ClientFunc(func(ctx context.Context, inventory ainormalize.Inventory) (ainormalize.Suggestion, error) {
		if len(inventory.Operations) == 0 {
			t.Fatal("expected operation inventory")
		}
		return ainormalize.Suggestion{
			TagAlias: map[string]string{"计划物流.": "logistics"},
			OperationAlias: map[string]string{
				"POST /api-apply/v2/get/supplierMrpMonth": "mrp-month",
				"POST /missing": "missing",
			},
		}, nil
	}))
	defer restore()

	cmd := app.NewRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"inspect", "--input", filepath.Join("..", "..", "examples", "supplier.json"), "--ai-suggest-config"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute inspect ai suggest: %v stderr=%s", err, stderr.String())
	}

	out := stdout.String()
	for _, want := range []string{
		"naming:",
		"tag_alias:",
		"计划物流.: logistics",
		"operation_alias:",
		"POST /api-apply/v2/get/supplierMrpMonth: mrp-month",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("AI suggestion output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "POST /missing") {
		t.Fatalf("AI suggestion output should reject unknown operation:\n%s", out)
	}
	if !strings.Contains(stderr.String(), "rejected") {
		t.Fatalf("expected rejection diagnostic on stderr, got %q", stderr.String())
	}
}

func TestInspectCommandAISuggestConfigWritesOutputFile(t *testing.T) {
	restore := app.SetAISuggestClientForTest(ainormalize.ClientFunc(func(ctx context.Context, inventory ainormalize.Inventory) (ainormalize.Suggestion, error) {
		return ainormalize.Suggestion{
			TagAlias: map[string]string{"计划物流.": "logistics"},
			OperationAlias: map[string]string{
				"POST /api-apply/v2/get/supplierMrpMonth": "mrp-month",
			},
		}, nil
	}))
	defer restore()

	outputPath := filepath.Join(t.TempDir(), "opencli.ai.yaml")
	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{
		"inspect",
		"--input", filepath.Join("..", "..", "examples", "supplier.json"),
		"--ai-suggest-config",
		"--output", outputPath,
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute inspect ai suggest: %v", err)
	}
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if !strings.Contains(string(content), "mrp-month") {
		t.Fatalf("output file missing suggestion:\n%s", content)
	}
}

func TestInspectCommandAISuggestConfigRequiresAIEnv(t *testing.T) {
	restore := app.SetAISuggestClientForTest(nil)
	defer restore()
	t.Setenv("OPENCLI_AI_BASE_URL", "")
	t.Setenv("OPENCLI_AI_API_KEY", "")
	t.Setenv("OPENCLI_AI_MODEL", "")

	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{"inspect", "--input", filepath.Join("..", "..", "examples", "supplier.json"), "--ai-suggest-config"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "OPENCLI_AI_BASE_URL") {
		t.Fatalf("expected missing AI env error, got %v", err)
	}
}
