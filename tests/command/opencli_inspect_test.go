package command_test

import (
	"bytes"
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
