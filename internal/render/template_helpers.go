package render

import (
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"
	"unicode"

	"one-cli/internal/model"
)

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
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
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
		if trimmed := strings.TrimSpace(field.FlagName); trimmed != "" {
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
	return goParamFlagName(parameter)
}

func cliBodyFlagName(target string, field model.BodyField) string {
	if strings.EqualFold(strings.TrimSpace(target), "rust") {
		return rustBodyFlagName(field)
	}
	return goBodyFlagName(field)
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

func groupHasDataBody(group model.Group) bool {
	for _, operation := range group.Operations {
		if strings.TrimSpace(operation.BodyMode) != "" && operation.BodyMode != model.BodyModeFormURLEncoded {
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

func skillName(group model.Group) string {
	name := normalizeSkillName(group.Name)
	if name == "" {
		name = normalizeSkillName(group.PackageName)
	}
	if name == "" {
		name = "skill"
	}
	if len(name) > 64 {
		name = strings.TrimRight(name[:64], "-")
	}
	return name
}

func normalizeSkillName(value string) string {
	var builder strings.Builder
	lastHyphen := false
	for _, r := range strings.TrimSpace(value) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastHyphen = false
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r + ('a' - 'A'))
			lastHyphen = false
		default:
			if builder.Len() > 0 && !lastHyphen {
				builder.WriteByte('-')
				lastHyphen = true
			}
		}
	}
	return strings.Trim(builder.String(), "-")
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

func appUsesToken(app model.App) bool {
	authType := strings.TrimSpace(app.Auth.Type)
	return authType == "" || authType == model.AuthTypeToken
}

func appUsesAPIKey(app model.App) bool {
	return strings.TrimSpace(app.Auth.Type) == model.AuthTypeAPIKey
}

func appUsesOAuth2(app model.App) bool {
	return strings.TrimSpace(app.Auth.Type) == model.AuthTypeOAuth2
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

func groupHasUserHeaders(app model.App, group model.Group) bool {
	for _, operation := range group.Operations {
		if operationHasUserHeaders(app, operation) {
			return true
		}
	}
	return false
}

func isHiddenAuthHeader(app model.App, parameter model.Parameter) bool {
	if strings.TrimSpace(parameter.In) != "header" {
		return false
	}
	name := strings.ToLower(strings.TrimSpace(parameter.Name))
	if appUsesToken(app) && name == "authorization" {
		return true
	}
	if !appUsesAKSK(app) {
		return false
	}
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

func groupHasFileFields(group model.Group) bool {
	for _, operation := range group.Operations {
		if len(operation.FileFields) > 0 {
			return true
		}
	}
	return false
}

func defaultFileField(operation model.Operation) string {
	if len(operation.FileFields) == 1 {
		return strings.TrimSpace(operation.FileFields[0].Name)
	}
	return ""
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

func bodyFieldExample(field model.BodyField) string {
	value := strings.TrimSpace(field.Example)
	if value == "" {
		value = exampleValue(field.Type, field.Name)
	}
	if strings.HasPrefix(value, "{") || strings.HasPrefix(value, "[") {
		return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
	}
	return fmt.Sprintf("%q", value)
}

func parameterExample(parameter model.Parameter) string {
	return bodyFieldExample(model.BodyField{
		Name:    parameter.Name,
		Type:    parameter.Type,
		Example: parameter.Example,
	})
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
		if len(field.JSONFields) > 0 {
			parts = append(parts, fmt.Sprintf("%q: %s", name, exampleJSONFields(field.JSONFields)))
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

func flattenBodyFields(fields []model.BodyField) []model.BodyField {
	var flattened []model.BodyField
	var walk func(string, []model.BodyField)
	walk = func(prefix string, nested []model.BodyField) {
		for _, field := range nested {
			copy := field
			if prefix != "" {
				copy.Name = prefix + "." + field.Name
			}
			flattened = append(flattened, copy)
			if len(field.JSONFields) > 0 {
				walk(copy.Name, field.JSONFields)
			}
		}
	}
	walk("", fields)
	return flattened
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
	keys := slices.Sorted(maps.Keys(values))
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
