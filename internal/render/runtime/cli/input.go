package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// ResolveInput resolves a structured CLI input value. Inline values pass
// through, @path reads a file, - reads stdin, and @@ escapes a literal @.
func ResolveInput(raw string, stdin io.Reader) (string, error) {
	if raw == "" {
		return "", nil
	}
	if raw == "-" {
		if stdin == nil {
			return "", fmt.Errorf("stdin is not available")
		}
		data, err := io.ReadAll(stdin)
		if err != nil {
			return "", fmt.Errorf("read stdin: %w", err)
		}
		value := strings.TrimSpace(string(data))
		if value == "" {
			return "", fmt.Errorf("stdin is empty")
		}
		return value, nil
	}
	if strings.HasPrefix(raw, "@@") {
		return raw[1:], nil
	}
	if strings.HasPrefix(raw, "@") {
		path := strings.TrimSpace(raw[1:])
		if path == "" {
			return "", fmt.Errorf("file path cannot be empty after @")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read input file %q: %w", path, err)
		}
		value := strings.TrimSpace(string(data))
		if value == "" {
			return "", fmt.Errorf("input file %q is empty", path)
		}
		return value, nil
	}
	if len(raw) >= 2 && raw[0] == '\'' && raw[len(raw)-1] == '\'' {
		return raw[1 : len(raw)-1], nil
	}
	return raw, nil
}
