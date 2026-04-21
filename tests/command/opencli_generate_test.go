package command_test

import (
	"path/filepath"
	"strings"
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

func TestGenerateCommandRequiresExactlyOneSource(t *testing.T) {
	dir := t.TempDir()
	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{
		"generate",
		"--output", dir,
		"--module", "github.com/acme/generated",
		"--app", "generated",
	})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "exactly one of --input or --mcp-config is required") {
		t.Fatalf("expected source selection error, got %v", err)
	}
}

func TestGenerateCommandRejectsMixedSources(t *testing.T) {
	dir := t.TempDir()
	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{
		"generate",
		"--input", filepath.Join("..", "..", "examples", "petstore.yaml"),
		"--mcp-config", filepath.Join("testdata", "mcp.json"),
		"--output", dir,
		"--module", "github.com/acme/generated",
		"--app", "generated",
	})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "exactly one of --input or --mcp-config is required") {
		t.Fatalf("expected mixed source error, got %v", err)
	}
}

func TestGenerateCommandAcceptsRustTargetWithOpenAPI(t *testing.T) {
	dir := t.TempDir()
	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{
		"generate",
		"--target", "rust",
		"--input", filepath.Join("..", "..", "examples", "petstore.yaml"),
		"--output", dir,
		"--module", "petcli",
		"--app", "petcli",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute rust generate: %v", err)
	}
}

func TestGenerateCommandRejectsUnknownTarget(t *testing.T) {
	dir := t.TempDir()
	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{
		"generate",
		"--target", "python",
		"--input", filepath.Join("..", "..", "examples", "petstore.yaml"),
		"--output", dir,
		"--module", "petcli",
		"--app", "petcli",
	})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "unsupported target") {
		t.Fatalf("expected unsupported target error, got %v", err)
	}
}
