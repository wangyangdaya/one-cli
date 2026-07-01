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
		if strings.EqualFold(strings.TrimSpace(op.Method), "MCP") || strings.HasPrefix(strings.TrimSpace(op.Backend), "mcp-") {
			return mcpToolCommandName(trimmed)
		}
		return simplifyOperationID(trimmed)
	}
	return deriveFromMethodPath(op.Method, op.Path)
}

func mcpToolCommandName(operationID string) string {
	parts := splitIdentifier(operationID)
	if len(parts) == 0 {
		return "tool"
	}
	return strings.Join(parts, "-")
}

func simplifyOperationID(operationID string) string {
	parts := splitIdentifier(stripOperationIDNoise(operationID))
	if len(parts) == 0 {
		return "command"
	}
	if len(parts) == 1 {
		return parts[0]
	}
	switch parts[0] {
	case "get":
		return simplifyGetOperation(parts)
	case "do":
		return parts[1]
	case "download":
		return simplifyDownloadOperation(parts)
	case "import":
		return strings.Join(parts[:min(2, len(parts))], "-")
	}
	if isGenericVerb(parts[0]) && len(parts) > 1 {
		return parts[0]
	}
	return parts[0]
}

func stripOperationIDNoise(operationID string) string {
	value := strings.TrimSpace(operationID)
	if idx := strings.LastIndex(value, "_"); idx >= 0 && allDigits(value[idx+1:]) {
		value = value[:idx]
	}
	lower := strings.ToLower(value)
	for _, suffix := range []string{"usingget", "usingpost", "usingput", "usingpatch", "usingdelete"} {
		if strings.HasSuffix(lower, suffix) && len(value) > len(suffix) {
			return value[:len(value)-len(suffix)]
		}
	}
	return value
}

func allDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func simplifyGetOperation(parts []string) string {
	subject := parts[1:]
	if len(subject) == 0 {
		return "get"
	}
	if len(subject) == 1 {
		return "get"
	}
	if contains(subject, "by") {
		return strings.Join(subject, "-")
	}
	last := subject[len(subject)-1]
	if isDetailNoun(last) && len(subject) > 1 {
		return strings.Join(subject[len(subject)-2:], "-")
	}
	return last
}

func simplifyDownloadOperation(parts []string) string {
	subject := parts[1:]
	if len(subject) == 0 {
		return "download"
	}
	if subject[len(subject)-1] != "result" {
		return "download-" + subject[0]
	}
	qualifiers := subject[:len(subject)-1]
	if len(qualifiers) > 1 && qualifiers[len(qualifiers)-1] == "mri" {
		qualifiers = qualifiers[:len(qualifiers)-1]
	}
	if len(qualifiers) == 0 {
		return "result"
	}
	return qualifiers[len(qualifiers)-1] + "-result"
}

func isDetailNoun(word string) bool {
	switch word {
	case "date", "mode", "result", "status", "time", "type":
		return true
	default:
		return false
	}
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
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
	runes := []rune(value)
	for i, r := range runes {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if shouldStartIdentifierPart(runes, current, i) {
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

func shouldStartIdentifierPart(runes []rune, current []rune, index int) bool {
	if len(current) == 0 || !unicode.IsUpper(runes[index]) {
		return false
	}
	previous := runes[index-1]
	if unicode.IsLower(previous) || unicode.IsDigit(previous) {
		return true
	}
	return unicode.IsUpper(previous) && index+1 < len(runes) && unicode.IsLower(runes[index+1])
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
