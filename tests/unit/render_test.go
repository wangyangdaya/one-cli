package unit_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"one-cli/internal/model"
	"one-cli/internal/render"
)

func TestRenderProject(t *testing.T) {
	dir := t.TempDir()
	app := model.App{
		Name: "one",
		Groups: []model.Group{
			{
				Name: "leave",
				Operations: []model.Operation{
					{CommandName: "list", Method: "GET", Path: "/leaves"},
				},
			},
		},
	}

	if err := render.Project(dir, "github.com/acme/one-cli", app); err != nil {
		t.Fatalf("render: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "cmd", "one", "main.go")); err != nil {
		t.Fatalf("missing main.go: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "internal", "leave", "command.go")); err != nil {
		t.Fatalf("missing group command: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "bin", "one")); err != nil {
		t.Fatalf("missing launcher: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "skills", "leave", "SKILL.md")); err != nil {
		t.Fatalf("missing generated skill markdown: %v", err)
	}
	skillContent, err := os.ReadFile(filepath.Join(dir, "skills", "leave", "SKILL.md"))
	if err != nil {
		t.Fatalf("read generated skill markdown: %v", err)
	}
	skillText := string(skillContent)
	if !strings.HasPrefix(skillText, "---\nname: leave\ndescription: Commands for the leave group in one\n---\n") {
		t.Fatalf("generated skill markdown missing YAML frontmatter:\n%s", skillText)
	}

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(),
		"GOCACHE="+filepath.Join(t.TempDir(), "gocache"),
		"GOTOOLCHAIN=local",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated project should compile, got %v, output: %s", err, string(out))
	}
}

func TestRenderProjectSkillIncludesHeaderUsageNotes(t *testing.T) {
	dir := t.TempDir()
	app := model.App{
		Name: "one",
		Groups: []model.Group{
			{
				Name: "auth",
				Operations: []model.Operation{
					{
						CommandName: "me",
						Method:      "GET",
						Path:        "/auth/me",
						Parameters: []model.Parameter{
							{Name: "authorization", In: "header", Type: "string"},
						},
					},
				},
			},
		},
	}

	if err := render.Project(dir, "github.com/acme/one-cli", app); err != nil {
		t.Fatalf("render: %v", err)
	}

	skillContent, err := os.ReadFile(filepath.Join(dir, "skills", "auth", "SKILL.md"))
	if err != nil {
		t.Fatalf("read generated skill markdown: %v", err)
	}
	skillText := string(skillContent)
	for _, want := range []string{
		"`authorization` (`header`, `string`) optional",
		`--header "authorization: <value>"`,
	} {
		if !strings.Contains(skillText, want) {
			t.Fatalf("generated skill markdown missing %q:\n%s", want, skillText)
		}
	}
}
