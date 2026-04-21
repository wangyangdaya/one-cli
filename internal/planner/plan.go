package planner

import (
	"strings"

	"one-cli/internal/configgen"
	"one-cli/internal/model"
	"one-cli/internal/openapi"
)

type Plan = model.App

func Build(doc openapi.Document, cfg configgen.Config) (Plan, error) {
	app := Plan{
		Name: appName(doc, cfg),
	}

	groupIndex := make(map[string]int)
	groupDescriptions := make(map[string]string, len(doc.Tags))
	for _, tag := range doc.Tags {
		groupDescriptions[strings.TrimSpace(tag.Name)] = strings.TrimSpace(tag.Description)
	}
	for _, op := range doc.Operations {
		groupName := groupName(op, cfg)
		commandName := commandName(op, cfg)

		plannedOp := model.Operation{
			Method:       strings.ToUpper(strings.TrimSpace(op.Method)),
			Path:         strings.TrimSpace(op.Path),
			CommandName:  commandName,
			RemoteName:   strings.TrimSpace(op.OperationID),
			Summary:      strings.TrimSpace(op.Summary),
			BodyMode:     bodyMode(op, groupName, commandName, cfg),
			BodyRequired: op.RequestBody.Required,
			BodyFields:   make([]model.BodyField, 0, len(op.RequestBody.JSONFields)),
			Parameters:   make([]model.Parameter, 0, len(op.Parameters)),
		}
		for _, field := range op.RequestBody.JSONFields {
			plannedOp.BodyFields = append(plannedOp.BodyFields, model.BodyField{
				Name:        field.Name,
				Description: field.Description,
				Required:    field.Required,
				Type:        field.Type,
			})
		}
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
			Headers:     cloneStringMap(op.Headers),
			Command:     strings.TrimSpace(op.Command),
			Args:        append([]string(nil), op.Args...),
			Env:         cloneStringMap(op.Env),
			Operations:  []model.Operation{plannedOp},
		})
	}

	return app, nil
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
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

func groupName(op openapi.Operation, cfg configgen.Config) string {
	if alias, ok := cfg.Naming.TagAlias[strings.TrimSpace(op.Tag)]; ok {
		if trimmed := strings.TrimSpace(alias); trimmed != "" {
			return trimmed
		}
	}
	if trimmed := strings.TrimSpace(op.Tag); trimmed != "" {
		return trimmed
	}
	return firstPathSegment(op.Path)
}

func bodyMode(op openapi.Operation, groupName, commandName string, cfg configgen.Config) string {
	if override, ok := bodyModeOverride(op, groupName, commandName, cfg); ok {
		return override
	}
	if len(op.RequestBody.ContentTypes) == 0 {
		return ""
	}
	if op.RequestBody.HasJSONSchema && op.RequestBody.IsSimpleJSON {
		return "simple-json"
	}
	return "file-or-data"
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
