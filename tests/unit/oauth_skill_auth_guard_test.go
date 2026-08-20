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

func TestRenderedOAuthAuthorizationCodeSkillsExposeAuthGuardAtEveryEntry(t *testing.T) {
	app := model.App{
		Name: "businesscli",
		Auth: model.Auth{Type: model.AuthTypeOAuth2},
		Groups: []model.Group{{
			Name:       "records",
			Operations: []model.Operation{{CommandName: "list", Method: "GET", Path: "/records", AuthRequired: true}},
		}},
	}

	tests := []struct {
		lang                   string
		rootDescription        string
		rootHeading            string
		groupHeading           string
		statusInstruction      string
		unconditionalLoginText string
	}{
		{
			lang:                   "en",
			rootDescription:        "check CLI login status",
			rootHeading:            "## OAuth login prerequisite",
			groupHeading:           "## Login prerequisite",
			statusInstruction:      "businesscli status",
			unconditionalLoginText: "Run `businesscli login` before invoking protected business commands",
		},
		{
			lang:                   "zh",
			rootDescription:        "检查 CLI 登录状态",
			rootHeading:            "## OAuth 登录前置",
			groupHeading:           "## 登录前置",
			statusInstruction:      "businesscli status",
			unconditionalLoginText: "调用受保护业务命令前执行 `businesscli login`",
		},
	}

	for _, tc := range tests {
		t.Run(tc.lang, func(t *testing.T) {
			dir := t.TempDir()
			if err := render.ProjectWithOptions(dir, "github.com/acme/businesscli", app, render.ProjectOptions{
				Target:        "go",
				SkillLang:     tc.lang,
				RuntimeBundle: &runtimeconfig.Bundle{OAuth2GrantType: "authorization_code"},
			}); err != nil {
				t.Fatalf("render %s skills: %v", tc.lang, err)
			}

			root := readGeneratedSkill(t, dir, "skills", "SKILL.md")
			for _, want := range []string{tc.rootDescription, tc.rootHeading, tc.statusInstruction, "cli-auth/SKILL.md", "needs_refresh", "not_logged_in"} {
				if !strings.Contains(root, want) {
					t.Errorf("generated %s root Skill missing auth guard %q:\n%s", tc.lang, want, root)
				}
			}

			group := readGeneratedSkill(t, dir, "skills", "records", "SKILL.md")
			for _, want := range []string{tc.groupHeading, tc.statusInstruction, "../cli-auth/SKILL.md", "needs_refresh", "not_logged_in"} {
				if !strings.Contains(group, want) {
					t.Errorf("generated %s group Skill missing auth fallback %q:\n%s", tc.lang, want, group)
				}
			}
			if strings.Contains(group, tc.unconditionalLoginText) {
				t.Errorf("generated %s group Skill still requires unconditional login:\n%s", tc.lang, group)
			}
		})
	}
}

func readGeneratedSkill(t *testing.T, root string, path ...string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(append([]string{root}, path...)...))
	if err != nil {
		t.Fatalf("read generated Skill: %v", err)
	}
	return string(content)
}
