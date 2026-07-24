package unit_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"one-cli/internal/configgen"
	"one-cli/internal/model"
	"one-cli/internal/openapi"
	"one-cli/internal/planner"
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
	serviceContent, err := os.ReadFile(filepath.Join(dir, "internal", "leave", "service.go"))
	if err != nil {
		t.Fatalf("read generated service.go: %v", err)
	}
	if !strings.Contains(string(serviceContent), "applyHeaders(req, cli.RequestHeaders())") {
		t.Fatalf("generated service.go should apply root --header values:\n%s", serviceContent)
	}
	rootContent, err := os.ReadFile(filepath.Join(dir, "internal", "cli", "root.go"))
	if err != nil {
		t.Fatalf("read generated root.go: %v", err)
	}
	if !strings.Contains(string(rootContent), `StringArrayVarP(&requestHeaders, "header", "H", nil`) {
		t.Fatalf("generated root.go should bind -H as shorthand for --header:\n%s", rootContent)
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
	if _, err := os.Stat(filepath.Join(dir, "skills", "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("single-group project should not generate skills router, got err: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "skills", "README.md")); err != nil {
		t.Fatalf("missing generated skills index: %v", err)
	}
	for _, rel := range []string{
		"skills/leave/README.md",
		"skills/leave/assets/demo-request.json",
		"skills/leave/references/command-routing.md",
		"skills/leave/references/commands.md",
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
	if !strings.Contains(skillText, "references/commands.md") {
		t.Fatalf("generated skill markdown missing complete command reference:\n%s", skillText)
	}
	commandsContent, err := os.ReadFile(filepath.Join(dir, "skills", "leave", "references", "commands.md"))
	if err != nil {
		t.Fatalf("read generated commands.md: %v", err)
	}
	for _, want := range []string{"# leave Command Reference", "one leave list", "one leave list --help", "GET", "/leaves"} {
		if !strings.Contains(string(commandsContent), want) {
			t.Fatalf("generated commands.md missing %q:\n%s", want, commandsContent)
		}
	}
	for _, unwanted := range []string{
		"## Global Request Headers",
		"ACCESS-STATUS",
	} {
		if strings.Contains(skillText, unwanted) {
			t.Fatalf("generated skill markdown should omit unused header guidance %q:\n%s", unwanted, skillText)
		}
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
		"./bin/one skills list",
		"./bin/one skills read <skill>",
		"./bin/one skills --skills-dir /path/to/skills read <skill>",
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
	skillsIndexContent, err := os.ReadFile(filepath.Join(dir, "skills", "README.md"))
	if err != nil {
		t.Fatalf("read generated skills index: %v", err)
	}
	skillsIndexText := string(skillsIndexContent)
	for _, want := range []string{
		"# Generated Skills",
		"one skills list",
		"one skills read <skill>",
		"| `leave` | 1 | [`SKILL.md`](leave/SKILL.md) |",
	} {
		if !strings.Contains(skillsIndexText, want) {
			t.Fatalf("generated skills index missing %q:\n%s", want, skillsIndexText)
		}
	}
	headerTestPath := filepath.Join(dir, "internal", "leave", "service_header_test.go")
	headerTest := `package leave

import (
	"net/http/httptest"
	"testing"
)

func TestApplyHeadersAcceptsEqualsSyntax(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.test/leaves", nil)
	if err := applyHeaders(req, []string{"ACCESS-STATUS=inner"}); err != nil {
		t.Fatalf("applyHeaders: %v", err)
	}
	if got := req.Header.Get("ACCESS-STATUS"); got != "inner" {
		t.Fatalf("ACCESS-STATUS = %q, want inner", got)
	}
}

`
	if err := os.WriteFile(headerTestPath, []byte(headerTest), 0o644); err != nil {
		t.Fatalf("write generated header test: %v", err)
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
		"skills",
		"--header",
		"--trace",
		"--version",
	} {
		if !strings.Contains(helpText, want) {
			t.Fatalf("generated project --help missing %q:\n%s", want, helpText)
		}
	}

	commandHelp := exec.Command(binary, "leave", "list", "--help")
	out, err = commandHelp.CombinedOutput()
	if err != nil {
		t.Fatalf("generated command --help should succeed, got %v, output: %s", err, string(out))
	}
	commandHelpText := string(out)
	for _, want := range []string{
		"--header",
		"-H,",
		"Name: Value",
		"Name=Value",
	} {
		if !strings.Contains(commandHelpText, want) {
			t.Fatalf("generated command --help missing %q:\n%s", want, commandHelpText)
		}
	}

	skillsList := exec.Command(binary, "skills", "list")
	skillsList.Dir = dir
	out, err = skillsList.CombinedOutput()
	if err != nil {
		t.Fatalf("generated skills list should succeed, got %v, output: %s", err, string(out))
	}
	if got := string(out); !strings.Contains(got, "leave\t") || !strings.Contains(got, "leave commands for one") {
		t.Fatalf("generated skills list output = %q, want leave skill with description", got)
	}

	skillsRead := exec.Command(binary, "skills", "read", "leave")
	skillsRead.Dir = dir
	out, err = skillsRead.CombinedOutput()
	if err != nil {
		t.Fatalf("generated skills read should succeed, got %v, output: %s", err, string(out))
	}
	if got := string(out); !strings.Contains(got, "name: leave") || !strings.Contains(got, "## Commands") {
		t.Fatalf("generated skills read output missing SKILL.md content:\n%s", got)
	}

	manualSkillContent := strings.Replace(skillText, "## Important Notes", "## Business Override\n\nManual business edit.\n\n## Important Notes", 1)
	if err := os.WriteFile(filepath.Join(dir, "skills", "leave", "SKILL.md"), []byte(manualSkillContent), 0o644); err != nil {
		t.Fatalf("write manual skill edit: %v", err)
	}
	skillsRead = exec.Command(binary, "skills", "read", "leave")
	skillsRead.Dir = dir
	out, err = skillsRead.CombinedOutput()
	if err != nil {
		t.Fatalf("generated skills read after manual edit should succeed, got %v, output: %s", err, string(out))
	}
	if got := string(out); !strings.Contains(got, "Manual business edit.") {
		t.Fatalf("generated skills read should reflect disk edits:\n%s", got)
	}

	hiddenSkillsDir := filepath.Join(dir, "skills.hidden")
	if err := os.Rename(filepath.Join(dir, "skills"), hiddenSkillsDir); err != nil {
		t.Fatalf("hide generated skills directory: %v", err)
	}

	nestedBinary := filepath.Join(dir, "target", "release", "one")
	if err := os.MkdirAll(filepath.Dir(nestedBinary), 0o755); err != nil {
		t.Fatalf("create nested binary dir: %v", err)
	}
	nestedBuild := exec.Command("go", "build", "-o", nestedBinary, "./cmd/one")
	nestedBuild.Dir = dir
	nestedBuild.Env = append(nestedBuild.Environ(),
		"GOCACHE="+filepath.Join(t.TempDir(), "gocache"),
		"GOTOOLCHAIN=local",
	)
	out, err = nestedBuild.CombinedOutput()
	if err != nil {
		t.Fatalf("generated nested project binary should build, got %v, output: %s", err, string(out))
	}
	nestedSkillsList := exec.Command(nestedBinary, "skills", "--skills-dir", hiddenSkillsDir, "list")
	nestedSkillsList.Dir = filepath.Join(dir, "target", "release")
	out, err = nestedSkillsList.CombinedOutput()
	if err != nil {
		t.Fatalf("nested generated skills list should read explicit skills dir, got %v, output: %s", err, string(out))
	}
	if got := string(out); !strings.Contains(got, "leave\t") {
		t.Fatalf("nested generated skills list output = %q, want leave skill", got)
	}

	nestedSkillsRead := exec.Command(nestedBinary, "skills", "--skills-dir", hiddenSkillsDir, "read", "leave")
	nestedSkillsRead.Dir = filepath.Join(dir, "target", "release")
	out, err = nestedSkillsRead.CombinedOutput()
	if err != nil {
		t.Fatalf("nested generated skills read should read explicit skills dir, got %v, output: %s", err, string(out))
	}
	if got := string(out); !strings.Contains(got, "name: leave") || !strings.Contains(got, "Manual business edit.") {
		t.Fatalf("nested generated skills read output missing SKILL.md content:\n%s", got)
	}
}

func TestRenderGoProjectCompilesBodyAndFileNameCollisions(t *testing.T) {
	dir := t.TempDir()
	app := model.App{
		Name: "collision-cli",
		Groups: []model.Group{{
			Name: "imports",
			Operations: []model.Operation{{
				CommandName:  "create",
				Method:       "POST",
				Path:         "/imports",
				BodyMode:     model.BodyModeSimpleJSON,
				BodyRequired: true,
				Parameters: []model.Parameter{
					{Name: "data", In: "query", Type: "string"},
					{Name: "user.id", In: "query", Type: "string"},
					{Name: "user-id", In: "query", Type: "string"},
				},
				BodyFields: []model.BodyField{
					{Name: "data", Type: "string"},
					{Name: "description", Type: "string"},
				},
				FileFields: []model.BodyField{{Name: "file", Type: "string", Format: "binary"}},
			}},
		}},
	}
	if err := render.Project(dir, "github.com/acme/collision-cli", app); err != nil {
		t.Fatalf("render: %v", err)
	}

	command, err := os.ReadFile(filepath.Join(dir, "internal", "imports", "command.go"))
	if err != nil {
		t.Fatalf("read command: %v", err)
	}
	text := string(command)
	if strings.Count(text, `"data"`) == 0 || strings.Contains(text, `JSON body field: data`) {
		t.Fatalf("body field data should fall back to --data without duplicate registration:\n%s", text)
	}
	if !strings.Contains(text, `StringVar(&uploadFile, "file", "", "File upload`) {
		t.Fatalf("binary body field should register a real --file upload flag:\n%s", text)
	}
	service, err := os.ReadFile(filepath.Join(dir, "internal", "imports", "service.go"))
	if err != nil {
		t.Fatalf("read service: %v", err)
	}
	if !strings.Contains(string(service), "multipart.NewWriter") {
		t.Fatalf("binary body field should generate multipart request construction:\n%s", service)
	}

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(), "GOCACHE="+filepath.Join(t.TempDir(), "gocache"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated collision project should compile: %v\n%s", err, out)
	}
}

func TestGeneratedSkillDocumentsResponseSchemas(t *testing.T) {
	dir := t.TempDir()
	app := model.App{
		Name: "analysis-cli",
		Groups: []model.Group{
			{
				Name: "chery-global",
				Operations: []model.Operation{
					{
						CommandName: "quality",
						Method:      "GET",
						Path:        "/cheryGlobal/aiAgent/qulity",
						Summary:     "质量",
						Responses: []model.Response{
							{
								Status:      "200",
								ContentType: "*/*",
								Description: "OK",
								Schemas: []model.Schema{
									{
										Name: "ResultAiQualityVO",
										Type: "object",
										Fields: []model.SchemaField{
											{Name: "code", Type: "integer"},
											{Name: "data", Type: "AiQualityVO"},
										},
									},
									{
										Name:        "AiQualityVO",
										Description: "质量",
										Type:        "object",
										Fields: []model.SchemaField{
											{Name: "qualityPerformanceVOS", Type: "QualityPerformanceVO[]", Description: "价值链信息"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if err := render.Project(dir, "github.com/acme/analysis-cli", app, "go", "zh"); err != nil {
		t.Fatalf("render: %v", err)
	}
	skillContent, err := os.ReadFile(filepath.Join(dir, "skills", "chery-global", "SKILL.md"))
	if err != nil {
		t.Fatalf("read generated skill: %v", err)
	}
	skillText := string(skillContent)
	for _, want := range []string{
		"name: chery-global",
		"analysis-cli skills read chery-global",
		"状态码：`200`",
		"内容类型：`*/*`",
		"`ResultAiQualityVO`",
		"| `data` | `AiQualityVO` |",
		"`AiQualityVO` — 质量",
		"| `qualityPerformanceVOS` | `QualityPerformanceVO[]` | 价值链信息 |",
	} {
		if !strings.Contains(skillText, want) {
			t.Fatalf("generated skill missing response documentation %q:\n%s", want, skillText)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "skills", "chery_global")); !os.IsNotExist(err) {
		t.Fatalf("legacy underscore skill directory should not exist, got err: %v", err)
	}
}

func TestRenderProjectGeneratesSkillsRouterForMultipleGroups(t *testing.T) {
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
			{
				Name: "attendance",
				Operations: []model.Operation{
					{CommandName: "punch", Method: "POST", Path: "/attendance/punch"},
				},
			},
		},
	}

	if err := render.Project(dir, "github.com/acme/one-cli", app); err != nil {
		t.Fatalf("render: %v", err)
	}

	skillsRouterContent, err := os.ReadFile(filepath.Join(dir, "skills", "SKILL.md"))
	if err != nil {
		t.Fatalf("read generated skills router: %v", err)
	}
	skillsRouterText := string(skillsRouterContent)
	for _, want := range []string{
		"name: one-skills",
		"Skills Router",
		"one skills list",
		"| `leave` | 1 | [`leave/SKILL.md`](leave/SKILL.md) |",
		"| `attendance` | 1 | [`attendance/SKILL.md`](attendance/SKILL.md) |",
	} {
		if !strings.Contains(skillsRouterText, want) {
			t.Fatalf("generated skills router missing %q:\n%s", want, skillsRouterText)
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
		"| `--userId` | 是 | userId（查询参数；string） |",
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
		"- [ ] `request`",
		"  - 参数 `userId`：缺少说明",
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
		"- [ ] `request`",
		"  - parameter `userId`: missing description",
		"  - parameter `filter`: missing type",
		"  - parameter `sessionId`: unsupported location `cookie`",
		"  - path parameter `{leaveId}`: missing matching `in: path` parameter",
		"`attendance` is missing a group description",
		"- [ ] `punch`",
		"  - request body field `siteCode`: missing description; missing type",
		"  - path parameter `tenantId`: not present in path `/attendance/punch`; should be required",
	} {
		if !strings.Contains(reportText, want) {
			t.Fatalf("generated report missing %q:\n%s", want, reportText)
		}
	}
	for _, notWant := range []string{
		"`request` parameter `userId` is missing a description",
		"`punch` request body field `siteCode` is missing a description",
		"`punch` request body field `siteCode` is missing a type",
	} {
		if strings.Contains(reportText, notWant) {
			t.Fatalf("generated report should group repeated API issue prefixes %q:\n%s", notWant, reportText)
		}
	}
}

func TestGenerationReportOmitsAllIssuesSectionForSingleGroup(t *testing.T) {
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

	if err := render.Project(dir, "github.com/acme/one-cli", app, "go", "zh"); err != nil {
		t.Fatalf("render: %v", err)
	}

	reportContent, err := os.ReadFile(filepath.Join(dir, "skills", "leave", "generation-report.md"))
	if err != nil {
		t.Fatalf("read generated report: %v", err)
	}
	reportText := string(reportContent)
	if strings.Contains(reportText, "## 全部 API 文档问题") {
		t.Fatalf("single-group report should not duplicate all API issues section:\n%s", reportText)
	}
	if !strings.Contains(reportText, "## 检测到的缺口") {
		t.Fatalf("single-group report should keep detected gaps section:\n%s", reportText)
	}
}

func TestGenerationReportUsesConfiguredBodyFieldSupplements(t *testing.T) {
	dir := t.TempDir()
	required := true
	doc := openapi.Document{
		Operations: []openapi.Operation{
			{
				Method:      "POST",
				Path:        "/leaves",
				Tag:         "leave",
				OperationID: "createLeave",
				Summary:     "Create leave request",
				RequestBody: openapi.RequestBody{
					ContentTypes:  []string{"application/json"},
					HasJSONSchema: true,
					IsSimpleJSON:  true,
					JSONSchemaFields: []openapi.BodyField{
						{Name: "reason", RequiredUnknown: true},
					},
				},
			},
		},
	}
	app := planner.Build(doc, configgen.Config{
		Overrides: configgen.OverrideConfig{
			BodyFields: map[string][]configgen.BodyField{
				"leave.create": {
					{
						Name:        "reason",
						Description: "Leave reason",
						Required:    &required,
						Type:        "string",
					},
				},
			},
		},
	})

	if err := render.Project(dir, "github.com/acme/one-cli", app); err != nil {
		t.Fatalf("render: %v", err)
	}

	reportContent, err := os.ReadFile(filepath.Join(dir, "skills", "leave", "generation-report.md"))
	if err != nil {
		t.Fatalf("read generated report: %v", err)
	}
	reportText := string(reportContent)
	for _, notWant := range []string{
		"`create` request body field `reason` is missing a description",
		"`create` request body field `reason` is missing a type",
		"`create` request body field `reason` does not declare whether it is required",
	} {
		if strings.Contains(reportText, notWant) {
			t.Fatalf("generated report should honor configured body field supplement %q:\n%s", notWant, reportText)
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

	skillsRouterContent, err := os.ReadFile(filepath.Join(dir, "skills", "SKILL.md"))
	if err != nil {
		t.Fatalf("read generated Rust skills router: %v", err)
	}
	skillsRouterText := string(skillsRouterContent)
	for _, want := range []string{
		"name: one-skills",
		"Skills Router",
		"one skills list",
		"| `sps` | 1 | [`sps/SKILL.md`](sps/SKILL.md) |",
		"| `default-2` | 1 | [`default-2/SKILL.md`](default-2/SKILL.md) |",
	} {
		if !strings.Contains(skillsRouterText, want) {
			t.Fatalf("generated Rust skills router missing %q:\n%s", want, skillsRouterText)
		}
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

func TestRenderRustUsesDataSourcesAndReservesFileForUploads(t *testing.T) {
	dir := t.TempDir()
	app := model.App{Name: "assets", Groups: []model.Group{{
		Name: "assets", Operations: []model.Operation{{
			CommandName: "create",
			Method:      "POST",
			Path:        "/assets",
			BodyMode:    model.BodyModeSimpleJSON,
			BodyFields: []model.BodyField{
				{Name: "data", Type: "string"},
				{Name: "description", Type: "string"},
			},
			FileFields: []model.BodyField{{Name: "asset", Type: "string", Format: "binary"}},
		}},
	}}}

	if err := render.Project(dir, "github.com/acme/assets-cli", app, "rust"); err != nil {
		t.Fatalf("render rust: %v", err)
	}
	command, err := os.ReadFile(filepath.Join(dir, "src", "commands", "assets.rs"))
	if err != nil {
		t.Fatalf("read command: %v", err)
	}
	text := string(command)
	for _, want := range []string{
		`#[arg(long = "data")]`,
		`#[arg(long = "file")]`,
		`client::resolve_input(data)?`,
		`client::request_multipart(`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("generated Rust command missing %q:\n%s", want, text)
		}
	}
	for _, notWant := range []string{"body_file", `#[arg(long = "body-data")]`} {
		if strings.Contains(text, notWant) {
			t.Fatalf("generated Rust command contains legacy/colliding %q:\n%s", notWant, text)
		}
	}

	client, err := os.ReadFile(filepath.Join(dir, "src", "client.rs"))
	if err != nil {
		t.Fatalf("read client: %v", err)
	}
	clientText := string(client)
	for _, want := range []string{"pub fn resolve_input", "pub async fn request_multipart", "reqwest::multipart"} {
		if !strings.Contains(clientText, want) {
			t.Fatalf("generated Rust client missing %q:\n%s", want, clientText)
		}
	}
}

func TestRenderRustNoAuthClientDoesNotMarkHeadersMutable(t *testing.T) {
	dir := t.TempDir()
	app := model.App{
		Name: "one",
		Auth: model.Auth{Type: model.AuthTypeNone},
		Groups: []model.Group{
			{
				Name: "items",
				Operations: []model.Operation{
					{CommandName: "list", Method: "GET", Path: "/items"},
				},
			},
		},
	}

	if err := render.Project(dir, "github.com/acme/one-cli", app, "rust"); err != nil {
		t.Fatalf("render rust: %v", err)
	}

	clientContent, err := os.ReadFile(filepath.Join(dir, "src", "client.rs"))
	if err != nil {
		t.Fatalf("read generated client.rs: %v", err)
	}
	clientText := string(clientContent)
	if strings.Contains(clientText, "let mut headers = merged_headers(headers);") {
		t.Fatalf("generated no-auth Rust client should not mark merged headers mutable:\n%s", clientText)
	}
	if !strings.Contains(clientText, "let headers = merged_headers(headers);") {
		t.Fatalf("generated no-auth Rust client missing immutable merged headers binding:\n%s", clientText)
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
							{Name: "X-Tenant-ID", In: "header", Type: "string", Description: "Tenant identifier"},
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
		"## Global Request Headers",
		`one auth <command> -H "Name: Value"`,
		"## Command Routing",
		"## Core Concepts",
		"## Important Notes",
		"## Common Workflows",
		"### one auth me",
		`--header "X-Tenant-ID: <value>"`,
		"**Parameters:**",
		"<!-- MANUAL:",
	} {
		if !strings.Contains(skillText, want) {
			t.Fatalf("generated SKILL.md missing %q:\n%s", want, skillText)
		}
	}
	for _, unwanted := range []string{
		"ACCESS-STATUS",
		`one auth <command> --header`,
	} {
		if strings.Contains(skillText, unwanted) {
			t.Fatalf("generated SKILL.md should omit redundant header example %q:\n%s", unwanted, skillText)
		}
	}
}

func TestRenderProjectSkillOmitsTokenAuthorizationHeaderGuidance(t *testing.T) {
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
							{Name: "Authorization", In: "header", Type: "string"},
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
	for _, unwanted := range []string{"## Global Request Headers", `--header "Authorization: <value>"`} {
		if strings.Contains(skillText, unwanted) {
			t.Fatalf("generated SKILL.md should rely on OPENCLI_AUTH_TOKEN instead of documenting %q:\n%s", unwanted, skillText)
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
							{
								Name: "workflowBaseInfo", Type: "WorkflowBaseInfo", JSONSchemaName: "WorkflowBaseInfo",
								JSONFields: []model.BodyField{
									{Name: "workflowId", Type: "string", Required: true, Description: "Workflow identifier"},
									{Name: "workflowName", Type: "string", Description: "Workflow name"},
								},
							},
						},
					},
				},
			},
		},
	}

	if err := render.Project(dir, "github.com/acme/les-cli", app); err != nil {
		t.Fatalf("render: %v", err)
	}

	skillContent, err := os.ReadFile(filepath.Join(dir, "skills", "te-mm-mri-current", "SKILL.md"))
	if err != nil {
		t.Fatalf("read generated skill markdown: %v", err)
	}
	skillText := string(skillContent)
	for _, want := range []string{
		"**Request Body Schema:**",
		"| `deliveryRecId` | no | 目的地ID (integer) |",
		"| `deliveryRecNo` | no | 目的地编号 (string) |",
		`--data '{"deliveryRecId": 123, "deliveryRecNo": "value", "workflowBaseInfo": {"workflowId": "value", "workflowName": "John Doe"}}'`,
		"--data @assets/demo-request.json",
		"**`workflowBaseInfo` JSON Fields (`WorkflowBaseInfo`):**",
		"| `workflowId` | yes | `string` | Workflow identifier |",
	} {
		if !strings.Contains(skillText, want) {
			t.Fatalf("generated SKILL.md missing %q:\n%s", want, skillText)
		}
	}

	routingContent, err := os.ReadFile(filepath.Join(dir, "skills", "te-mm-mri-current", "references", "command-routing.md"))
	if err != nil {
		t.Fatalf("read generated command routing: %v", err)
	}
	if !strings.Contains(string(routingContent), "`les te-mm-mri-current clear`") {
		t.Fatalf("generated command routing missing clear command:\n%s", routingContent)
	}

	demoContent, err := os.ReadFile(filepath.Join(dir, "skills", "te-mm-mri-current", "assets", "demo-request.json"))
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

func TestRenderProjectsHonorGETFormRequestBodyAndDocumentFlags(t *testing.T) {
	app := model.App{
		Name: "bpm-cli",
		Groups: []model.Group{
			{
				Name: "bpm",
				Operations: []model.Operation{
					{
						CommandName:  "task-query",
						Method:       "GET",
						Path:         "/tasks",
						Summary:      "Query tasks",
						BodyMode:     model.BodyModeFormURLEncoded,
						BodyRequired: true,
						BodyFields: []model.BodyField{
							{
								Name: "tqm", Type: "string", Required: true, Description: "query JSON", Example: `{"owner":"user-123","taskState":1}`,
								JSONSchemaName: "TaskQueryModel",
								JSONFields: []model.BodyField{
									{Name: "owner", Type: "string", Required: true, Description: "User uid"},
									{Name: "taskState", Type: "integer", Description: "1 means pending"},
								},
							},
							{Name: "firstRow", Type: "integer", Required: true},
							{Name: "rowCount", Type: "integer", Required: true},
						},
					},
				},
			},
		},
	}

	goDir := t.TempDir()
	if err := render.Project(goDir, "github.com/acme/bpm-cli", app); err != nil {
		t.Fatalf("render Go: %v", err)
	}
	goCommand := mustReadFile(t, filepath.Join(goDir, "internal", "bpm", "command.go"))
	goService := mustReadFile(t, filepath.Join(goDir, "internal", "bpm", "service.go"))
	goSkill := mustReadFile(t, filepath.Join(goDir, "skills", "bpm", "SKILL.md"))
	for _, want := range []string{`"tqm"`, `"first-row"`, `"row-count"`, `MarkFlagRequired("tqm")`} {
		if !strings.Contains(goCommand, want) {
			t.Fatalf("generated Go command missing %q:\n%s", want, goCommand)
		}
	}
	for _, want := range []string{`url.Values{}`, `values.Set("tqm"`, `values.Set("firstRow"`, `"application/x-www-form-urlencoded"`} {
		if !strings.Contains(goService, want) {
			t.Fatalf("generated Go service missing %q:\n%s", want, goService)
		}
	}
	for _, want := range []string{"--tqm", "--first-row", "--row-count", "form field"} {
		if !strings.Contains(goSkill, want) {
			t.Fatalf("generated Skill missing %q:\n%s", want, goSkill)
		}
	}
	if !strings.Contains(goSkill, `--tqm '{"owner":"user-123","taskState":1}'`) {
		t.Fatalf("generated Skill should use the documented tqm JSON example:\n%s", goSkill)
	}
	for _, want := range []string{"**`--tqm` JSON Fields (`TaskQueryModel`):**", "| `owner` | yes | `string` | User uid |", "| `taskState` | no | `integer` | 1 means pending |"} {
		if !strings.Contains(goSkill, want) {
			t.Fatalf("generated Skill missing nested JSON documentation %q:\n%s", want, goSkill)
		}
	}
	if strings.Contains(goSkill, "--data") || strings.Contains(goSkill, "JSON request body") {
		t.Fatalf("generated form Skill should not document JSON --data:\n%s", goSkill)
	}

	rustDir := t.TempDir()
	if err := render.Project(rustDir, "github.com/acme/bpm-cli", app, "rust"); err != nil {
		t.Fatalf("render Rust: %v", err)
	}
	rustCommand := mustReadFile(t, filepath.Join(rustDir, "src", "commands", "bpm.rs"))
	for _, want := range []string{`long = "tqm"`, `long = "first-row"`, `long = "row-count"`, `serde_urlencoded::to_string`, `request_form_text`} {
		if !strings.Contains(rustCommand, want) {
			t.Fatalf("generated Rust command missing %q:\n%s", want, rustCommand)
		}
	}
	if strings.Contains(rustCommand, `request_json_text("GET", &path, query, headers, body)`) {
		t.Fatalf("generated Rust form command should not send the body through the JSON client:\n%s", rustCommand)
	}
}

func TestRenderProjectsKeepPOSTFormFieldsInRequestBody(t *testing.T) {
	app := model.App{
		Name: "form-cli",
		Groups: []model.Group{{
			Name: "forms",
			Operations: []model.Operation{{
				CommandName: "submit",
				Method:      "POST",
				Path:        "/forms",
				BodyMode:    model.BodyModeFormURLEncoded,
				BodyFields:  []model.BodyField{{Name: "name", Type: "string", Required: true}},
			}},
		}},
	}

	goDir := t.TempDir()
	if err := render.Project(goDir, "github.com/acme/form-cli", app); err != nil {
		t.Fatalf("render Go: %v", err)
	}
	goService := mustReadFile(t, filepath.Join(goDir, "internal", "forms", "service.go"))
	for _, want := range []string{`values.Set("name"`, `"application/x-www-form-urlencoded"`} {
		if !strings.Contains(goService, want) {
			t.Fatalf("generated Go POST form service missing %q:\n%s", want, goService)
		}
	}

	rustDir := t.TempDir()
	if err := render.Project(rustDir, "github.com/acme/form-cli", app, "rust"); err != nil {
		t.Fatalf("render Rust: %v", err)
	}
	rustCommand := mustReadFile(t, filepath.Join(rustDir, "src", "commands", "forms.rs"))
	for _, want := range []string{`serde_urlencoded::to_string`, `request_form_text("POST"`} {
		if !strings.Contains(rustCommand, want) {
			t.Fatalf("generated Rust POST form command missing %q:\n%s", want, rustCommand)
		}
	}
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
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

	skillContent, err := os.ReadFile(filepath.Join(dir, "skills", "te-mm-mri-current", "SKILL.md"))
	if err != nil {
		t.Fatalf("read generated skill markdown: %v", err)
	}
	skillText := string(skillContent)
	for _, want := range []string{
		`--current "1"`,
		`--pagesize "25"`,
		"| `--pagesize` | yes | 页码 (query parameter; integer) |",
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

func TestRenderProjectsHandleFlattenedQueryParameterNames(t *testing.T) {
	app := model.App{
		Name: "mycli",
		Groups: []model.Group{{
			Name: "role",
			Operations: []model.Operation{{
				CommandName: "allocated-list",
				Method:      "GET",
				Path:        "/role/authUser/allocatedList",
				Parameters: []model.Parameter{{
					Name: "dept.children[0].createBy",
					In:   "query",
					Type: "string",
				}},
			}},
		}},
	}

	goDir := t.TempDir()
	if err := render.Project(goDir, "github.com/example/tw-cli", app, "go"); err != nil {
		t.Fatalf("render Go project: %v", err)
	}
	goCommand, err := os.ReadFile(filepath.Join(goDir, "internal", "role", "command.go"))
	if err != nil {
		t.Fatalf("read generated Go command: %v", err)
	}
	goService, err := os.ReadFile(filepath.Join(goDir, "internal", "role", "service.go"))
	if err != nil {
		t.Fatalf("read generated Go service: %v", err)
	}
	if !strings.Contains(string(goCommand), "paramDeptChildren0Createby string") {
		t.Fatalf("generated Go command does not sanitize flattened query identifier:\n%s", goCommand)
	}
	if !strings.Contains(string(goService), `query.Set("dept.children[0].createBy", input.DeptChildren0Createby)`) {
		t.Fatalf("generated Go service does not preserve query wire name:\n%s", goService)
	}

	rustDir := t.TempDir()
	if err := render.Project(rustDir, "github.com/example/tw-cli", app, "rust"); err != nil {
		t.Fatalf("render Rust project: %v", err)
	}
	rustCommand, err := os.ReadFile(filepath.Join(rustDir, "src", "commands", "role.rs"))
	if err != nil {
		t.Fatalf("read generated Rust command: %v", err)
	}
	if !strings.Contains(string(rustCommand), "pub dept_children_0_createby: Option<String>") {
		t.Fatalf("generated Rust command does not sanitize flattened query identifier:\n%s", rustCommand)
	}
	if !strings.Contains(string(rustCommand), `("dept.children[0].createBy".to_string(), value.to_string())`) {
		t.Fatalf("generated Rust command does not preserve query wire name:\n%s", rustCommand)
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

func TestRenderMCPHTTPUsesRuntimeConfigurationForGoAndRust(t *testing.T) {
	app := model.App{
		Name: "mcpcli",
		Auth: model.Auth{Type: model.AuthTypeToken},
		Groups: []model.Group{{
			Name:     "search",
			Backend:  model.BackendMCPHTTP,
			Endpoint: "https://example.com/mcp",
			Operations: []model.Operation{{
				CommandName: "query",
				RemoteName:  "search_query",
				Method:      "MCP",
			}},
		}},
	}

	goDir := t.TempDir()
	if err := render.Project(goDir, "github.com/acme/mcpcli", app, "go"); err != nil {
		t.Fatalf("render Go MCP project: %v", err)
	}
	goService, err := os.ReadFile(filepath.Join(goDir, "internal", "search", "service.go"))
	if err != nil {
		t.Fatalf("read Go MCP service: %v", err)
	}
	for _, want := range []string{"config.ResolveBaseURL", "config.ResolveCredential"} {
		if !strings.Contains(string(goService), want) {
			t.Fatalf("generated Go MCP service missing %q:\n%s", want, goService)
		}
	}

	rustDir := t.TempDir()
	if err := render.Project(rustDir, "mcpcli", app, "rust"); err != nil {
		t.Fatalf("render Rust MCP project: %v", err)
	}
	rustClient, err := os.ReadFile(filepath.Join(rustDir, "src", "client.rs"))
	if err != nil {
		t.Fatalf("read Rust MCP client: %v", err)
	}
	for _, want := range []string{"runtime_config::resolve_base_url", "runtime_config::resolve_credential"} {
		if !strings.Contains(string(rustClient), want) {
			t.Fatalf("generated Rust MCP client missing %q:\n%s", want, rustClient)
		}
	}
}
