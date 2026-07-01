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
		Name:    "one",
		Version: "0.0.1",
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
	mainContent, err := os.ReadFile(filepath.Join(dir, "cmd", "one", "main.go"))
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if !strings.Contains(string(mainContent), `var version = "0.0.1"`) {
		t.Fatalf("generated main.go missing app version:\n%s", mainContent)
	}
	if _, err := os.Stat(filepath.Join(dir, "skills", "leave", "SKILL.md")); err != nil {
		t.Fatalf("missing generated skill markdown: %v", err)
	}
	for _, rel := range []string{
		"skills/leave/README.md",
		"skills/leave/assets/demo-request.json",
		"skills/leave/references/command-routing.md",
		"skills/leave/references/workflows.md",
		"skills/leave/references/production-checklist.md",
	} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Fatalf("missing generated skill package file %s: %v", rel, err)
		}
	}
	skillContent, err := os.ReadFile(filepath.Join(dir, "skills", "leave", "SKILL.md"))
	if err != nil {
		t.Fatalf("read generated skill markdown: %v", err)
	}
	skillText := string(skillContent)
	if !strings.Contains(skillText, "name: leave") {
		t.Fatalf("generated skill markdown missing name in frontmatter:\n%s", skillText)
	}
	if !strings.Contains(skillText, "version: 1.0.0") {
		t.Fatalf("generated skill markdown missing version in frontmatter:\n%s", skillText)
	}
	if !strings.Contains(skillText, "## Package Files") {
		t.Fatalf("generated skill markdown missing package file index:\n%s", skillText)
	}
	if !strings.Contains(skillText, "assets/demo-request.json") {
		t.Fatalf("generated skill markdown missing demo request reference:\n%s", skillText)
	}
	readmeContent, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatalf("read generated README: %v", err)
	}
	readmeText := string(readmeContent)
	for _, want := range []string{
		"## Version",
		"Generated Go CLIs use `0.0.1` for `--version`.",
		"./bin/one --version",
		"## Generated Skills",
		"demo-request.json",
		"production-checklist.md",
	} {
		if !strings.Contains(readmeText, want) {
			t.Fatalf("generated README missing %q:\n%s", want, readmeText)
		}
	}
	if strings.Contains(readmeText, "ldflags") {
		t.Fatalf("generated README should not recommend build-time version overrides:\n%s", readmeText)
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

	binary := filepath.Join(dir, "one")
	build := exec.Command("go", "build", "-o", binary, "./cmd/one")
	build.Dir = dir
	build.Env = append(build.Environ(),
		"GOCACHE="+filepath.Join(t.TempDir(), "gocache"),
		"GOTOOLCHAIN=local",
	)
	out, err = build.CombinedOutput()
	if err != nil {
		t.Fatalf("generated project binary should build, got %v, output: %s", err, string(out))
	}

	version := exec.Command(binary, "--version")
	out, err = version.CombinedOutput()
	if err != nil {
		t.Fatalf("generated project --version should succeed, got %v, output: %s", err, string(out))
	}
	if got := string(out); !strings.Contains(got, "one version 0.0.1") {
		t.Fatalf("generated project --version output = %q, want app name and generated version", got)
	}

	help := exec.Command(binary, "--help")
	out, err = help.CombinedOutput()
	if err != nil {
		t.Fatalf("generated project --help should succeed, got %v, output: %s", err, string(out))
	}
	helpText := string(out)
	for _, want := range []string{
		"one CLI",
		"USAGE:",
		"one [options] [command]",
		"EXAMPLES:",
		"one leave list",
		"More help: one <command> --help",
		"Available Commands:",
		"leave",
		"--trace",
		"--version",
	} {
		if !strings.Contains(helpText, want) {
			t.Fatalf("generated project --help missing %q:\n%s", want, helpText)
		}
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
		"## Commands",
		"## Command Routing",
		"## Core Concepts",
		"## Important Notes",
		"## Common Workflows",
		"### one auth me",
		`--header "authorization: <value>"`,
		"**Parameters:**",
		"<!-- MANUAL:",
	} {
		if !strings.Contains(skillText, want) {
			t.Fatalf("generated SKILL.md missing %q:\n%s", want, skillText)
		}
	}
}

func TestRenderProjectSkillDocumentsFileOrDataBodyFields(t *testing.T) {
	dir := t.TempDir()
	app := model.App{
		Name: "les",
		Groups: []model.Group{
			{
				Name: "te-mm-mri-current",
				Operations: []model.Operation{
					{
						CommandName:  "clear",
						Method:       "POST",
						Path:         "/les/api/teMmMriCurrent/clear",
						Summary:      "清空",
						BodyMode:     model.BodyModeFileOrData,
						BodyRequired: true,
						BodySchemaFields: []model.BodyField{
							{Name: "deliveryRecId", Type: "integer", Description: "目的地ID"},
							{Name: "deliveryRecNo", Type: "string", Description: "目的地编号"},
						},
					},
				},
			},
		},
	}

	if err := render.Project(dir, "github.com/acme/les-cli", app); err != nil {
		t.Fatalf("render: %v", err)
	}

	skillContent, err := os.ReadFile(filepath.Join(dir, "skills", "te_mm_mri_current", "SKILL.md"))
	if err != nil {
		t.Fatalf("read generated skill markdown: %v", err)
	}
	skillText := string(skillContent)
	for _, want := range []string{
		"**Request Body Schema:**",
		"| `deliveryRecId` | no | 目的地ID (integer) |",
		"| `deliveryRecNo` | no | 目的地编号 (string) |",
		`--data '{"deliveryRecId": 123, "deliveryRecNo": "value"}'`,
		"--file assets/demo-request.json",
	} {
		if !strings.Contains(skillText, want) {
			t.Fatalf("generated SKILL.md missing %q:\n%s", want, skillText)
		}
	}

	routingContent, err := os.ReadFile(filepath.Join(dir, "skills", "te_mm_mri_current", "references", "command-routing.md"))
	if err != nil {
		t.Fatalf("read generated command routing: %v", err)
	}
	if !strings.Contains(string(routingContent), "`les te-mm-mri-current clear`") {
		t.Fatalf("generated command routing missing clear command:\n%s", routingContent)
	}

	demoContent, err := os.ReadFile(filepath.Join(dir, "skills", "te_mm_mri_current", "assets", "demo-request.json"))
	if err != nil {
		t.Fatalf("read generated demo request: %v", err)
	}
	demoText := string(demoContent)
	for _, want := range []string{
		`"deliveryRecId": 123`,
		`"deliveryRecNo": "value"`,
	} {
		if !strings.Contains(demoText, want) {
			t.Fatalf("generated demo request missing %q:\n%s", want, demoText)
		}
	}
}

func TestRenderRustProjectSkillUsesActualCliFlagNames(t *testing.T) {
	dir := t.TempDir()
	app := model.App{
		Name: "les",
		Groups: []model.Group{
			{
				Name: "te-mm-mri-current",
				Operations: []model.Operation{
					{
						CommandName: "page",
						Method:      "GET",
						Path:        "/les/api/teMmMriCurrent/page",
						Summary:     "分页查询",
						Parameters: []model.Parameter{
							{Name: "current", In: "query", Required: true, Type: "integer"},
							{Name: "pageSize", In: "query", Required: true, Type: "integer", Description: "页码"},
						},
					},
				},
			},
		},
	}

	if err := render.Project(dir, "github.com/acme/les-cli", app, "rust"); err != nil {
		t.Fatalf("render: %v", err)
	}

	skillContent, err := os.ReadFile(filepath.Join(dir, "skills", "te_mm_mri_current", "SKILL.md"))
	if err != nil {
		t.Fatalf("read generated skill markdown: %v", err)
	}
	skillText := string(skillContent)
	for _, want := range []string{
		`--current "1"`,
		`--pagesize "25"`,
		"| `--pagesize` | yes | 页码 (integer) |",
	} {
		if !strings.Contains(skillText, want) {
			t.Fatalf("generated Rust SKILL.md missing %q:\n%s", want, skillText)
		}
	}
	if strings.Contains(skillText, "--pageSize") {
		t.Fatalf("generated Rust SKILL.md contains non-existent --pageSize flag:\n%s", skillText)
	}
}

func TestRenderProjectCompilesWhenGroupNameContainsHyphen(t *testing.T) {
	dir := t.TempDir()
	app := model.App{
		Name: "quark",
		Groups: []model.Group{
			{
				Name:        "tool-quark-web-search",
				PackageName: "tool_quark_web_search",
				Backend:     "mcp-streamable-http",
				Endpoint:    "https://example.com/mcp",
				Operations: []model.Operation{
					{CommandName: "quark", Method: "MCP", Path: "/quark_web_search"},
				},
			},
		},
	}

	if err := render.Project(dir, "github.com/acme/quark-cli", app); err != nil {
		t.Fatalf("render: %v", err)
	}

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(),
		"GOCACHE="+filepath.Join(t.TempDir(), "gocache"),
		"GOTOOLCHAIN=local",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated project should compile for hyphenated groups, got %v, output: %s", err, string(out))
	}
}
