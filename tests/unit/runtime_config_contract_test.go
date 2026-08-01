package unit_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"one-cli/internal/model"
	"one-cli/internal/render"
	"one-cli/internal/runtimeconfig"
)

func TestRenderedRuntimeRequiresConfiguredCredentialsAndIgnoresCWDConfig(t *testing.T) {
	app := model.App{
		Name: "securecli",
		Auth: model.Auth{Type: model.AuthTypeAPIKey},
		Groups: []model.Group{{
			Name:       "records",
			Operations: []model.Operation{{CommandName: "list", Method: "GET", Path: "/records"}},
		}},
	}

	tests := []struct {
		target string
		path   string
		wants  []string
		avoids []string
	}{
		{
			target: "go",
			path:   "internal/config/runtime_config.go",
			wants: []string{
				"missing API key: set OPENCLI_API_KEY or configure runtime auth",
				"missing bearer token: set OPENCLI_AUTH_TOKEN or configure runtime auth",
			},
			avoids: []string{"os.Getwd()", `filepath.Join(cwd, "config", "runtime.yaml")`},
		},
		{
			target: "rust",
			path:   "src/runtime_config.rs",
			wants: []string{
				"missing API key: set OPENCLI_API_KEY or configure runtime auth",
				"missing bearer token: set OPENCLI_AUTH_TOKEN or configure runtime auth",
			},
			avoids: []string{"env::current_dir()", `cwd.join("config").join("runtime.yaml")`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			dir := t.TempDir()
			if err := render.ProjectWithOptions(dir, "github.com/acme/securecli", app, render.ProjectOptions{Target: tt.target}); err != nil {
				t.Fatalf("render %s: %v", tt.target, err)
			}
			content, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(tt.path)))
			if err != nil {
				t.Fatalf("read %s runtime config: %v", tt.target, err)
			}
			text := string(content)
			for _, want := range tt.wants {
				if !strings.Contains(text, want) {
					t.Errorf("%s runtime config missing %q", tt.target, want)
				}
			}
			for _, avoid := range tt.avoids {
				if strings.Contains(text, avoid) {
					t.Errorf("%s runtime config must not contain CWD lookup %q", tt.target, avoid)
				}
			}
		})
	}
}

func TestRenderedRustTraceUsesHeaderSpecificRedaction(t *testing.T) {
	dir := t.TempDir()
	app := model.App{
		Name: "securecli",
		Auth: model.Auth{Type: model.AuthTypeAPIKey},
		Groups: []model.Group{{
			Name:       "records",
			Operations: []model.Operation{{CommandName: "list", Method: "GET", Path: "/records"}},
		}},
	}
	if err := render.ProjectWithOptions(dir, "github.com/acme/securecli", app, render.ProjectOptions{Target: "rust"}); err != nil {
		t.Fatalf("render Rust: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "src", "client.rs"))
	if err != nil {
		t.Fatalf("read Rust client: %v", err)
	}
	text := string(content)
	for _, want := range []string{
		"let headers_preview = preview_headers(&headers);",
		"fn preview_headers(",
		"if is_safe_trace_header(name)",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("Rust client missing trace redaction contract %q", want)
		}
	}
}

func TestRenderedGoOAuth2IncludesAuthLoginCommand(t *testing.T) {
	dir := t.TempDir()
	app := model.App{
		Name: "businesscli",
		Auth: model.Auth{Type: model.AuthTypeOAuth2},
		Groups: []model.Group{{
			Name:       "records",
			Operations: []model.Operation{{CommandName: "list", Method: "GET", Path: "/records", AuthRequired: true}},
		}},
	}
	if err := render.ProjectWithOptions(dir, "github.com/acme/businesscli", app, render.ProjectOptions{
		Target:        "go",
		RuntimeBundle: &runtimeconfig.Bundle{OAuth2GrantType: "authorization_code"},
	}); err != nil {
		t.Fatalf("render Go: %v", err)
	}

	root, err := os.ReadFile(filepath.Join(dir, "cmd", "businesscli", "main.go"))
	if err != nil {
		t.Fatalf("read generated root: %v", err)
	}
	if !strings.Contains(string(root), "auth.NewOAuth2Command") {
		t.Fatalf("generated root missing OAuth2 auth command:\n%s", root)
	}
	if _, err := os.Stat(filepath.Join(dir, "internal", "auth", "oauth2.go")); err != nil {
		t.Fatalf("generated OAuth2 auth command missing: %v", err)
	}
	readme, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatalf("read generated README: %v", err)
	}
	if !strings.Contains(string(readme), "auth login") || strings.Contains(string(readme), "sealed client secret") {
		t.Fatalf("generated authorization-code README has incorrect OAuth guidance:\n%s", readme)
	}
}

func TestRenderedGoOAuth2ClientCredentialsDoesNotAddAuthLoginCommand(t *testing.T) {
	dir := t.TempDir()
	app := model.App{
		Name: "servicecli",
		Auth: model.Auth{Type: model.AuthTypeOAuth2},
		Groups: []model.Group{{
			Name:       "records",
			Operations: []model.Operation{{CommandName: "list", Method: "GET", Path: "/records", AuthRequired: true}},
		}},
	}
	if err := render.ProjectWithOptions(dir, "github.com/acme/servicecli", app, render.ProjectOptions{
		Target:        "go",
		RuntimeBundle: &runtimeconfig.Bundle{OAuth2GrantType: "client_credentials"},
	}); err != nil {
		t.Fatalf("render Go: %v", err)
	}
	root, err := os.ReadFile(filepath.Join(dir, "cmd", "servicecli", "main.go"))
	if err != nil {
		t.Fatalf("read generated root: %v", err)
	}
	if strings.Contains(string(root), "auth.NewOAuth2Command") {
		t.Fatalf("client_credentials root unexpectedly contains interactive auth command:\n%s", root)
	}
	if _, err := os.Stat(filepath.Join(dir, "internal", "auth", "oauth2.go")); !os.IsNotExist(err) {
		t.Fatalf("client_credentials unexpectedly generated interactive auth command: %v", err)
	}
}
