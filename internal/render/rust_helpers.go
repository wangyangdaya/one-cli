package render

import (
	"strings"
	"unicode"

	"one-cli/internal/model"
)

func rustModuleName(group model.Group) string {
	return rustFieldName(groupPackageName(group))
}

func rustTypeName(value string) string {
	fieldName := rustFieldName(value)
	parts := strings.FieldsFunc(fieldName, func(r rune) bool {
		return r == '_'
	})
	if len(parts) == 0 {
		return "Value"
	}
	for i, part := range parts {
		runes := []rune(part)
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, "")
}

func rustFieldName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "value"
	}

	var builder strings.Builder
	lastUnderscore := false
	for _, r := range value {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(unicode.ToLower(r))
			lastUnderscore = false
		default:
			if !lastUnderscore {
				builder.WriteRune('_')
				lastUnderscore = true
			}
		}
	}

	result := strings.Trim(builder.String(), "_")
	if result == "" {
		result = "value"
	}
	if result[0] >= '0' && result[0] <= '9' {
		result = "field_" + result
	}

	switch result {
	case "type", "match", "loop", "move", "ref", "mod", "crate", "self", "super", "use", "where", "async", "await", "dyn":
		return result + "_value"
	default:
		return result
	}
}

func rustParamFieldName(parameter model.Parameter) string {
	if trimmed := strings.TrimSpace(parameter.FieldName); trimmed != "" {
		return trimmed
	}
	return rustFieldName(parameter.Name)
}

func rustParamFlagName(parameter model.Parameter) string {
	if trimmed := strings.TrimSpace(parameter.FlagName); trimmed != "" {
		return trimmed
	}
	return rustFlagName(parameter.Name)
}

func rustBodyFieldName(field model.BodyField) string {
	if trimmed := strings.TrimSpace(field.FieldName); trimmed != "" {
		return trimmed
	}
	return rustFieldName(field.Name)
}

func rustBodyFlagName(field model.BodyField) string {
	if trimmed := strings.TrimSpace(field.FlagName); trimmed != "" {
		return trimmed
	}
	return rustFlagName(field.Name)
}

func rustFlagName(value string) string {
	return strings.ReplaceAll(rustFieldName(value), "_", "-")
}

func rustType(value string) string {
	switch strings.TrimSpace(value) {
	case "integer":
		return "i64"
	case "number":
		return "f64"
	case "boolean":
		return "bool"
	default:
		return "String"
	}
}

func cargoPackageName(module string) string {
	module = strings.TrimSpace(module)
	if module == "" {
		return "generated-cli"
	}
	if idx := strings.LastIndex(module, "/"); idx >= 0 && idx+1 < len(module) {
		module = module[idx+1:]
	}

	var builder strings.Builder
	lastDash := false
	for _, r := range module {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(unicode.ToLower(r))
			lastDash = false
		default:
			if !lastDash {
				builder.WriteRune('-')
				lastDash = true
			}
		}
	}

	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "generated-cli"
	}
	return result
}
