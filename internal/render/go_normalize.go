package render

import (
	"slices"
	"strconv"
	"strings"
	"unicode"

	"one-cli/internal/model"
)

func normalizeGoApp(app model.App) model.App {
	app.Groups = slices.Clone(app.Groups)
	for groupIndex := range app.Groups {
		group := &app.Groups[groupIndex]
		group.Operations = slices.Clone(group.Operations)
		for operationIndex := range group.Operations {
			operation := &group.Operations[operationIndex]
			operation.Parameters = slices.Clone(operation.Parameters)
			operation.BodyFields = slices.Clone(operation.BodyFields)
			operation.FileFields = slices.Clone(operation.FileFields)
			normalizeGoOperationArgs(operation)
		}
	}
	return app
}

func normalizeGoOperationArgs(operation *model.Operation) {
	fieldNames := make(map[string]int)
	// --file is reserved for binary uploads, even when this operation has no
	// binary field. A JSON property or request parameter named "file" must not
	// silently acquire upload semantics.
	flagNames := map[string]int{"file": 1}
	if operation.BodyMode != "" {
		fieldNames["BodyData"] = 1
		flagNames["data"] = 1
	}
	if operationHasHeaderParams(*operation) {
		fieldNames["Headers"] = 1
		flagNames["header"] = 1
	}
	if len(operation.FileFields) > 0 {
		fieldNames["UploadFile"] = 1
	}

	for index := range operation.Parameters {
		parameter := &operation.Parameters[index]
		if parameter.In != "path" && parameter.In != "query" {
			continue
		}
		baseField := goIdentifier(parameter.Name)
		if fieldNames[baseField] > 0 {
			baseField = goIdentifier(parameter.In) + baseField
		}
		parameter.FieldName = uniqueGoIdentifier(baseField, fieldNames)
		baseFlag := normalizedFlagName(parameter.Name)
		if flagNames[baseFlag] > 0 {
			baseFlag = parameter.In + "-" + baseFlag
		}
		parameter.FlagName = uniqueFlagName(baseFlag, flagNames)
	}

	for index := range operation.BodyFields {
		field := &operation.BodyFields[index]
		baseField := goIdentifier(field.Name)
		if fieldNames[baseField] > 0 {
			baseField = "Body" + baseField
		}
		field.FieldName = uniqueGoIdentifier(baseField, fieldNames)
		baseFlag := normalizedFlagName(field.Name)
		if flagNames[baseFlag] > 0 {
			field.FlagName = ""
			continue
		}
		field.FlagName = uniqueFlagName(baseFlag, flagNames)
	}

	for index := range operation.FileFields {
		field := &operation.FileFields[index]
		field.FieldName = uniqueGoIdentifier("FileField"+goIdentifier(field.Name), fieldNames)
		field.FlagName = "file"
	}
}

func goIdentifier(value string) string {
	result := pascal(value)
	if result == "" {
		result = "Value"
	}
	if first := []rune(result)[0]; unicode.IsDigit(first) {
		result = "Field" + result
	}
	switch result {
	case "Break", "Default", "Func", "Interface", "Select", "Case", "Defer", "Go", "Map", "Struct", "Chan", "Else", "Goto", "Package", "Switch", "Const", "Fallthrough", "If", "Range", "Type", "Continue", "For", "Import", "Return", "Var":
		return result + "Value"
	default:
		return result
	}
}

func normalizedFlagName(value string) string {
	value = strings.TrimSpace(value)
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
			lastDash = false
		default:
			if builder.Len() > 0 && !lastDash {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "value"
	}
	return result
}

func uniqueGoIdentifier(base string, counts map[string]int) string {
	counts[base]++
	if counts[base] == 1 {
		return base
	}
	return base + strconv.Itoa(counts[base])
}

func uniqueFlagName(base string, counts map[string]int) string {
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

func goParamFieldName(parameter model.Parameter) string {
	if parameter.FieldName != "" {
		return parameter.FieldName
	}
	return goIdentifier(parameter.Name)
}

func goParamFlagName(parameter model.Parameter) string {
	if parameter.FlagName != "" {
		return parameter.FlagName
	}
	return normalizedFlagName(parameter.Name)
}

func goBodyFieldName(field model.BodyField) string {
	if field.FieldName != "" {
		return field.FieldName
	}
	return "BodyField" + goIdentifier(field.Name)
}

func goBodyFieldHasFlag(field model.BodyField) bool { return field.FlagName != "" }

func goBodyFlagName(field model.BodyField) string { return field.FlagName }

func goInputFieldName(value any) string {
	switch value := value.(type) {
	case model.Parameter:
		return goParamFieldName(value)
	case model.BodyField:
		return goBodyFieldName(value)
	default:
		return "Value"
	}
}
