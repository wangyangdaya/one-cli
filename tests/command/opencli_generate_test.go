package command_test

import (
	"path/filepath"
	"testing"

	"one-cli/internal/app"
)

func TestGenerateCommand(t *testing.T) {
	dir := t.TempDir()
	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{
		"generate",
		"--input", filepath.Join("..", "..", "examples", "petstore.yaml"),
		"--output", dir,
		"--module", "github.com/acme/generated",
		"--app", "petcli",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute generate: %v", err)
	}
}

func TestGenerateCommandWithSimpleJSONBodySpec(t *testing.T) {
	dir := t.TempDir()
	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{
		"generate",
		"--input", filepath.Join("..", "..", "examples", "openapi.json"),
		"--output", dir,
		"--module", "github.com/acme/generated",
		"--app", "openapi-cli",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute generate: %v", err)
	}
}
