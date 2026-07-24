package render

import (
	"strings"

	"one-cli/internal/model"
)

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
		return formFlagName(name)
	}
	return fallback(name)
}
