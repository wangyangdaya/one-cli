package command_test

import (
	"bytes"
	"encoding/json"
	"os"
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

func TestGenerateCommandJSONOutput(t *testing.T) {
	dir := t.TempDir()
	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{
		"--json",
		"generate",
		"--input", filepath.Join("..", "..", "examples", "petstore.yaml"),
		"--output", dir,
		"--module", "github.com/acme/generated",
		"--app", "petcli",
	})

	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute generate: %v output=%s", err, out.String())
	}

	var envelope struct {
		OK      bool   `json:"ok"`
		Command string `json:"command"`
		Message string `json:"message"`
		Data    struct {
			Output string `json:"output"`
			Module string `json:"module"`
			App    string `json:"app"`
			Target string `json:"target"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(out.String()), &envelope); err != nil {
		t.Fatalf("expected JSON output, got error %v output=%s", err, out.String())
	}
	if !envelope.OK || envelope.Command != "opencli generate" || envelope.Data.Output != dir || envelope.Data.Target != "go" {
		t.Fatalf("unexpected JSON envelope: %+v", envelope)
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

func TestGenerateCommandAcceptsChineseSkillLanguage(t *testing.T) {
	dir := t.TempDir()
	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{
		"generate",
		"--input", filepath.Join("..", "..", "examples", "petstore.yaml"),
		"--output", dir,
		"--module", "github.com/acme/generated",
		"--app", "petcli",
		"--skill-lang", "zh",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute generate: %v", err)
	}

	skillContent, err := os.ReadFile(filepath.Join(dir, "skills", "pet", "SKILL.md"))
	if err != nil {
		t.Fatalf("read generated skill: %v", err)
	}
	if !strings.Contains(string(skillContent), "## 包文件") {
		t.Fatalf("generated skill is not Chinese:\n%s", skillContent)
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

func TestGenerateCommandJSONErrorOutput(t *testing.T) {
	cmd := newGoRunCommand(t, "./cmd/opencli", "--json", "generate", "--output", t.TempDir(), "--module", "github.com/acme/generated", "--app", "generated")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected generate to fail, got output=%s", out)
	}

	var envelope struct {
		OK      bool   `json:"ok"`
		Command string `json:"command"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	line := bytes.SplitN(bytes.TrimSpace(out), []byte("\n"), 2)[0]
	if err := json.Unmarshal(line, &envelope); err != nil {
		t.Fatalf("expected JSON error output, got error %v output=%s", err, out)
	}
	if envelope.OK || envelope.Error.Code != "command_error" || !strings.Contains(envelope.Error.Message, "exactly one of --input or --mcp-config is required") {
		t.Fatalf("unexpected JSON error envelope: %+v", envelope)
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

func TestGenerateCommandAcceptsAppVersionFlag(t *testing.T) {
	dir := t.TempDir()
	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{
		"generate",
		"--target", "rust",
		"--input", filepath.Join("..", "..", "examples", "petstore.yaml"),
		"--output", dir,
		"--module", "petcli",
		"--app", "petcli",
		"--app-version", "0.0.1",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute rust generate: %v", err)
	}

	cargo, err := os.ReadFile(filepath.Join(dir, "Cargo.toml"))
	if err != nil {
		t.Fatalf("read Cargo.toml: %v", err)
	}
	if !strings.Contains(string(cargo), `version = "0.0.1"`) {
		t.Fatalf("generated Cargo.toml missing app version:\n%s", cargo)
	}
}

func TestGenerateCommandAppVersionFlagOverridesConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "opencli.yaml")
	if err := os.WriteFile(configPath, []byte(`app:
  version: 0.0.1
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	outDir := filepath.Join(dir, "out")
	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{
		"generate",
		"--target", "rust",
		"--input", filepath.Join("..", "..", "examples", "petstore.yaml"),
		"--output", outDir,
		"--module", "petcli",
		"--app", "petcli",
		"--config", configPath,
		"--app-version", "0.0.2",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute rust generate: %v", err)
	}

	cargo, err := os.ReadFile(filepath.Join(outDir, "Cargo.toml"))
	if err != nil {
		t.Fatalf("read Cargo.toml: %v", err)
	}
	if !strings.Contains(string(cargo), `version = "0.0.2"`) {
		t.Fatalf("generated Cargo.toml did not prefer flag version:\n%s", cargo)
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
