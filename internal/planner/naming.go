package planner

import (
	"strings"
	"unicode"

	"one-cli/internal/configgen"
	"one-cli/internal/openapi"
)

func commandName(op openapi.Operation, cfg configgen.Config) string {
	if alias, ok := cfg.Naming.OperationAlias[strings.TrimSpace(op.OperationID)]; ok {
		if trimmed := strings.TrimSpace(alias); trimmed != "" {
			return trimmed
		}
	}
	if trimmed := strings.TrimSpace(op.OperationID); trimmed != "" {
		return simplifyOperationID(trimmed)
	}
	return deriveFromMethodPath(op.Method, op.Path)
}

func simplifyOperationID(operationID string) string {
	parts := splitIdentifier(operationID)
	if len(parts) == 0 {
		return "command"
	}
	if len(parts) == 1 {
		return parts[0]
	}
	if parts[0] == "get" && len(parts) > 2 {
		return parts[len(parts)-1]
	}
	if isGenericVerb(parts[0]) && len(parts) > 1 {
		return parts[0]
	}
	return parts[0]
}

func deriveFromMethodPath(method, path string) string {
	segments := []string{strings.ToLower(strings.TrimSpace(method))}
	for _, segment := range strings.Split(strings.TrimSpace(path), "/") {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		segment = strings.Trim(segment, "{}")
		segment = strings.ReplaceAll(segment, "{", "")
		segment = strings.ReplaceAll(segment, "}", "")
		segment = strings.ReplaceAll(segment, "_", "-")
		segments = append(segments, strings.ToLower(segment))
	}
	return strings.Join(filterEmptySegments(segments), "-")
}

func firstPathSegment(path string) string {
	for _, segment := range strings.Split(strings.TrimSpace(path), "/") {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		segment = strings.Trim(segment, "{}")
		segment = strings.ReplaceAll(segment, "_", "-")
		if segment != "" {
			return strings.ToLower(segment)
		}
	}
	return "default"
}

func slugify(text string) string {
	var builder strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(text)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

func splitIdentifier(value string) []string {
	var parts []string
	var current []rune
	for _, r := range value {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if unicode.IsUpper(r) && len(current) > 0 {
				parts = append(parts, strings.ToLower(string(current)))
				current = current[:0]
			}
			current = append(current, unicode.ToLower(r))
		default:
			if len(current) > 0 {
				parts = append(parts, strings.ToLower(string(current)))
				current = current[:0]
			}
		}
	}
	if len(current) > 0 {
		parts = append(parts, strings.ToLower(string(current)))
	}
	return filterEmptySegments(parts)
}

func isGenericVerb(word string) bool {
	switch word {
	case "get", "list", "create", "check", "update", "delete", "patch", "post", "put":
		return true
	default:
		return false
	}
}

func filterEmptySegments(values []string) []string {
	out := values[:0]
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
