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

func TestRenderedOAuth2InjectsAuthorizationForAllHTTPOperations(t *testing.T) {
	app := model.App{
		Name: "oauthcli",
		Auth: model.Auth{Type: model.AuthTypeOAuth2},
		Groups: []model.Group{{
			Name:        "records",
			PackageName: "records",
			Operations: []model.Operation{{
				CommandName:  "list",
				Method:       "GET",
				Path:         "/records",
				AuthRequired: false,
			}},
		}},
	}

	tests := []struct {
		name   string
		target string
		path   string
		want   string
	}{
		{
			name:   "go",
			target: "go",
			path:   "internal/records/service.go",
			want:   "if err := applyOAuth2(req); err != nil",
		},
		{
			name:   "rust",
			target: "rust",
			path:   "src/commands/records.rs",
			want:   `client::request_json_text("GET", &path, query, headers, body, true).await?`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			err := render.ProjectWithOptions(dir, "github.com/acme/oauthcli", app, render.ProjectOptions{
				Target:        tt.target,
				RuntimeBundle: &runtimeconfig.Bundle{OAuth2GrantType: "authorization_code"},
			})
			if err != nil {
				t.Fatalf("render %s OAuth2 project: %v", tt.target, err)
			}
			content, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(tt.path)))
			if err != nil {
				t.Fatalf("read generated %s command: %v", tt.target, err)
			}
			if !strings.Contains(string(content), tt.want) {
				t.Fatalf("generated %s command does not inject OAuth2 authorization for an operation without OpenAPI security:\n%s", tt.target, content)
			}
		})
	}
}
