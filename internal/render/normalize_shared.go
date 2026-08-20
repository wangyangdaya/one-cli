package render

import (
	"strings"
	"unicode"

	"one-cli/internal/model"
)

func kebabFlagName(value string) string {
	runes := []rune(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for index, current := range runes {
		if !unicode.IsLetter(current) && !unicode.IsDigit(current) {
			if builder.Len() > 0 && !lastDash {
				builder.WriteByte('-')
				lastDash = true
			}
			continue
		}

		if unicode.IsUpper(current) && builder.Len() > 0 && !lastDash {
			previous := runes[index-1]
			nextIsLower := index+1 < len(runes) && unicode.IsLower(runes[index+1])
			if unicode.IsLower(previous) || unicode.IsDigit(previous) || (unicode.IsUpper(previous) && nextIsLower) {
				builder.WriteByte('-')
			}
		}
		builder.WriteRune(unicode.ToLower(current))
		lastDash = false
	}

	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "value"
	}
	return result
}

func operationUsesBodyData(operation *model.Operation) bool {
	return operation.BodyMode != "" && operation.BodyMode != model.BodyModeFormURLEncoded
}

func preferredParameterFlag(parameter model.Parameter, fallback func(string) string) string {
	if preferred := strings.TrimSpace(parameter.PreferredFlagName); preferred != "" {
		return preferred
	}
	return fallback(parameter.Name)
}

func bodyFieldFlag(operation *model.Operation, name string, fallback func(string) string) string {
	if operation.BodyMode == model.BodyModeFormURLEncoded {
		return kebabFlagName(name)
	}
	return fallback(name)
}
