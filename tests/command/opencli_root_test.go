package command_test

import (
	"encoding/json"
	"strings"
	"testing"

	"one-cli/internal/configgen"
)

func TestOpenCLIHelp(t *testing.T) {
	cmd := newGoRunCommand(t, "./cmd/opencli", "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected help command to succeed, got error: %v, output: %s", err, string(out))
	}

	output := string(out)
	for _, want := range []string{"opencli", "init", "inspect", "generate"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected help output to mention %q, got: %s", want, output)
		}
	}
}

func TestInitCommandJSONOutput(t *testing.T) {
	cmd := newGoRunCommand(t, "./cmd/opencli", "--json", "init")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected init --json to succeed, got error: %v, output: %s", err, string(out))
	}

	var envelope struct {
		OK      bool   `json:"ok"`
		Command string `json:"command"`
	}
	if err := json.Unmarshal(out, &envelope); err != nil {
		t.Fatalf("expected JSON output, got error %v output=%s", err, out)
	}
	if !envelope.OK || envelope.Command != "opencli init" {
		t.Fatalf("unexpected JSON envelope: %+v", envelope)
	}
}

func TestConfigGenLoadBytesYAML(t *testing.T) {
	cfg, err := configgen.LoadBytes([]byte(`
app:
  binary: opencli
  root_command: opencli
  version: 0.0.1
naming:
  tag_alias:
    pet: pets
runtime:
  auth_header: Authorization
overrides:
  body_mode:
    submit: json
`))
	if err != nil {
		t.Fatalf("load yaml config: %v", err)
	}

	if cfg.App.Binary != "opencli" {
		t.Fatalf("unexpected binary: %q", cfg.App.Binary)
	}
	if cfg.App.Version != "0.0.1" {
		t.Fatalf("unexpected version: %q", cfg.App.Version)
	}
	if cfg.Naming.TagAlias["pet"] != "pets" {
		t.Fatalf("unexpected tag alias: %q", cfg.Naming.TagAlias["pet"])
	}
	if cfg.Overrides.BodyMode["submit"] != "json" {
		t.Fatalf("unexpected body mode: %q", cfg.Overrides.BodyMode["submit"])
	}
}
