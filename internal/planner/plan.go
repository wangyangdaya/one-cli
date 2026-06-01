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
			Headers:     model.CloneStringMap(op.Headers),
			Command:     strings.TrimSpace(op.Command),
			Args:        append([]string(nil), op.Args...),
			Env:         model.CloneStringMap(op.Env),
			Operations:  []model.Operation{plannedOp},
		})
	}

	// Post-process: strip redundant group name prefix from command names
	// and detect/resolve naming conflicts within each group.
	for i := range app.Groups {
		deduplicateCommandNames(&app.Groups[i])
	}

	return app, nil
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

// deduplicateCommandNames strips redundant group name prefixes from command
// names and resolves conflicts by keeping the full hyphenated name.
func deduplicateCommandNames(group *model.Group) {
	groupParts := splitIdentifier(group.Name)
	if len(groupParts) == 0 {
		return
	}
	groupPrefix := strings.ToLower(groupParts[0])

	// First pass: try to strip group prefix from each command name
	stripped := make([]string, len(group.Operations))
	for i, op := range group.Operations {
		parts := strings.Split(op.CommandName, "-")
		if len(parts) > 1 && parts[0] == groupPrefix {
			stripped[i] = strings.Join(parts[1:], "-")
		} else {
			stripped[i] = op.CommandName
		}
	}

	// Second pass: detect conflicts and revert to full name where needed
	counts := make(map[string]int)
	for _, name := range stripped {
		counts[name]++
	}
	for i, name := range stripped {
		if counts[name] > 1 {
			// Conflict: keep original full name
			stripped[i] = group.Operations[i].CommandName
		}
	}

	// Third pass: detect conflicts among final names (original names that
	// were kept might conflict with stripped names)
	finalCounts := make(map[string][]int)
	for i, name := range stripped {
		finalCounts[name] = append(finalCounts[name], i)
	}
	for _, indices := range finalCounts {
		if len(indices) <= 1 {
			continue
		}
		// Expand all conflicting entries to full operationID-based name
		for _, idx := range indices {
			opID := group.Operations[idx].RemoteName
			if opID != "" {
				parts := splitIdentifier(opID)
				if len(parts) > 0 {
					stripped[idx] = strings.Join(parts, "-")
				}
			}
		}
	}

	// Apply
	for i := range group.Operations {
		group.Operations[i].CommandName = stripped[i]
	}
}
