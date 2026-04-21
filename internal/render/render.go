package render

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
		groupDir := groupPackageName(group)
		files = append(files,
			generatedFile{Path: filepath.Join("internal", groupDir, "command.go"), Template: "group_command.go.tmpl", Data: data},
			generatedFile{Path: filepath.Join("internal", groupDir, "service.go"), Template: "group_service.go.tmpl", Data: data},
			generatedFile{Path: filepath.Join("internal", groupDir, "types.go"), Template: "group_types.go.tmpl", Data: data},
			generatedFile{Path: filepath.Join("skills", groupDir, "SKILL.md"), Template: "skill.md.tmpl", Data: data},
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
		"pascal":                   pascal,
		"bodyFlagHelp":             bodyFlagHelp,
		"goType":                   goType,
		"groupHasBodyInput":        groupHasBodyInput,
		"groupHasHeaderParams":     groupHasHeaderParams,
		"groupUsesMCPHTTP":         groupUsesMCPHTTP,
		"groupUsesMCPStdio":        groupUsesMCPStdio,
		"appHasMCPHTTP":            appHasMCPHTTP,
		"appHasMCPStdio":           appHasMCPStdio,
		"appHasAnyMCP":             appHasAnyMCP,
		"groupPackageName":         groupPackageName,
		"operationHasHeaderParams": operationHasHeaderParams,
		"stringMapLiteral":         stringMapLiteral,
		"stringSliceLiteral":       stringSliceLiteral,
		"exampleValue":             exampleValue,
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

func groupHasHeaderParams(group model.Group) bool {
	for _, operation := range group.Operations {
		if operationHasHeaderParams(operation) {
			return true
		}
	}
	return false
}

func groupUsesMCPHTTP(group model.Group) bool {
	return strings.TrimSpace(group.Backend) == "mcp-streamable-http"
}

func groupUsesMCPStdio(group model.Group) bool {
	return strings.TrimSpace(group.Backend) == "mcp-stdio"
}

func groupPackageName(group model.Group) string {
	if trimmed := strings.TrimSpace(group.PackageName); trimmed != "" {
		return trimmed
	}
	value := strings.TrimSpace(group.Name)
	if value == "" {
		return "default"
	}
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, ".", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return strings.ToLower(value)
}

func appHasMCPHTTP(app model.App) bool {
	for _, group := range app.Groups {
		if groupUsesMCPHTTP(group) {
			return true
		}
	}
	return false
}

func appHasMCPStdio(app model.App) bool {
	for _, group := range app.Groups {
		if groupUsesMCPStdio(group) {
			return true
		}
	}
	return false
}

func appHasAnyMCP(app model.App) bool {
	return appHasMCPHTTP(app) || appHasMCPStdio(app)
}

func operationHasHeaderParams(operation model.Operation) bool {
	for _, parameter := range operation.Parameters {
		if strings.TrimSpace(parameter.In) == "header" {
			return true
		}
	}
	return false
}

func exampleValue(fieldType, fieldName string) string {
	fieldType = strings.TrimSpace(strings.ToLower(fieldType))
	fieldName = strings.TrimSpace(strings.ToLower(fieldName))

	// Type-specific examples
	switch fieldType {
	case "boolean", "bool":
		return "true"
	case "integer", "int":
		if strings.Contains(fieldName, "age") {
			return "25"
		}
		if strings.Contains(fieldName, "count") || strings.Contains(fieldName, "quantity") {
			return "10"
		}
		if strings.Contains(fieldName, "id") {
			return "123"
		}
		return "1"
	case "number", "float", "double":
		if strings.Contains(fieldName, "price") || strings.Contains(fieldName, "amount") {
			return "99.99"
		}
		if strings.Contains(fieldName, "rate") {
			return "0.85"
		}
		return "1.5"
	}

	// String field name-based examples
	if strings.Contains(fieldName, "email") {
		return "user@example.com"
	}
	if strings.Contains(fieldName, "password") {
		return "secret123"
	}
	if strings.Contains(fieldName, "name") {
		return "John Doe"
	}
	if strings.Contains(fieldName, "phone") {
		return "+1234567890"
	}
	if strings.Contains(fieldName, "url") || strings.Contains(fieldName, "link") {
		return "https://example.com"
	}
	if strings.Contains(fieldName, "token") {
		return "eyJhbGci..."
	}
	if strings.Contains(fieldName, "date") {
		return "2026-04-21"
	}
	if strings.Contains(fieldName, "time") {
		return "14:30:00"
	}
	if strings.Contains(fieldName, "address") {
		return "123 Main St"
	}
	if strings.Contains(fieldName, "city") {
		return "New York"
	}
	if strings.Contains(fieldName, "country") {
		return "USA"
	}
	if strings.Contains(fieldName, "code") {
		return "ABC123"
	}
	if strings.Contains(fieldName, "status") {
		return "active"
	}
	if strings.Contains(fieldName, "type") {
		return "standard"
	}
	if strings.Contains(fieldName, "description") {
		return "Sample description"
	}
	if strings.Contains(fieldName, "title") {
		return "Sample Title"
	}

	// Default
	return "value"
}

func stringMapLiteral(values map[string]string) string {
	if len(values) == 0 {
		return "map[string]string(nil)"
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%q: %q", key, values[key]))
	}
	return "map[string]string{" + strings.Join(parts, ", ") + "}"
}

func stringSliceLiteral(values []string) string {
	if len(values) == 0 {
		return "[]string(nil)"
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("%q", value))
	}
	return "[]string{" + strings.Join(parts, ", ") + "}"
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
