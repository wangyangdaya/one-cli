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

func TestRenderedGoOAuth2IncludesTopLevelLoginCommand(t *testing.T) {
	dir := t.TempDir()
	app := model.App{
		Name: "businesscli",
		Auth: model.Auth{Type: model.AuthTypeOAuth2},
		Groups: []model.Group{{
			Name:       "auth",
			Operations: []model.Operation{{CommandName: "business-login", Method: "POST", Path: "/api/login"}},
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
	if !strings.Contains(string(root), "auth.NewOAuth2LoginCommand") || !strings.Contains(string(root), "auth.NewOAuth2StatusCommand") || !strings.Contains(string(root), "auth.NewOAuth2LogoutCommand") {
		t.Fatalf("generated root missing OAuth2 login/status/logout commands:\n%s", root)
	}
	if got := strings.Count(string(root), `"github.com/acme/businesscli/internal/auth"`); got != 1 {
		t.Fatalf("generated root imports auth package %d times, want once:\n%s", got, root)
	}
	if !strings.Contains(string(root), "auth.NewCommand()") {
		t.Fatalf("generated root missing business auth command group:\n%s", root)
	}
	if _, err := os.Stat(filepath.Join(dir, "internal", "auth", "oauth2.go")); err != nil {
		t.Fatalf("generated OAuth2 auth command missing: %v", err)
	}
	for _, rel := range []string{filepath.Join("skills", "cli-auth", "SKILL.md"), filepath.Join("skills", "SKILL.md")} {
		content, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			t.Fatalf("read generated OAuth2 skill %s: %v", rel, err)
		}
		if !strings.Contains(string(content), "businesscli login") && !strings.Contains(string(content), "cli-auth/SKILL.md") {
			t.Fatalf("generated OAuth2 skill %s lacks login routing:\n%s", rel, content)
		}
	}
	readme, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatalf("read generated README: %v", err)
	}
	if !strings.Contains(string(readme), "businesscli login") || !strings.Contains(string(readme), "businesscli status") || !strings.Contains(string(readme), "businesscli logout") || strings.Contains(string(readme), "sealed client secret") {
		t.Fatalf("generated authorization-code README has incorrect OAuth guidance:\n%s", readme)
	}
}

func TestRenderedOAuth2LoginReportsBusinessErrorBody(t *testing.T) {
	app := model.App{Name: "businesscli", Auth: model.Auth{Type: model.AuthTypeOAuth2}}
	for _, target := range []string{"go", "rust"} {
		t.Run(target, func(t *testing.T) {
			dir := t.TempDir()
			if err := render.ProjectWithOptions(dir, "github.com/acme/businesscli", app, render.ProjectOptions{
				Target:        target,
				RuntimeBundle: &runtimeconfig.Bundle{OAuth2GrantType: "authorization_code"},
			}); err != nil {
				t.Fatalf("render %s: %v", target, err)
			}
			path := filepath.Join(dir, "src", "oauth_auth.rs")
			want := "business login endpoint returned {}: {}"
			if target == "go" {
				path = filepath.Join(dir, "internal", "auth", "oauth2.go")
				want = "business login endpoint returned %s: %s"
			}
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read generated OAuth login: %v", err)
			}
			if !strings.Contains(string(content), want) {
				t.Fatalf("generated %s OAuth login does not report the business error response body", target)
			}
		})
	}
}

func TestRenderedGoOAuth2ClientCredentialsDoesNotAddLoginCommand(t *testing.T) {
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
	if strings.Contains(string(root), "auth.NewOAuth2LoginCommand") || strings.Contains(string(root), "auth.NewOAuth2LogoutCommand") {
		t.Fatalf("client_credentials root unexpectedly contains interactive auth command:\n%s", root)
	}
	if _, err := os.Stat(filepath.Join(dir, "internal", "auth", "oauth2.go")); !os.IsNotExist(err) {
		t.Fatalf("client_credentials unexpectedly generated interactive auth command: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "skills", "cli-auth", "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("client_credentials unexpectedly generated cli-auth skill: %v", err)
	}
}
