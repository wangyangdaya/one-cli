package planner

// bodymode.go keeps the request-body mode heuristics isolated from planning.

import (
	"strings"

	"one-cli/internal/configgen"
	"one-cli/internal/model"
	"one-cli/internal/openapi"
)

// bodyMode resolves the BodyMode string for an operation.
// Override keys are tried in priority order (most-specific first):
//  1. "<groupName>.<commandName>"
//  2. "<tag>.<commandName>"
//  3. "<commandName>"
//  4. "<operationId>"
//  5. "<method> <path>"
//  6. "<path>"
func bodyMode(op openapi.Operation, groupName, commandName string, cfg configgen.Config) string {
	if override, ok := bodyModeOverride(op, groupName, commandName, cfg); ok {
		return override
	}
	if len(op.RequestBody.ContentTypes) == 0 {
		return ""
	}
	if op.RequestBody.HasJSONSchema && op.RequestBody.IsSimpleJSON {
		return model.BodyModeSimpleJSON
	}
	return model.BodyModeFileOrData
}

func bodyModeOverride(op openapi.Operation, groupName, commandName string, cfg configgen.Config) (string, bool) {
	candidates := []string{
		strings.TrimSpace(groupName) + "." + strings.TrimSpace(commandName),
		strings.TrimSpace(op.Tag) + "." + strings.TrimSpace(commandName),
		strings.TrimSpace(commandName),
		strings.TrimSpace(op.OperationID),
		strings.ToLower(strings.TrimSpace(op.Method)) + " " + strings.TrimSpace(op.Path),
		strings.TrimSpace(op.Path),
	}
	for _, key := range candidates {
		if key == "" {
			continue
		}
		if override, ok := cfg.Overrides.BodyMode[key]; ok {
			if trimmed := strings.TrimSpace(override); trimmed != "" {
				return trimmed, true
			}
		}
	}
	return "", false
}
