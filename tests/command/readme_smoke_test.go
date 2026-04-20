package command_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"one-cli/internal/app"
)

func TestREADMEContainsOpenCLICommands(t *testing.T) {
	data, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatalf("expected README to be readable, got %v", err)
	}

	text := string(data)
	for _, needle := range []string{"opencli init", "opencli inspect", "opencli generate"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("expected README to mention %q, got: %s", needle, text)
		}
	}
}

func TestGeneratedREADMEIncludesSetupAndTraceGuidance(t *testing.T) {
	dir := t.TempDir()
	if err := app.RunGenerate(filepath.Join("..", "..", "examples", "openapi.json"), dir, "github.com/acme/generated", "openapi-cli", ""); err != nil {
		t.Fatalf("generate: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatalf("readme: %v", err)
	}

	text := string(content)
	for _, want := range []string{
		"OPENCLI_BASE_URL",
		"--data",
		"--file",
		"--trace",
		"auth login",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("README missing %q\n%s", want, text)
		}
	}
}
