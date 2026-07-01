package render

import (
	"bytes"
	"encoding/json"
	"fmt"
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
		if err := writeFile(filepath.Join(outputDir, file.Path), content, file.Mode); err != nil {
			return err
		}
	}
	return nil
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
			"pascal":                          pascal,
			"bodyFlagHelp":                    bodyFlagHelp,
			"cargoPackageName":                cargoPackageName,
			"goType":                          goType,
			"groupHasBodyInput":               groupHasBodyInput,
			"groupHasHeaderParams":            groupHasHeaderParams,
			"groupHasBodyFields":              groupHasBodyFields,
			"groupUsesMCPHTTP":                groupUsesMCPHTTP,
			"groupUsesMCPStdio":               groupUsesMCPStdio,
			"appHasMCPHTTP":                   appHasMCPHTTP,
			"appHasMCPStdio":                  appHasMCPStdio,
			"appHasAnyMCP":                    appHasAnyMCP,
			"groupPackageName":                groupPackageName,
			"operationHasHeaderParams":        operationHasHeaderParams,
			"operationHasPathParams":          operationHasPathParams,
			"operationHasQueryParams":         operationHasQueryParams,
			"goAppVersion":                    goAppVersion,
			"rustAppVersion":                  rustAppVersion,
			"rustFieldName":                   rustFieldName,
			"rustModuleName":                  rustModuleName,
			"rustType":                        rustType,
			"stringMapLiteral":                stringMapLiteral,
			"stringSliceLiteral":              stringSliceLiteral,
			"exampleValue":                    exampleValue,
			"exampleJSONFields":               exampleJSONFields,
			"demoRequestJSON":                 demoRequestJSON,
			"demoRequestJSONPretty":           demoRequestJSONPretty,
			"operationIsWriteMethod":          operationIsWriteMethod,
			"operationRiskLevel":              operationRiskLevel,
			"operationRiskLevelChinese":       operationRiskLevelChinese,
			"operationRiskDescriptionChinese": operationRiskDescriptionChinese,
			"operationRequiresListContext":    operationRequiresListContext,
			"groupHasTimeParams":              groupHasTimeParams,
			"groupHasPagination":              groupHasPagination,
			"groupHasListOperation":           groupHasListOperation,
			"groupHasDetailOperation":         groupHasDetailOperation,
			"groupHasSubmitOperation":         groupHasSubmitOperation,
			"generateUserIntentChinese":       generateUserIntentChinese,
			"chineseCommandDescription":       chineseCommandDescription,
			"groupSwaggerIssues":              groupSwaggerIssues,
			"hasOptionalFields":               hasOptionalFields,
			"upper":                           strings.ToUpper,
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
		if strings.Contains(fieldName, "count") || strings.Contains(fieldName, "quantity") || strings.Contains(fieldName, "num") {
			return "10"
		}
		if strings.Contains(fieldName, "id") {
			return "12345"
		}
		if strings.Contains(fieldName, "page") || strings.Contains(fieldName, "no") {
			return "1"
		}
		if strings.Contains(fieldName, "size") {
			return "20"
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

	// String field name-based examples (Chinese-first for business contexts)
	if strings.Contains(fieldName, "email") {
		return "user@example.com"
	}
	if strings.Contains(fieldName, "password") {
		return "<password>"
	}
	if strings.Contains(fieldName, "phone") || strings.Contains(fieldName, "mobile") {
		return "13800138000"
	}
	if strings.Contains(fieldName, "url") || strings.Contains(fieldName, "link") || strings.Contains(fieldName, "appendix") {
		return "https://example.com/file.pdf"
	}
	if strings.Contains(fieldName, "token") {
		return "eyJhbGci..."
	}
	if strings.Contains(fieldName, "name") {
		if strings.Contains(fieldName, "user") || strings.Contains(fieldName, "person") {
			return "张三"
		}
		if strings.Contains(fieldName, "company") || strings.Contains(fieldName, "supplier") || strings.Contains(fieldName, "suppl") {
			return "示例供应商"
		}
		return "示例名称"
	}
	if strings.Contains(fieldName, "date") {
		return "2025-08-01"
	}
	if strings.Contains(fieldName, "time") {
		return "14:30:00"
	}
	if strings.Contains(fieldName, "address") {
		return "安徽省芜湖市"
	}
	if strings.Contains(fieldName, "city") {
		return "芜湖"
	}
	if strings.Contains(fieldName, "code") || strings.Contains(fieldName, "no") {
		return "NO20250001"
	}
	if strings.Contains(fieldName, "status") {
		return "1"
	}
	if strings.Contains(fieldName, "type") {
		return "standard"
	}
	if strings.Contains(fieldName, "description") || strings.Contains(fieldName, "reason") || strings.Contains(fieldName, "remark") {
		return "示例说明"
	}
	if strings.Contains(fieldName, "title") {
		return "示例标题"
	}
	if strings.Contains(fieldName, "note") {
		return "备注信息"
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

func demoRequestJSONPretty(group model.Group) string {
	raw := demoRequestJSON(group)
	var data map[string]any
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return raw
	}
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return raw
	}
	return string(out)
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

// operationRiskLevel returns read/write/bulk/irreversible based on method and command name.
func operationRiskLevel(operation model.Operation) string {
	name := strings.ToLower(strings.TrimSpace(operation.CommandName))
	method := strings.ToUpper(strings.TrimSpace(operation.Method))

	if containsAny(name, []string{"clear", "delete", "remove", "reset", "erase"}) {
		return "irreversible"
	}
	if containsAny(name, []string{"import", "batch", "bulk", "save", "submit"}) {
		return "bulk"
	}
	switch method {
	case "POST", "PUT", "PATCH":
		return "write"
	case "DELETE":
		return "irreversible"
	default:
		return "read"
	}
}

func operationRiskLevelChinese(level string) string {
	switch level {
	case "irreversible":
		return "不可逆（高风险）"
	case "bulk":
		return "批量/写入（中高风险）"
	case "write":
		return "写入（中风险）"
	default:
		return "只读（低风险）"
	}
}

func operationRiskDescriptionChinese(op model.Operation) string {
	switch operationRiskLevel(op) {
	case "irreversible":
		return "执行前必须确认目标，操作后可能无法恢复，建议先查询再执行。"
	case "bulk":
		return "可能影响多条记录，执行前需确认影响范围，建议先通过查询命令核对。"
	case "write":
		return "会修改数据，执行前需确认用户意图和参数正确性。"
	default:
		return "只读查询，不会修改数据。"
	}
}

func operationRequiresListContext(op model.Operation) bool {
	name := strings.ToLower(strings.TrimSpace(op.CommandName))
	return containsAny(name, []string{"detail", "info", "submit", "reject", "withdraw", "update", "delete", "approve", "complete"})
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func groupHasTimeParams(group model.Group) bool {
	for _, op := range group.Operations {
		for _, param := range op.Parameters {
			name := strings.ToLower(param.Name)
			if strings.Contains(name, "date") || strings.Contains(name, "time") {
				return true
			}
		}
		for _, field := range op.BodySchemaFields {
			name := strings.ToLower(field.Name)
			if strings.Contains(name, "date") || strings.Contains(name, "time") {
				return true
			}
		}
	}
	return false
}

func groupHasPagination(group model.Group) bool {
	for _, op := range group.Operations {
		for _, param := range op.Parameters {
			name := strings.ToLower(param.Name)
			if strings.Contains(name, "page") || name == "firstrow" || name == "rowcount" || name == "pagesize" || name == "pageno" {
				return true
			}
		}
		for _, field := range op.BodySchemaFields {
			name := strings.ToLower(field.Name)
			if strings.Contains(name, "page") || name == "firstrow" || name == "rowcount" || name == "pagesize" || name == "pageno" {
				return true
			}
		}
	}
	return false
}

func groupHasListOperation(group model.Group) bool {
	for _, op := range group.Operations {
		name := strings.ToLower(op.CommandName)
		if containsAny(name, []string{"list", "page", "query", "search", "todo", "all"}) {
			return true
		}
	}
	return false
}

func groupHasDetailOperation(group model.Group) bool {
	for _, op := range group.Operations {
		name := strings.ToLower(op.CommandName)
		if containsAny(name, []string{"detail", "info", "get"}) {
			return true
		}
	}
	return false
}

func groupHasSubmitOperation(group model.Group) bool {
	for _, op := range group.Operations {
		name := strings.ToLower(op.CommandName)
		if containsAny(name, []string{"submit", "approve", "reject", "withdraw", "complete", "save", "create", "update"}) {
			return true
		}
	}
	return false
}

func generateUserIntentChinese(op model.Operation) string {
	name := strings.ToLower(strings.TrimSpace(op.CommandName))
	summary := strings.TrimSpace(op.Summary)
	if summary != "" {
		return summary
	}
	if containsAny(name, []string{"list", "page", "query", "search", "todo", "all"}) {
		return "查询列表/待办"
	}
	if containsAny(name, []string{"detail", "info", "get"}) {
		return "查看详情"
	}
	if containsAny(name, []string{"submit", "approve", "complete"}) {
		return "同意/提交"
	}
	if containsAny(name, []string{"reject", "withdraw"}) {
		return "退回/撤回"
	}
	if containsAny(name, []string{"create", "save"}) {
		return "创建/保存"
	}
	if containsAny(name, []string{"update"}) {
		return "更新"
	}
	if containsAny(name, []string{"delete"}) {
		return "删除"
	}
	if containsAny(name, []string{"clear"}) {
		return "清空"
	}
	if containsAny(name, []string{"import"}) {
		return "导入数据"
	}
	return "执行" + op.CommandName
}

func chineseCommandDescription(op model.Operation) string {
	summary := strings.TrimSpace(op.Summary)
	if summary != "" {
		return summary
	}
	risk := operationRiskLevelChinese(operationRiskLevel(op))
	return generateUserIntentChinese(op) + "（" + risk + "）"
}

func groupSwaggerIssues(group model.Group) []string {
	var issues []string
	if strings.TrimSpace(group.Description) == "" {
		issues = append(issues, "`"+group.Name+"` 缺少 group 描述（description）")
	}
	for _, op := range group.Operations {
		if strings.TrimSpace(op.Summary) == "" {
			issues = append(issues, "`"+op.CommandName+"` 缺少命令描述（summary）")
		}
		for _, param := range op.Parameters {
			if strings.TrimSpace(param.Description) == "" {
				issues = append(issues, "`"+op.CommandName+"` 的参数 `"+param.Name+"` 缺少说明")
			}
		}
		for _, field := range op.BodySchemaFields {
			if strings.TrimSpace(field.Description) == "" {
				issues = append(issues, "`"+op.CommandName+"` 的请求体字段 `"+field.Name+"` 缺少说明")
			}
		}
	}
	return issues
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
