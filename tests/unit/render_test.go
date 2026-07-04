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
		"skills/leave/generation-report.md",
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
		"generation-report.md",
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

func TestRenderProjectCanGenerateChineseSkillPackage(t *testing.T) {
	dir := t.TempDir()
	app := model.App{
		Name: "one",
		Groups: []model.Group{
			{
				Name: "leave",
				Operations: []model.Operation{
					{
						CommandName: "request",
						Method:      "POST",
						Path:        "/leaves",
						Parameters: []model.Parameter{
							{Name: "userId", In: "query", Required: true, Type: "string"},
						},
					},
				},
			},
		},
	}

	if err := render.Project(dir, "github.com/acme/one-cli", app, "go", "zh"); err != nil {
		t.Fatalf("render: %v", err)
	}

	skillContent, err := os.ReadFile(filepath.Join(dir, "skills", "leave", "SKILL.md"))
	if err != nil {
		t.Fatalf("read generated skill markdown: %v", err)
	}
	skillText := string(skillContent)
	for _, want := range []string{
		"## 包文件",
		"## 前置条件",
		"## 命令",
		"**参数：**",
		"| `--userId` | 是 | userId（string） |",
		"| `one leave request` | request | 写入 |",
		"generation-report.md",
		"OPENCLI_BASE_URL",
	} {
		if !strings.Contains(skillText, want) {
			t.Fatalf("generated Chinese SKILL.md missing %q:\n%s", want, skillText)
		}
	}

	reportContent, err := os.ReadFile(filepath.Join(dir, "skills", "leave", "generation-report.md"))
	if err != nil {
		t.Fatalf("read generated report: %v", err)
	}
	reportText := string(reportContent)
	for _, want := range []string{
		"# 生成报告：leave",
		"`leave` 缺少分组描述",
		"`request` 的参数 `userId` 缺少说明",
	} {
		if !strings.Contains(reportText, want) {
			t.Fatalf("generated Chinese report missing %q:\n%s", want, reportText)
		}
	}

	demoContent, err := os.ReadFile(filepath.Join(dir, "skills", "leave", "assets", "demo-request.json"))
	if err != nil {
		t.Fatalf("read generated demo request: %v", err)
	}
	if !strings.Contains(string(demoContent), `"demo": true`) {
		t.Fatalf("generated Chinese demo request missing fallback JSON:\n%s", demoContent)
	}
}

func TestGenerationReportListsAllDocumentationIssues(t *testing.T) {
	dir := t.TempDir()
	app := model.App{
		Name: "one",
		Groups: []model.Group{
			{
				Name: "leave",
				Operations: []model.Operation{
					{
						CommandName: "request",
						Method:      "POST",
						Path:        "/leaves/{leaveId}",
						Parameters: []model.Parameter{
							{Name: "userId", In: "query", Required: true, Type: "string"},
							{Name: "filter", In: "query", Description: "Filter value"},
							{Name: "sessionId", In: "cookie", Description: "Session ID", Type: "string"},
						},
					},
				},
			},
			{
				Name: "attendance",
				Operations: []model.Operation{
					{
						CommandName: "punch",
						Method:      "POST",
						Path:        "/attendance/punch",
						Summary:     "Punch attendance",
						Parameters: []model.Parameter{
							{Name: "tenantId", In: "path", Description: "Tenant ID", Type: "string"},
						},
						BodySchemaFields: []model.BodyField{
							{Name: "siteCode"},
						},
					},
				},
			},
		},
	}

	if err := render.Project(dir, "github.com/acme/one-cli", app); err != nil {
		t.Fatalf("render: %v", err)
	}

	reportContent, err := os.ReadFile(filepath.Join(dir, "skills", "leave", "generation-report.md"))
	if err != nil {
		t.Fatalf("read generated report: %v", err)
	}
	reportText := string(reportContent)
	for _, want := range []string{
		"## All API Documentation Issues",
		"`leave` is missing a group description",
		"`request` parameter `userId` is missing a description",
		"`request` parameter `filter` is missing a type",
		"`request` parameter `sessionId` uses unsupported location `cookie`",
		"`request` path parameter `{leaveId}` is missing a matching `in: path` parameter",
		"`attendance` is missing a group description",
		"`punch` request body field `siteCode` is missing a description",
		"`punch` request body field `siteCode` is missing a type",
		"`punch` declares path parameter `tenantId` that is not present in path `/attendance/punch`",
		"`punch` path parameter `tenantId` should be required",
	} {
		if !strings.Contains(reportText, want) {
			t.Fatalf("generated report missing %q:\n%s", want, reportText)
		}
	}
}

func TestRenderRustDisambiguatesIdentifiers(t *testing.T) {
	dir := t.TempDir()
	app := model.App{
		Name:    "one",
		Version: "0.0.1",
		Groups: []model.Group{
			{
				Name:        "过点车辆定时任务(线边、sps)",
				PackageName: "default",
				Operations: []model.Operation{
					{
						CommandName: "update",
						Method:      "PUT",
						Path:        "/items/{id}",
						Parameters: []model.Parameter{
							{Name: "id", In: "path", Required: true, Type: "string"},
						},
						BodyMode: model.BodyModeSimpleJSON,
						BodyFields: []model.BodyField{
							{Name: "id", Type: "string"},
						},
					},
				},
			},
			{
				Name:        "另一个分组",
				PackageName: "default",
				Operations: []model.Operation{
					{CommandName: "list", Method: "GET", Path: "/items"},
				},
			},
		},
	}

	if err := render.Project(dir, "github.com/acme/one-cli", app, "rust"); err != nil {
		t.Fatalf("render rust: %v", err)
	}

	cliContent, err := os.ReadFile(filepath.Join(dir, "src", "cli.rs"))
	if err != nil {
		t.Fatalf("read generated cli.rs: %v", err)
	}
	cliText := string(cliContent)
	for _, want := range []string{
		`#[command(name = "过点车辆定时任务(线边、sps)")]`,
		"Default {",
		"Default2 {",
	} {
		if !strings.Contains(cliText, want) {
			t.Fatalf("generated cli.rs missing %q:\n%s", want, cliText)
		}
	}
	if strings.Contains(cliText, "过点车辆定时任务(线边、sps) {") {
		t.Fatalf("generated cli.rs used display name as Rust variant:\n%s", cliText)
	}

	modContent, err := os.ReadFile(filepath.Join(dir, "src", "commands", "mod.rs"))
	if err != nil {
		t.Fatalf("read generated commands/mod.rs: %v", err)
	}
	modText := string(modContent)
	for _, want := range []string{
		"pub mod default;",
		"pub mod default_2;",
	} {
		if !strings.Contains(modText, want) {
			t.Fatalf("generated commands/mod.rs missing %q:\n%s", want, modText)
		}
	}

	commandContent, err := os.ReadFile(filepath.Join(dir, "src", "commands", "default.rs"))
	if err != nil {
		t.Fatalf("read generated command: %v", err)
	}
	commandText := string(commandContent)
	for _, want := range []string{
		`#[arg(long = "id")]`,
		"pub id: String,",
		`#[arg(long = "body-id")]`,
		"pub body_id: Option<String>,",
		`parts.push(format!("\"id\":{value}"));`,
	} {
		if !strings.Contains(commandText, want) {
			t.Fatalf("generated command missing %q:\n%s", want, commandText)
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

func TestRenderRustProjectSkillFlagNamesMatchCodeForUnderscoreNames(t *testing.T) {
	dir := t.TempDir()
	app := model.App{
		Name: "les",
		Groups: []model.Group{
			{
				Name: "orders",
				Operations: []model.Operation{
					{
						CommandName: "create",
						Method:      "POST",
						Path:        "/orders",
						BodyMode:    model.BodyModeSimpleJSON,
						Parameters: []model.Parameter{
							{Name: "user_id", In: "query", Required: true, Type: "string"},
							{Name: "delivery-rec", In: "query", Required: false, Type: "string"},
						},
						BodyFields: []model.BodyField{
							{Name: "ref_no", Type: "string", Description: "单据编号"},
						},
					},
				},
			},
		},
	}

	if err := render.Project(dir, "github.com/acme/les-cli", app, "rust"); err != nil {
		t.Fatalf("render: %v", err)
	}

	skillText, err := os.ReadFile(filepath.Join(dir, "skills", "orders", "SKILL.md"))
	if err != nil {
		t.Fatalf("read generated skill markdown: %v", err)
	}
	codeText, err := os.ReadFile(filepath.Join(dir, "src", "commands", "orders.rs"))
	if err != nil {
		t.Fatalf("read generated rust command: %v", err)
	}

	for _, flag := range []string{"--user-id", "--delivery-rec", "--ref-no"} {
		if !strings.Contains(string(skillText), flag) {
			t.Fatalf("generated Rust SKILL.md missing %q:\n%s", flag, skillText)
		}
		if !strings.Contains(string(codeText), `long = "`+strings.TrimPrefix(flag, "--")+`"`) {
			t.Fatalf("generated Rust code missing flag %q:\n%s", flag, codeText)
		}
	}

	for _, stale := range []string{"--user_id", "--delivery_rec", "--ref_no"} {
		if strings.Contains(string(skillText), stale) {
			t.Fatalf("generated Rust SKILL.md contains underscore flag %q that clap will not accept:\n%s", stale, skillText)
		}
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
