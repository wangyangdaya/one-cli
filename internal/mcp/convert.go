package mcp

import (
	"fmt"
	"strings"

	"one-cli/internal/openapi"
)

func ConvertTools(serverName string, tools []Tool) (openapi.Document, error) {
	return ConvertServer(serverName, ServerConfig{}, tools)
}

func ConvertServer(serverName string, server ServerConfig, tools []Tool) (openapi.Document, error) {
	serverName = strings.TrimSpace(serverName)
	doc := openapi.Document{
		Title: serverName,
		Tags: []openapi.Tag{
			{Name: serverName},
		},
		Operations: make([]openapi.Operation, 0, len(tools)),
	}

	for _, tool := range tools {
		body, err := convertInputSchema(tool.InputSchema)
		if err != nil {
			return openapi.Document{}, fmt.Errorf("tool %q: %w", tool.Name, err)
		}
		doc.Operations = append(doc.Operations, openapi.Operation{
			Method:      "MCP",
			Path:        "/" + strings.TrimSpace(tool.Name),
			Tag:         serverName,
			OperationID: strings.TrimSpace(tool.Name),
			Summary:     strings.TrimSpace(tool.Description),
			Backend:     backendForServer(server),
			Endpoint:    strings.TrimSpace(server.URL),
			Headers:     cloneMap(server.Headers),
			Command:     strings.TrimSpace(server.Command),
			Args:        append([]string(nil), server.Args...),
			Env:         cloneMap(server.Env),
			RequestBody: body,
		})
	}

	return doc, nil
}

func convertInputSchema(schema map[string]any) (openapi.RequestBody, error) {
	if len(schema) == 0 {
		return openapi.RequestBody{}, nil
	}

	body := openapi.RequestBody{
		ContentTypes:  []string{"application/json"},
		HasJSONSchema: true,
		IsSimpleJSON:  true,
	}

	if strings.TrimSpace(asString(schema["type"])) != "object" {
		body.IsSimpleJSON = false
		return body, nil
	}

	required := requiredFields(schema["required"])
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		body.IsSimpleJSON = false
		return body, nil
	}

	body.JSONFields = make([]openapi.BodyField, 0, len(properties))
	for name, rawProperty := range properties {
		property, ok := rawProperty.(map[string]any)
		if !ok {
			body.IsSimpleJSON = false
			body.JSONFields = nil
			return body, nil
		}

		fieldType := strings.TrimSpace(asString(property["type"]))
		switch fieldType {
		case "string", "integer", "number", "boolean":
		default:
			body.IsSimpleJSON = false
			body.JSONFields = nil
			return body, nil
		}

		body.JSONFields = append(body.JSONFields, openapi.BodyField{
			Name:        strings.TrimSpace(name),
			Description: strings.TrimSpace(asString(property["description"])),
			Required:    required[name],
			Type:        fieldType,
		})
	}

	if len(body.JSONFields) == 0 {
		body.IsSimpleJSON = false
	}

	return body, nil
}

func requiredFields(value any) map[string]bool {
	required := make(map[string]bool)
	items, ok := value.([]any)
	if !ok {
		return required
	}
	for _, item := range items {
		name := strings.TrimSpace(asString(item))
		if name != "" {
			required[name] = true
		}
	}
	return required
}

func asString(value any) string {
	text, _ := value.(string)
	return text
}

func parseToolsResult(result map[string]any) ([]Tool, error) {
	rawTools, ok := result["tools"].([]any)
	if !ok {
		return nil, fmt.Errorf("missing tools result")
	}

	tools := make([]Tool, 0, len(rawTools))
	for _, rawTool := range rawTools {
		toolMap, ok := rawTool.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid tool payload")
		}

		schema, _ := toolMap["inputSchema"].(map[string]any)
		tools = append(tools, Tool{
			Name:        strings.TrimSpace(asString(toolMap["name"])),
			Description: strings.TrimSpace(asString(toolMap["description"])),
			InputSchema: schema,
		})
	}

	return tools, nil
}

func backendForServer(server ServerConfig) string {
	switch strings.TrimSpace(server.Transport) {
	case "streamable_http":
		return "mcp-streamable-http"
	case "stdio":
		return "mcp-stdio"
	default:
		return ""
	}
}

func cloneMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
