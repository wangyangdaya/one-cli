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
	loginSkill, err := os.ReadFile(filepath.Join(dir, "skills", "cli-auth", "SKILL.md"))
	if err != nil {
		t.Fatalf("read generated OAuth2 login skill: %v", err)
	}
	for _, want := range []string{"terminal", "terminal_id", "login --no-browser", "same terminal session"} {
		if !strings.Contains(string(loginSkill), want) {
			t.Fatalf("generated OAuth2 login skill lacks persistent-terminal guidance %q:\n%s", want, loginSkill)
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

func TestRenderedOAuth2LoginIncludesOptionalPKCEAndOIDC(t *testing.T) {
	app := model.App{Name: "businesscli", Auth: model.Auth{Type: model.AuthTypeOAuth2}}
	for _, target := range []string{"go", "rust"} {
		t.Run(target, func(t *testing.T) {
			dir := t.TempDir()
			if err := render.ProjectWithOptions(dir, "github.com/acme/businesscli", app, render.ProjectOptions{
				Target: target,
				RuntimeBundle: &runtimeconfig.Bundle{
					OAuth2GrantType:   "authorization_code",
					OAuth2PKCEEnabled: true,
					OIDCEnabled:       true,
				},
			}); err != nil {
				t.Fatalf("render %s: %v", target, err)
			}
			path := filepath.Join(dir, "src", "oauth_auth.rs")
			wants := []string{"code_challenge", "code_verifier", "nonce", "id_token", "jwks_uri", "RS256"}
			if target == "go" {
				path = filepath.Join(dir, "internal", "auth", "oauth2.go")
			}
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read generated %s OAuth login: %v", target, err)
			}
			for _, want := range wants {
				if !strings.Contains(string(content), want) {
					t.Errorf("generated %s OAuth login missing %q", target, want)
				}
			}
		})
	}
}

func TestRenderedOAuth2TokenRequestsAcceptJSON(t *testing.T) {
	for _, target := range []string{"go", "rust"} {
		t.Run(target, func(t *testing.T) {
			auth, runtime := renderOAuthAuthorizationCodeSources(t, target)
			loginAccept := `requestHeaders.Set("Accept", "application/json")`
			refreshAccept := `req.Header.Set("Accept", "application/json")`
			if target == "rust" {
				loginAccept = `request = request.header("Accept", "application/json")`
				refreshAccept = `.header("Accept", "application/json")`
			}
			if !strings.Contains(auth, loginAccept) {
				t.Errorf("generated %s login token request does not accept JSON", target)
			}
			if !strings.Contains(runtime, refreshAccept) {
				t.Errorf("generated %s refresh token request does not accept JSON", target)
			}
		})
	}
}

func TestRenderedOAuth2RefreshClassifiesFailures(t *testing.T) {
	for _, target := range []string{"go", "rust"} {
		t.Run(target, func(t *testing.T) {
			_, runtime := renderOAuthAuthorizationCodeSources(t, target)
			for _, want := range []string{"invalid_grant", "login_required: run", "refresh_failed:"} {
				if !strings.Contains(runtime, want) {
					t.Errorf("generated %s OAuth refresh flow missing %q", target, want)
				}
			}
		})
	}
}

func TestRenderedOAuth2RequiresTokenType(t *testing.T) {
	for _, target := range []string{"go", "rust"} {
		t.Run(target, func(t *testing.T) {
			auth, runtime := renderOAuthAuthorizationCodeSources(t, target)
			if !strings.Contains(auth, "business token response is missing token_type") {
				t.Errorf("generated %s OAuth login accepts a response without token_type", target)
			}
			if !strings.Contains(runtime, "OAuth token response is missing token_type") {
				t.Errorf("generated %s OAuth runtime accepts a response without token_type", target)
			}
		})
	}
}

func renderOAuthAuthorizationCodeSources(t *testing.T, target string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	app := model.App{Name: "businesscli", Auth: model.Auth{Type: model.AuthTypeOAuth2}}
	if err := render.ProjectWithOptions(dir, "github.com/acme/businesscli", app, render.ProjectOptions{
		Target:        target,
		RuntimeBundle: &runtimeconfig.Bundle{OAuth2GrantType: "authorization_code"},
	}); err != nil {
		t.Fatalf("render %s: %v", target, err)
	}
	authPath := filepath.Join(dir, "src", "oauth_auth.rs")
	runtimePath := filepath.Join(dir, "src", "runtime_config.rs")
	if target == "go" {
		authPath = filepath.Join(dir, "internal", "auth", "oauth2.go")
		runtimePath = filepath.Join(dir, "internal", "config", "runtime_config.go")
	}
	auth, err := os.ReadFile(authPath)
	if err != nil {
		t.Fatalf("read generated %s OAuth login: %v", target, err)
	}
	runtime, err := os.ReadFile(runtimePath)
	if err != nil {
		t.Fatalf("read generated %s OAuth runtime: %v", target, err)
	}
	return string(auth), string(runtime)
}

func TestRenderedOAuth2RuntimeValidatesSecuritySensitiveURLs(t *testing.T) {
	app := model.App{Name: "businesscli", Auth: model.Auth{Type: model.AuthTypeOAuth2}}
	for _, target := range []string{"go", "rust"} {
		t.Run(target, func(t *testing.T) {
			dir := t.TempDir()
			if err := render.ProjectWithOptions(dir, "github.com/acme/businesscli", app, render.ProjectOptions{
				Target: target,
				RuntimeBundle: &runtimeconfig.Bundle{
					OAuth2GrantType: "authorization_code",
					OIDCEnabled:     true,
				},
			}); err != nil {
				t.Fatalf("render %s: %v", target, err)
			}

			runtimePath := filepath.Join(dir, "src", "runtime_config.rs")
			authPath := filepath.Join(dir, "src", "oauth_auth.rs")
			wantsRuntime := []string{"validate_oauth_loopback_redirect_uri", "127.0.0.1", "openid", "oidc_issuer"}
			wantsAuth := []string{"validate_oidc_endpoint", "jwks_uri"}
			if target == "go" {
				runtimePath = filepath.Join(dir, "internal", "config", "runtime_config.go")
				authPath = filepath.Join(dir, "internal", "auth", "oauth2.go")
				wantsRuntime = []string{"validateOAuthLoopbackRedirectURI", "127.0.0.1", "openid", "OIDCIssuer"}
				wantsAuth = []string{"validateOIDCEndpoint", "jwks_uri"}
			}
			for path, wants := range map[string][]string{runtimePath: wantsRuntime, authPath: wantsAuth} {
				content, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("read generated %s source %s: %v", target, path, err)
				}
				for _, want := range wants {
					if !strings.Contains(string(content), want) {
						t.Errorf("generated %s source %s missing security validation %q", target, path, want)
					}
				}
			}
		})
	}
}

func TestRenderedOAuth2SkillReportsThreeCommands(t *testing.T) {
	app := model.App{
		Name: "businesscli",
		Auth: model.Auth{Type: model.AuthTypeOAuth2},
		Groups: []model.Group{{
			Name:       "records",
			Operations: []model.Operation{{CommandName: "list", Method: "GET", Path: "/records"}},
		}},
	}
	for _, lang := range []string{"en", "zh"} {
		t.Run(lang, func(t *testing.T) {
			dir := t.TempDir()
			if err := render.ProjectWithOptions(dir, "github.com/acme/businesscli", app, render.ProjectOptions{
				Target:        "go",
				SkillLang:     lang,
				RuntimeBundle: &runtimeconfig.Bundle{OAuth2GrantType: "authorization_code"},
			}); err != nil {
				t.Fatalf("render %s skills: %v", lang, err)
			}
			for _, rel := range []string{filepath.Join("skills", "README.md"), filepath.Join("skills", "SKILL.md")} {
				content, err := os.ReadFile(filepath.Join(dir, rel))
				if err != nil {
					t.Fatalf("read generated %s: %v", rel, err)
				}
				if !strings.Contains(string(content), "| `cli-auth` | 3 |") {
					t.Errorf("generated %s %s has stale cli-auth command count:\n%s", lang, rel, content)
				}
			}
		})
	}
}

func TestRenderedRustOAuthTokenFileCreatedPrivate(t *testing.T) {
	dir := t.TempDir()
	app := model.App{Name: "businesscli", Auth: model.Auth{Type: model.AuthTypeOAuth2}}
	if err := render.ProjectWithOptions(dir, "github.com/acme/businesscli", app, render.ProjectOptions{
		Target:        "rust",
		RuntimeBundle: &runtimeconfig.Bundle{OAuth2GrantType: "authorization_code"},
	}); err != nil {
		t.Fatalf("render Rust: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "src", "runtime_config.rs"))
	if err != nil {
		t.Fatalf("read generated Rust runtime config: %v", err)
	}
	text := string(content)
	for _, want := range []string{"OpenOptions::new()", ".create_new(true)", ".mode(0o600)", "write_all(&payload)"} {
		if !strings.Contains(text, want) {
			t.Errorf("generated Rust OAuth token writer missing %q", want)
		}
	}
	if strings.Contains(text, "fs::write(&temporary, payload)") {
		t.Errorf("generated Rust OAuth token writer writes credentials before setting mode 0600")
	}
}

func TestRenderedOAuth2TokenPathUsesUserHomeAndAuthIdentity(t *testing.T) {
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

			path := filepath.Join(dir, "src", "runtime_config.rs")
			wants := []string{
				`dirs::home_dir()`,
				`.join(".opencli").join("oauth2").join(namespace).join("oauth-token.json")`,
				`Sha256::digest`,
			}
			if target == "go" {
				path = filepath.Join(dir, "internal", "config", "runtime_config.go")
				wants = []string{
					`os.UserHomeDir()`,
					`filepath.Join(home, ".opencli", "oauth2", namespace, "oauth-token.json")`,
					`sha256.Sum256`,
				}
			}
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read generated %s runtime config: %v", target, err)
			}
			for _, want := range wants {
				if !strings.Contains(string(content), want) {
					t.Errorf("generated %s OAuth token path missing %q", target, want)
				}
			}
			if target == "rust" {
				cargo, err := os.ReadFile(filepath.Join(dir, "Cargo.toml"))
				if err != nil {
					t.Fatalf("read generated Cargo.toml: %v", err)
				}
				for _, dependency := range []string{`dirs = "6"`, `sha2 = "0.10"`} {
					if !strings.Contains(string(cargo), dependency) {
						t.Errorf("generated Cargo.toml missing %s", dependency)
					}
				}
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
