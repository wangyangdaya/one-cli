package render

import (
	"strconv"
	"strings"

	"one-cli/internal/model"
)

func normalizeRustApp(app model.App) model.App {
	groupNames := make(map[string]int, len(app.Groups))
	for groupIndex := range app.Groups {
		group := &app.Groups[groupIndex]
		base := rustModuleName(*group)
		group.PackageName = uniqueRustIdentifier(base, groupNames)

		for operationIndex := range group.Operations {
			normalizeRustOperationArgs(&group.Operations[operationIndex])
		}
	}
	return app
}

func normalizeRustOperationArgs(operation *model.Operation) {
	fieldNames := map[string]int{
		"body_data": 1,
		"body_file": 1,
	}
	flagNames := map[string]int{
		"data": 1,
		"file": 1,
	}
	if operationHasHeaderParams(*operation) {
		fieldNames["header"] = 1
		flagNames["header"] = 1
	}

	for index := range operation.Parameters {
		parameter := &operation.Parameters[index]
		if parameter.In != "path" && parameter.In != "query" {
			continue
		}

		baseField := rustFieldName(parameter.Name)
		parameter.FieldName = uniqueRustIdentifier(baseField, fieldNames)

		baseFlag := rustFlagName(parameter.Name)
		if flagNames[baseFlag] > 0 {
			baseFlag = parameter.In + "-" + baseFlag
		}
		parameter.FlagName = uniqueRustFlag(baseFlag, flagNames)
	}

	for index := range operation.BodyFields {
		field := &operation.BodyFields[index]
		baseField := rustFieldName(field.Name)
		if fieldNames[baseField] > 0 {
			baseField = "body_" + baseField
		}
		field.FieldName = uniqueRustIdentifier(baseField, fieldNames)

		baseFlag := rustFlagName(field.Name)
		if flagNames[baseFlag] > 0 {
			baseFlag = "body-" + baseFlag
		}
		field.FlagName = uniqueRustFlag(baseFlag, flagNames)
	}
}

func uniqueRustIdentifier(base string, counts map[string]int) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "value"
	}
	counts[base]++
	if counts[base] == 1 {
		return base
	}
	return base + "_" + strconv.Itoa(counts[base])
}

func uniqueRustFlag(base string, counts map[string]int) string {
	base = strings.Trim(strings.TrimSpace(base), "-")
	if base == "" {
		base = "value"
	}
	counts[base]++
	if counts[base] == 1 {
		return base
	}
	return base + "-" + strconv.Itoa(counts[base])
}
