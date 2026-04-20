package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"one-cli/internal/app"
)

func TestGenerateSmoke(t *testing.T) {
	dir := t.TempDir()
	if err := app.RunGenerate(filepath.Join("..", "..", "examples", "petstore.yaml"), dir, "github.com/acme/generated", "petcli", ""); err != nil {
		t.Fatalf("run generate: %v", err)
	}

	for _, rel := range []string{
		"cmd/petcli/main.go",
		"bin/petcli",
		"internal/pet/command.go",
		"internal/pet/service.go",
		"skills/pet/SKILL.md",
		"README.md",
	} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
	}
}

func TestGenerateSmokeIncludesSimpleJSONBodyFlags(t *testing.T) {
	dir := t.TempDir()
	if err := app.RunGenerate(filepath.Join("..", "..", "examples", "openapi.json"), dir, "github.com/acme/generated", "openapi-cli", ""); err != nil {
		t.Fatalf("run generate: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "internal", "auth", "command.go"))
	if err != nil {
		t.Fatalf("read command: %v", err)
	}

	text := string(content)
	for _, want := range []string{
		`cmd.Flags().StringVar(&bodyData, "data", "", "Raw JSON request body")`,
		`cmd.Flags().StringVar(&bodyFile, "file", "", "Path to JSON request body file")`,
		`cmd.Flags().StringVar(&bodyEmail, "email", "", "JSON body field: email")`,
		`cmd.Flags().StringVar(&bodyPassword, "password", "", "JSON body field: password")`,
		`cmd.Flags().BoolVar(&bodyRemember, "remember", false, "JSON body field: remember")`,
		`Body input: --email/--password/--remember, --data, or --file`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("generated command missing %q", want)
		}
	}
}
