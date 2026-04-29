package render

import (
	"path/filepath"
	"strings"

	"one-cli/internal/model"
)

func writeGoProject(outputDir, module string, app model.App) error {
	files := []generatedFile{
		{Path: filepath.Join("cmd", app.Name, "main.go"), Template: "go/root_main.go.tmpl", Data: templateData{Module: module, App: app}},
		{Path: "README.md", Template: "go/readme.md.tmpl", Data: templateData{Module: module, App: app}},
		{Path: filepath.Join("bin", app.Name), Template: "go/bin_launcher.sh.tmpl", Data: templateData{Module: module, App: app}, Mode: 0o755},
		{Path: filepath.Join("bin", app.Name+".cmd"), Template: "go/bin_launcher.cmd.tmpl", Data: templateData{Module: module, App: app}, Mode: 0o644},
	}
	for _, group := range app.Groups {
		data := templateData{Module: module, App: app, Group: group}
		groupDir := groupPackageName(group)
		files = append(files,
			generatedFile{Path: filepath.Join("internal", groupDir, "command.go"), Template: "go/group_command.go.tmpl", Data: data},
			generatedFile{Path: filepath.Join("internal", groupDir, "service.go"), Template: serviceTemplate(group), Data: data},
			generatedFile{Path: filepath.Join("internal", groupDir, "types.go"), Template: "go/group_types.go.tmpl", Data: data},
			generatedFile{Path: filepath.Join("skills", groupDir, "SKILL.md"), Template: "go/skill.md.tmpl", Data: data},
		)
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
	return writeRuntime(outputDir)
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
