package command_test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

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
