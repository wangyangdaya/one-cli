package render

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"

	"one-cli/internal/model"
)

func Project(outputDir, module string, app model.App) error {
	if strings.TrimSpace(outputDir) == "" {
		return fmt.Errorf("missing output directory")
	}
	if strings.TrimSpace(module) == "" {
		return fmt.Errorf("missing module path")
	}
	if strings.TrimSpace(app.Name) == "" {
		return fmt.Errorf("missing app name")
	}

	files := []generatedFile{
		{Path: filepath.Join("cmd", app.Name, "main.go"), Template: "root_main.go.tmpl", Data: templateData{Module: module, App: app}},
		{Path: "README.md", Template: "readme.md.tmpl", Data: templateData{Module: module, App: app}},
		{Path: filepath.Join("bin", app.Name), Template: "bin_launcher.sh.tmpl", Data: templateData{Module: module, App: app}, Mode: 0o755},
		{Path: filepath.Join("bin", app.Name+".cmd"), Template: "bin_launcher.cmd.tmpl", Data: templateData{Module: module, App: app}, Mode: 0o644},
	}
	for _, group := range app.Groups {
		data := templateData{Module: module, App: app, Group: group}
		files = append(files,
			generatedFile{Path: filepath.Join("internal", group.Name, "command.go"), Template: "group_command.go.tmpl", Data: data},
			generatedFile{Path: filepath.Join("internal", group.Name, "service.go"), Template: "group_service.go.tmpl", Data: data},
			generatedFile{Path: filepath.Join("internal", group.Name, "types.go"), Template: "group_types.go.tmpl", Data: data},
			generatedFile{Path: filepath.Join("skills", group.Name, "SKILL.md"), Template: "skill.md.tmpl", Data: data},
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
	if err := writeRuntime(outputDir); err != nil {
		return err
	}

	return nil
}

func writeGoMod(outputDir, module string) error {
	content, err := os.ReadFile(filepath.Join(filepath.Dir(packageRoot()), "..", "go.mod"))
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
	content, err := os.ReadFile(filepath.Join(filepath.Dir(packageRoot()), "..", "go.sum"))
	if err != nil {
		return err
	}
	return writeFile(filepath.Join(outputDir, "go.sum"), content, 0)
}

func writeTemplates(outputDir string, files []generatedFile) error {
	for _, file := range files {
		content, err := renderTemplate(file.Template, file.Data)
		if err != nil {
			return err
		}
		if err := writeFile(filepath.Join(outputDir, file.Path), content, file.Mode); err != nil {
			return err
		}
	}
	return nil
}

func renderTemplate(name string, data any) ([]byte, error) {
	raw, err := os.ReadFile(templatePath(name))
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New(name).Funcs(template.FuncMap{
		"pascal":            pascal,
		"bodyFlagHelp":      bodyFlagHelp,
		"goType":            goType,
		"groupHasBodyInput": groupHasBodyInput,
	}).Parse(string(raw))
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func pascal(value string) string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '-' || r == '_' || r == ' ' || r == '.'
	})
	for i, part := range parts {
		runes := []rune(part)
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		for j := 1; j < len(runes); j++ {
			runes[j] = unicode.ToLower(runes[j])
		}
		parts[i] = string(runes)
	}
	return strings.Join(parts, "")
}

func bodyFlagHelp(fields []model.BodyField) string {
	if len(fields) == 0 {
		return ""
	}
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		if trimmed := strings.TrimSpace(field.Name); trimmed != "" {
			parts = append(parts, "--"+trimmed)
		}
	}
	return strings.Join(parts, "/")
}

func goType(value string) string {
	switch strings.TrimSpace(value) {
	case "integer":
		return "int"
	case "number":
		return "float64"
	case "boolean":
		return "bool"
	default:
		return "string"
	}
}

func groupHasBodyInput(group model.Group) bool {
	for _, operation := range group.Operations {
		if strings.TrimSpace(operation.BodyMode) != "" {
			return true
		}
	}
	return false
}

func writeRuntime(outputDir string) error {
	root := runtimeRoot()
	paths, err := readRuntimeDir(root)
	if err != nil {
		return err
	}

	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if err := writeFile(filepath.Join(outputDir, "internal", relative), content, 0); err != nil {
			return err
		}
	}

	return nil
}

func writeFile(path string, content []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if mode == 0 {
		mode = 0o644
	}
	return os.WriteFile(path, content, mode)
}
