package planner

import (
	"strconv"
	"strings"

	"one-cli/internal/configgen"
	"one-cli/internal/model"
	"one-cli/internal/openapi"
)

type Plan = model.App

func Build(doc openapi.Document, cfg configgen.Config) Plan {
	app := Plan{
		Name:    appName(doc, cfg),
		Version: strings.TrimSpace(cfg.App.Version),
	}

	groupIndex := make(map[string]int)
	groupCommandCounts := make(map[string]map[string]int)
	groupDescriptions := make(map[string]string, len(doc.Tags))
	for _, tag := range doc.Tags {
		groupDescriptions[strings.TrimSpace(tag.Name)] = strings.TrimSpace(tag.Description)
	}
	for _, op := range doc.Operations {
		groupName := groupName(op, cfg, groupDescriptions)
		commandName := uniqueCommandName(commandName(op, cfg), groupName, groupCommandCounts)

		plannedOp := model.Operation{
			Method:           strings.ToUpper(strings.TrimSpace(op.Method)),
			Path:             strings.TrimSpace(op.Path),
			CommandName:      commandName,
			RemoteName:       strings.TrimSpace(op.OperationID),
			Summary:          strings.TrimSpace(op.Summary),
			BodyMode:         bodyMode(op, groupName, commandName, cfg),
			BodyRequired:     op.RequestBody.Required,
			BodyFields:       make([]model.BodyField, 0, len(op.RequestBody.JSONFields)),
			BodySchemaFields: make([]model.BodyField, 0, len(op.RequestBody.JSONSchemaFields)),
			Parameters:       make([]model.Parameter, 0, len(op.Parameters)),
		}
		for _, field := range op.RequestBody.JSONFields {
			plannedOp.BodyFields = append(plannedOp.BodyFields, model.BodyField{
				Name:            field.Name,
				Description:     field.Description,
				Required:        field.Required,
				RequiredUnknown: field.RequiredUnknown,
				Type:            field.Type,
			})
		}
		for _, field := range op.RequestBody.JSONSchemaFields {
			plannedOp.BodySchemaFields = append(plannedOp.BodySchemaFields, model.BodyField{
				Name:            field.Name,
				Description:     field.Description,
				Required:        field.Required,
				RequiredUnknown: field.RequiredUnknown,
				Type:            field.Type,
			})
		}
		applyBodyFieldOverrides(op, groupName, commandName, cfg, &plannedOp)
		for _, parameter := range op.Parameters {
			plannedOp.Parameters = append(plannedOp.Parameters, model.Parameter{
				Name:        parameter.Name,
				In:          parameter.In,
				Required:    parameter.Required,
				Description: parameter.Description,
				Type:        parameter.Type,
			})
		}

		if idx, ok := groupIndex[groupName]; ok {
			app.Groups[idx].Operations = append(app.Groups[idx].Operations, plannedOp)
			continue
		}

		groupIndex[groupName] = len(app.Groups)
		app.Groups = append(app.Groups, model.Group{
			Name:        groupName,
			PackageName: packageName(groupName),
			Description: groupDescription(op, groupName, groupDescriptions),
			Backend:     strings.TrimSpace(op.Backend),
			Endpoint:    strings.TrimSpace(op.Endpoint),
			Headers:     model.CloneStringMap(op.Headers),
			Command:     strings.TrimSpace(op.Command),
			Args:        append([]string(nil), op.Args...),
			Env:         model.CloneStringMap(op.Env),
			Operations:  []model.Operation{plannedOp},
		})
	}

	return app
}

func applyBodyFieldOverrides(op openapi.Operation, groupName, commandName string, cfg configgen.Config, plannedOp *model.Operation) {
	for _, key := range overrideCandidates(op, groupName, commandName) {
		fields := cfg.Overrides.BodyFields[key]
		if len(fields) == 0 {
			continue
		}
		plannedOp.BodyFields = applyFieldOverrides(plannedOp.BodyFields, fields)
		plannedOp.BodySchemaFields = applyFieldOverrides(plannedOp.BodySchemaFields, fields)
		return
	}
}

func applyFieldOverrides(fields []model.BodyField, overrides []configgen.BodyField) []model.BodyField {
	index := make(map[string]int, len(fields))
	for i, field := range fields {
		index[strings.TrimSpace(field.Name)] = i
	}
	for _, override := range overrides {
		name := strings.TrimSpace(override.Name)
		if name == "" {
			continue
		}
		field := model.BodyField{Name: name}
		if i, ok := index[name]; ok {
			field = fields[i]
		} else {
			index[name] = len(fields)
			fields = append(fields, field)
		}
		if override.Description != "" {
			field.Description = strings.TrimSpace(override.Description)
		}
		if override.Required != nil {
			field.Required = *override.Required
			field.RequiredUnknown = false
		}
		if override.Type != "" {
			field.Type = strings.TrimSpace(override.Type)
		}
		fields[index[name]] = field
	}
	return fields
}

func uniqueCommandName(value, groupName string, counts map[string]map[string]int) string {
	base := commandIdentifier(value)
	groupName = strings.TrimSpace(groupName)
	if groupName == "" {
		groupName = "default"
	}
	if counts[groupName] == nil {
		counts[groupName] = make(map[string]int)
	}
	counts[groupName][base]++
	if counts[groupName][base] == 1 {
		return base
	}
	return base + "-" + strconv.Itoa(counts[groupName][base])
}

func commandIdentifier(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "command"
	}
	if value[0] >= '0' && value[0] <= '9' {
		return "command-" + value
	}
	return value
}

func packageName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "default"
	}

	var builder strings.Builder
	lastUnderscore := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			builder.WriteRune(r)
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
		return "default"
	}
	if result[0] >= '0' && result[0] <= '9' {
		return "group_" + result
	}
	return strings.ToLower(result)
}

func groupDescription(op openapi.Operation, groupName string, descriptions map[string]string) string {
	if desc := strings.TrimSpace(descriptions[strings.TrimSpace(op.Tag)]); desc != "" {
		return desc
	}
	if desc := strings.TrimSpace(descriptions[strings.TrimSpace(groupName)]); desc != "" {
		return desc
	}
	return ""
}

func appName(doc openapi.Document, cfg configgen.Config) string {
	if name := strings.TrimSpace(cfg.App.RootCommand); name != "" {
		return name
	}
	if name := strings.TrimSpace(cfg.App.Binary); name != "" {
		return name
	}
	if title := strings.TrimSpace(doc.Title); title != "" {
		return slugify(title)
	}
	return "app"
}

func groupName(op openapi.Operation, cfg configgen.Config, descriptions map[string]string) string {
	if alias, ok := cfg.Naming.TagAlias[strings.TrimSpace(op.Tag)]; ok {
		if trimmed := strings.TrimSpace(alias); trimmed != "" {
			return trimmed
		}
	}
	if trimmed := strings.TrimSpace(op.Tag); trimmed != "" {
		if controllerName, ok := controllerGroupName(trimmed); ok {
			return controllerName
		}
		if !isCLIIdentifier(trimmed) {
			if descName, ok := controllerGroupName(descriptions[trimmed]); ok {
				return descName
			}
			if pathName := pathResourceGroupName(op.Path); pathName != "" {
				return pathName
			}
		}
		return trimmed
	}
	return firstPathSegment(op.Path)
}

func isCLIIdentifier(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			continue
		case r == '-' || r == '_':
			continue
		default:
			return false
		}
	}
	return true
}

func pathResourceGroupName(path string) string {
	var candidates [][]string
	for _, segment := range strings.Split(strings.TrimSpace(path), "/") {
		segment = strings.TrimSpace(segment)
		if segment == "" || strings.EqualFold(segment, "api") || strings.EqualFold(segment, "les") {
			continue
		}
		parts := splitIdentifier(strings.Trim(segment, "{}"))
		if len(parts) > 0 {
			candidates = append(candidates, parts)
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	if len(candidates[0]) == 1 && len(candidates[0][0]) <= 3 && len(candidates) > 1 {
		return strings.Join(append(append([]string{}, candidates[0]...), candidates[1]...), "-")
	}
	return strings.Join(candidates[0], "-")
}

func controllerGroupName(tag string) (string, bool) {
	fields := strings.FieldsFunc(tag, func(r rune) bool {
		switch r {
		case '-', '/', '\\', ':', '：':
			return true
		default:
			return false
		}
	})
	for i := len(fields) - 1; i >= 0; i-- {
		field := strings.TrimSpace(fields[i])
		base, ok := trimControllerSuffix(field)
		if !ok {
			continue
		}
		parts := splitIdentifier(base)
		if len(parts) == 0 {
			continue
		}
		return strings.Join(parts, "-"), true
	}
	return "", false
}

func trimControllerSuffix(value string) (string, bool) {
	const suffix = "controller"
	value = strings.TrimSpace(value)
	if len(value) <= len(suffix) {
		return "", false
	}
	lower := strings.ToLower(value)
	if !strings.HasSuffix(lower, suffix) {
		return "", false
	}
	return value[:len(value)-len(suffix)], true
}
