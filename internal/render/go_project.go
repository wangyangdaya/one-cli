package render

import (
	"path/filepath"
	"strings"

	"one-cli/internal/model"
	"one-cli/internal/runtimeconfig"
)

func writeGoProject(outputDir, module string, app model.App, skillLang string, runtimeBundle *runtimeconfig.Bundle) error {
	app = normalizeGoApp(app)
	files := []generatedFile{
		{Path: filepath.Join("cmd", app.Name, "main.go"), Template: "go/root_main.go.tmpl", Data: templateData{Module: module, Target: "go", SkillLang: skillLang, App: app}},
		{Path: "README.md", Template: "go/readme.md.tmpl", Data: templateData{Module: module, Target: "go", SkillLang: skillLang, App: app}},
		{Path: filepath.Join("skills", "README.md"), Template: skillIndexTemplate(skillLang), Data: templateData{Module: module, Target: "go", SkillLang: skillLang, App: app}},
		{Path: filepath.Join("bin", app.Name), Template: "go/bin_launcher.sh.tmpl", Data: templateData{Module: module, Target: "go", SkillLang: skillLang, App: app}, Mode: 0o755},
		{Path: filepath.Join("bin", app.Name+".cmd"), Template: "go/bin_launcher.cmd.tmpl", Data: templateData{Module: module, Target: "go", SkillLang: skillLang, App: app}, Mode: 0o644},
		{Path: filepath.Join("internal", "config", "runtime_config.go"), Template: "go/runtime_config.go.tmpl", Data: templateData{Module: module, Target: "go", SkillLang: skillLang, App: app, RuntimeBundle: runtimeBundle}},
	}
	if len(app.Groups) > 1 {
		files = append(files, generatedFile{Path: filepath.Join("skills", "SKILL.md"), Template: skillRouterTemplate(skillLang), Data: templateData{Module: module, Target: "go", SkillLang: skillLang, App: app}})
	}
	if appUsesAKSK(app) {
		files = append(files, generatedFile{Path: filepath.Join("internal", "auth", "aksk.go"), Template: "go/auth_aksk.go.tmpl", Data: templateData{Module: module, Target: "go", SkillLang: skillLang, App: app}})
	}
	for _, group := range app.Groups {
		data := templateData{Module: module, Target: "go", SkillLang: skillLang, App: app, Group: group}
		groupDir := groupPackageName(group)
		files = append(files,
			generatedFile{Path: filepath.Join("internal", groupDir, "command.go"), Template: "go/group_command.go.tmpl", Data: data},
			generatedFile{Path: filepath.Join("internal", groupDir, "service.go"), Template: serviceTemplate(group), Data: data},
			generatedFile{Path: filepath.Join("internal", groupDir, "types.go"), Template: "go/group_types.go.tmpl", Data: data},
		)
		files = append(files, skillPackageFiles(skillName(group), data)...)
	}

	if err := writeGoMod(outputDir, module); err != nil {
		return err
	}
	if err := writeGoSum(outputDir); err != nil {
		return err
	}
	if err := writeTemplates(outputDir, files); err != nil {
		return err
	}
	if err := writeRuntime(outputDir); err != nil {
		return err
	}
	return writeRuntimeConfig(outputDir, runtimeBundle)
}

func skillRouterTemplate(skillLang string) string {
	if skillLang == "zh" {
		return "skill_router_zh.md.tmpl"
	}
	return "skill_router.md.tmpl"
}

func skillIndexTemplate(skillLang string) string {
	if skillLang == "zh" {
		return "skill_index_zh.md.tmpl"
	}
	return "skill_index.md.tmpl"
}

// serviceTemplate returns the template name for the service file based on the group's backend type.
func serviceTemplate(group model.Group) string {
	switch strings.TrimSpace(group.Backend) {
	case model.BackendMCPHTTP:
		return "go/group_service_mcp_http.go.tmpl"
	case model.BackendMCPStdio:
		return "go/group_service_mcp_stdio.go.tmpl"
	default:
		return "go/group_service_http.go.tmpl"
	}
}

func writeGoMod(outputDir, module string) error {
	content, err := embeddedFS.ReadFile("gomod.tmpl")
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "module ") {
			lines[i] = "module " + module
			break
		}
	}

	return writeFile(filepath.Join(outputDir, "go.mod"), []byte(strings.Join(lines, "\n")), 0)
}

func writeGoSum(outputDir string) error {
	content, err := embeddedFS.ReadFile("gosum.tmpl")
	if err != nil {
		return err
	}
	return writeFile(filepath.Join(outputDir, "go.sum"), content, 0)
}
