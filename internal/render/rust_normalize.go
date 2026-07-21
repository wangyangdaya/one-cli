package render

import (
	"slices"
	"strconv"
	"strings"

	"one-cli/internal/model"
)

func normalizeRustApp(app model.App) model.App {
	app.Groups = slices.Clone(app.Groups)
	groupNames := make(map[string]int, len(app.Groups))
	for groupIndex := range app.Groups {
		group := &app.Groups[groupIndex]
		group.Operations = slices.Clone(group.Operations)
		base := rustModuleName(*group)
		group.PackageName = uniqueRustIdentifier(base, groupNames)

		for operationIndex := range group.Operations {
			operation := &group.Operations[operationIndex]
			operation.Parameters = slices.Clone(operation.Parameters)
			operation.BodyFields = slices.Clone(operation.BodyFields)
			operation.FileFields = slices.Clone(operation.FileFields)
			normalizeRustOperationArgs(operation)
		}
	}
	return app
}

func normalizeRustOperationArgs(operation *model.Operation) {
	fieldNames := make(map[string]int)
	// --file is reserved for binary uploads, even when this operation has no
	// binary field. A JSON property or request parameter named "file" must not
	// silently acquire upload semantics.
	flagNames := map[string]int{"file": 1}
	if operation.BodyMode != "" {
		fieldNames["body_data"] = 1
		flagNames["data"] = 1
	}
	if len(operation.FileFields) > 0 {
		fieldNames["upload_file"] = 1
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
			if baseFlag == "data" || baseFlag == "file" {
				field.FlagName = ""
				continue
			}
			baseFlag = "body-" + baseFlag
		}
		field.FlagName = uniqueRustFlag(baseFlag, flagNames)
	}

	for index := range operation.FileFields {
		field := &operation.FileFields[index]
		field.FieldName = uniqueRustIdentifier("file_"+rustFieldName(field.Name), fieldNames)
		field.FlagName = "file"
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
