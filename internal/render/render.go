package render

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"text/template"
	"unicode"

	"one-cli/internal/model"
)

// templateCache caches parsed templates keyed by template name to avoid
// re-parsing the same template on every renderTemplate call.
var templateCache sync.Map

func writeTemplates(outputDir string, files []generatedFile) error {
	for _, file := range files {
		content, err := renderTemplate(file.Template, file.Data)
		if err != nil {
			return err
		}
		if isGoSourceTemplate(file.Template) {
			formatted, err := format.Source(content)
			if err != nil {
				return fmt.Errorf("format %s: %w", file.Template, err)
			}
			content = formatted
		}
		if err := writeFile(filepath.Join(outputDir, file.Path), content, file.Mode); err != nil {
			return err
		}
	}
	return nil
}

func isGoSourceTemplate(name string) bool {
	return strings.HasPrefix(name, "go/") && strings.HasSuffix(name, ".go.tmpl")
}

func renderTemplate(name string, data any) ([]byte, error) {
	var tmpl *template.Template
	if cached, ok := templateCache.Load(name); ok {
		tmpl = cached.(*template.Template)
	} else {
		raw, err := readTemplate(name)
		if err != nil {
			return nil, err
		}
		parsed, err := template.New(name).Funcs(template.FuncMap{
			"pascal":                   pascal,
			"bodyFlagHelp":             bodyFlagHelp,
			"cargoPackageName":         cargoPackageName,
			"goType":                   goType,
			"groupHasBodyInput":        groupHasBodyInput,
			"groupHasHeaderParams":     groupHasHeaderParams,
			"groupHasBodyFields":       groupHasBodyFields,
			"groupUsesMCPHTTP":         groupUsesMCPHTTP,
			"groupUsesMCPStdio":        groupUsesMCPStdio,
			"appHasMCPHTTP":            appHasMCPHTTP,
			"appHasMCPStdio":           appHasMCPStdio,
			"appHasAnyMCP":             appHasAnyMCP,
			"appUsesAKSK":              appUsesAKSK,
			"appSignerProfile":         appSignerProfile,
			"appSigner":                appSigner,
			"goString":                 goString,
			"rustString":               rustString,
			"groupPackageName":         groupPackageName,
			"operationHasHeaderParams": operationHasHeaderParams,
			"operationHasUserHeaders":  operationHasUserHeaders,
			"operationHasPathParams":   operationHasPathParams,
			"operationHasQueryParams":  operationHasQueryParams,
			"isHiddenAuthHeader":       isHiddenAuthHeader,
			"goAppVersion":             goAppVersion,
			"rustAppVersion":           rustAppVersion,
			"cliFlagName":              cliFlagName,
			"cliParamFlagName":         cliParamFlagName,
			"cliBodyFlagName":          cliBodyFlagName,
			"rustFieldName":            rustFieldName,
			"rustTypeName":             rustTypeName,
			"rustParamFieldName":       rustParamFieldName,
			"rustParamFlagName":        rustParamFlagName,
			"rustBodyFieldName":        rustBodyFieldName,
			"rustBodyFlagName":         rustBodyFlagName,
			"rustBodyFieldsForSigner":  rustBodyFieldsForSigner,
			"rustModuleName":           rustModuleName,
			"rustType":                 rustType,
			"stringMapLiteral":         stringMapLiteral,
			"stringSliceLiteral":       stringSliceLiteral,
			"exampleValue":             exampleValue,
			"exampleJSONFields":        exampleJSONFields,
			"bodyRequiredLabel":        bodyRequiredLabel,
			"demoRequestJSON":          demoRequestJSON,
			"operationIsWriteMethod":   operationIsWriteMethod,
			"operationRiskLabel":       operationRiskLabel,
			"groupDocumentationIssues": groupDocumentationIssues,
			"appDocumentationIssues":   appDocumentationIssues,
			"hasOptionalFields":        hasOptionalFields,
			"upper":                    strings.ToUpper,
		}).Parse(string(raw))
		if err != nil {
			return nil, err
		}
		templateCache.Store(name, parsed)
		tmpl = parsed
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func bodyRequiredLabel(field model.BodyField, lang string) string {
	if field.RequiredUnknown {
		if lang == "zh" {
			return "未声明"
		}
		return "unspecified"
	}
	if field.Required {
		if lang == "zh" {
			return "是"
		}
		return "yes"
	}
	if lang == "zh" {
		return "否"
	}
	return "no"
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

func cliFlagName(target, name string) string {
	if strings.EqualFold(strings.TrimSpace(target), "rust") {
		return rustFieldName(name)
	}
	return strings.TrimSpace(name)
}

func cliParamFlagName(target string, parameter model.Parameter) string {
	if strings.EqualFold(strings.TrimSpace(target), "rust") {
		return rustParamFlagName(parameter)
	}
	return strings.TrimSpace(parameter.Name)
}

func cliBodyFlagName(target string, field model.BodyField) string {
	if strings.EqualFold(strings.TrimSpace(target), "rust") {
		return rustBodyFlagName(field)
	}
	return strings.TrimSpace(field.Name)
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

func groupHasBodyFields(group model.Group) bool {
	for _, operation := range group.Operations {
		if len(operation.BodyFields) > 0 {
			return true
		}
	}
	return false
}

func groupUsesMCPHTTP(group model.Group) bool {
	return strings.TrimSpace(group.Backend) == model.BackendMCPHTTP
}

func groupUsesMCPStdio(group model.Group) bool {
	return strings.TrimSpace(group.Backend) == model.BackendMCPStdio
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

func appUsesAKSK(app model.App) bool {
	return strings.TrimSpace(app.Auth.Type) == model.AuthTypeAKSK
}

func appSignerProfile(app model.App) string {
	if profile := strings.TrimSpace(app.Auth.Signer.Profile); profile != "" {
		return profile
	}
	if profile := strings.TrimSpace(app.Auth.SignerProfile); profile != "" {
		return profile
	}
	if appUsesAKSK(app) {
		return model.SignerProfileSupplierEDI
	}
	return ""
}

func appSigner(app model.App) model.Signer {
	return app.Auth.Signer
}

func goString(value string) string {
	return fmt.Sprintf("%q", value)
}

func rustString(value string) string {
	return fmt.Sprintf("%q", value)
}

func rustBodyFieldsForSigner(app model.App, fields []model.BodyField) []model.BodyField {
	ordered := append([]model.BodyField(nil), fields...)
	if strings.EqualFold(strings.TrimSpace(app.Auth.Signer.BodyOrder), "alpha") {
		sort.SliceStable(ordered, func(i, j int) bool {
			return strings.ToLower(strings.TrimSpace(ordered[i].Name)) < strings.ToLower(strings.TrimSpace(ordered[j].Name))
		})
	}
	return ordered
}

func goAppVersion(app model.App) string {
	if version := strings.TrimSpace(app.Version); version != "" {
		return version
	}
	return "dev"
}

func rustAppVersion(app model.App) string {
	if version := strings.TrimSpace(app.Version); version != "" {
		return version
	}
	return "0.1.0"
}

func operationHasParamsIn(operation model.Operation, location string) bool {
	for _, parameter := range operation.Parameters {
		if strings.TrimSpace(parameter.In) == location {
			return true
		}
	}
	return false
}

func operationHasHeaderParams(operation model.Operation) bool {
	return operationHasParamsIn(operation, "header")
}

func operationHasUserHeaders(app model.App, operation model.Operation) bool {
	for _, parameter := range operation.Parameters {
		if strings.TrimSpace(parameter.In) == "header" && !isHiddenAuthHeader(app, parameter) {
			return true
		}
	}
	return false
}

func isHiddenAuthHeader(app model.App, parameter model.Parameter) bool {
	if !appUsesAKSK(app) || strings.TrimSpace(parameter.In) != "header" {
		return false
	}
	name := strings.ToLower(strings.TrimSpace(parameter.Name))
	signer := app.Auth.Signer
	for _, candidate := range []string{
		signer.AccessKeyHeader,
		signer.SignatureHeader,
		signer.TimestampHeader,
		signer.NonceHeader,
	} {
		if name == strings.ToLower(strings.TrimSpace(candidate)) {
			return true
		}
	}
	return false
}

func operationHasPathParams(operation model.Operation) bool {
	return operationHasParamsIn(operation, "path")
}

func operationHasQueryParams(operation model.Operation) bool {
	return operationHasParamsIn(operation, "query")
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
		return "<password>"
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

func exampleJSONFields(fields []model.BodyField) string {
	if len(fields) == 0 {
		return `{"field": "value"}`
	}
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		name := strings.TrimSpace(field.Name)
		if name == "" {
			continue
		}
		value := exampleValue(field.Type, field.Name)
		switch strings.TrimSpace(strings.ToLower(field.Type)) {
		case "integer", "number", "boolean":
			parts = append(parts, fmt.Sprintf("%q: %s", name, value))
		default:
			parts = append(parts, fmt.Sprintf("%q: %q", name, value))
		}
	}
	if len(parts) == 0 {
		return `{"field": "value"}`
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func demoRequestJSON(group model.Group) string {
	for _, operation := range group.Operations {
		if len(operation.BodySchemaFields) > 0 {
			return exampleJSONFields(operation.BodySchemaFields)
		}
		if len(operation.BodyFields) > 0 {
			return exampleJSONFields(operation.BodyFields)
		}
	}
	return `{"demo": true}`
}

func operationIsWriteMethod(operation model.Operation) bool {
	method := strings.ToUpper(strings.TrimSpace(operation.Method))
	switch method {
	case "POST", "PUT", "PATCH", "DELETE":
		return true
	default:
		return false
	}
}

func operationRiskLevel(operation model.Operation) string {
	switch strings.ToUpper(strings.TrimSpace(operation.Method)) {
	case "GET", "HEAD", "OPTIONS":
		return "read"
	case "DELETE":
		return "destructive"
	case "POST", "PUT", "PATCH":
		return "write"
	default:
		return "unknown"
	}
}

func operationRiskLabel(operation model.Operation, lang string) string {
	zh := strings.EqualFold(strings.TrimSpace(lang), "zh")
	switch operationRiskLevel(operation) {
	case "read":
		if zh {
			return "只读"
		}
		return "read"
	case "destructive":
		if zh {
			return "高风险"
		}
		return "destructive"
	case "write":
		if zh {
			return "写入"
		}
		return "write"
	default:
		if zh {
			return "未知"
		}
		return "unknown"
	}
}

func groupDocumentationIssues(group model.Group, lang string) []string {
	zh := strings.EqualFold(strings.TrimSpace(lang), "zh")
	issue := func(en, cn string) string {
		if zh {
			return cn
		}
		return en
	}

	var issues []string
	if strings.TrimSpace(group.Description) == "" {
		issues = append(issues, issue(
			"`"+group.Name+"` is missing a group description",
			"`"+group.Name+"` 缺少分组描述",
		))
	}
	for _, op := range group.Operations {
		if strings.TrimSpace(op.Summary) == "" {
			issues = append(issues, issue(
				"`"+op.CommandName+"` is missing a command summary",
				"`"+op.CommandName+"` 缺少命令摘要",
			))
		}
		issues = append(issues, operationParameterIssues(op, issue)...)
		issues = append(issues, operationBodyFieldIssues(op, issue)...)
		issues = append(issues, operationPathIssues(op, issue)...)
	}
	return issues
}

func appDocumentationIssues(app model.App, lang string) []string {
	var issues []string
	for _, group := range app.Groups {
		issues = append(issues, groupDocumentationIssues(group, lang)...)
	}
	return issues
}

func operationParameterIssues(op model.Operation, issue func(string, string) string) []string {
	var issues []string
	seen := make(map[string]bool, len(op.Parameters))
	for _, param := range op.Parameters {
		name := strings.TrimSpace(param.Name)
		location := strings.TrimSpace(param.In)
		if name == "" {
			continue
		}
		if location == "" {
			issues = append(issues, issue(
				"`"+op.CommandName+"` parameter `"+name+"` is missing a location",
				"`"+op.CommandName+"` 的参数 `"+name+"` 缺少位置",
			))
		} else if !supportedParameterLocation(location) {
			issues = append(issues, issue(
				"`"+op.CommandName+"` parameter `"+name+"` uses unsupported location `"+location+"`",
				"`"+op.CommandName+"` 的参数 `"+name+"` 使用了不支持的位置 `"+location+"`",
			))
		}
		key := location + "\x00" + name
		if seen[key] {
			issues = append(issues, issue(
				"`"+op.CommandName+"` declares duplicate `"+location+"` parameter `"+name+"`",
				"`"+op.CommandName+"` 重复声明了 `"+location+"` 参数 `"+name+"`",
			))
		}
		seen[key] = true
		if strings.TrimSpace(param.Description) == "" {
			issues = append(issues, issue(
				"`"+op.CommandName+"` parameter `"+name+"` is missing a description",
				"`"+op.CommandName+"` 的参数 `"+name+"` 缺少说明",
			))
		}
		if strings.TrimSpace(param.Type) == "" {
			issues = append(issues, issue(
				"`"+op.CommandName+"` parameter `"+name+"` is missing a type",
				"`"+op.CommandName+"` 的参数 `"+name+"` 缺少类型",
			))
		}
	}
	return issues
}

func operationBodyFieldIssues(op model.Operation, issue func(string, string) string) []string {
	var issues []string
	seen := make(map[string]bool, len(op.BodyFields)+len(op.BodySchemaFields))
	for _, field := range append(append([]model.BodyField(nil), op.BodySchemaFields...), op.BodyFields...) {
		name := strings.TrimSpace(field.Name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		if strings.TrimSpace(field.Description) == "" {
			issues = append(issues, issue(
				"`"+op.CommandName+"` request body field `"+name+"` is missing a description",
				"`"+op.CommandName+"` 的请求体字段 `"+name+"` 缺少说明",
			))
		}
		if strings.TrimSpace(field.Type) == "" {
			issues = append(issues, issue(
				"`"+op.CommandName+"` request body field `"+name+"` is missing a type",
				"`"+op.CommandName+"` 的请求体字段 `"+name+"` 缺少类型",
			))
		}
		if field.RequiredUnknown {
			issues = append(issues, issue(
				"`"+op.CommandName+"` request body field `"+name+"` does not declare whether it is required",
				"`"+op.CommandName+"` 的请求体字段 `"+name+"` 未声明是否必填",
			))
		}
	}
	return issues
}

func operationPathIssues(op model.Operation, issue func(string, string) string) []string {
	var issues []string
	path := strings.TrimSpace(op.Path)
	templateParams := pathTemplateParams(path)
	declaredPathParams := make(map[string]model.Parameter, len(op.Parameters))
	for _, param := range op.Parameters {
		if strings.TrimSpace(param.In) != "path" {
			continue
		}
		name := strings.TrimSpace(param.Name)
		if name == "" {
			continue
		}
		declaredPathParams[name] = param
		if !templateParams[name] {
			issues = append(issues, issue(
				"`"+op.CommandName+"` declares path parameter `"+name+"` that is not present in path `"+path+"`",
				"`"+op.CommandName+"` 声明了路径参数 `"+name+"`，但路径 `"+path+"` 中不存在该占位符",
			))
		}
		if !param.Required {
			issues = append(issues, issue(
				"`"+op.CommandName+"` path parameter `"+name+"` should be required",
				"`"+op.CommandName+"` 的路径参数 `"+name+"` 应声明为必填",
			))
		}
	}
	for _, name := range sortedIssueNames(templateParams) {
		if _, ok := declaredPathParams[name]; !ok {
			issues = append(issues, issue(
				"`"+op.CommandName+"` path parameter `{"+name+"}` is missing a matching `in: path` parameter",
				"`"+op.CommandName+"` 的路径参数 `{"+name+"}` 缺少匹配的 `in: path` 参数声明",
			))
		}
	}
	return issues
}

func supportedParameterLocation(location string) bool {
	switch strings.TrimSpace(location) {
	case "path", "query", "header":
		return true
	default:
		return false
	}
}

func pathTemplateParams(path string) map[string]bool {
	params := make(map[string]bool)
	for {
		start := strings.Index(path, "{")
		if start < 0 {
			return params
		}
		path = path[start+1:]
		end := strings.Index(path, "}")
		if end < 0 {
			return params
		}
		if name := strings.TrimSpace(path[:end]); name != "" {
			params[name] = true
		}
		path = path[end+1:]
	}
}

func sortedIssueNames(values map[string]bool) []string {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func hasOptionalFields(operation model.Operation) bool {
	for _, param := range operation.Parameters {
		if !param.Required && strings.TrimSpace(param.In) != "header" {
			return true
		}
	}
	for _, field := range operation.BodyFields {
		if !field.Required {
			return true
		}
	}
	return false
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
	paths, err := listEmbedDir(embeddedFS, "runtime")
	if err != nil {
		return err
	}

	for _, path := range paths {
		content, err := embeddedFS.ReadFile(path)
		if err != nil {
			return err
		}

		// Strip the "runtime/" prefix to get the relative path
		relative := path[len("runtime/"):]
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
