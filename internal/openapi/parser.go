package openapi

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// MaxSimpleJSONFields is the maximum number of properties a JSON schema can have
// to be treated as "simple JSON" (individual CLI flags). Both the OpenAPI parser
// and the MCP converter reference this constant to stay in sync.
const MaxSimpleJSONFields = 5

func Parse(data []byte) (Document, error) {
	if len(data) == 0 {
		return Document{}, nil
	}

	var raw rawDocument
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(false)
	if err := dec.Decode(&raw); err != nil {
		return Document{}, fmt.Errorf("decode openapi: %w", err)
	}

	return normalizeDocument(raw), nil
}

type rawDocument struct {
	OpenAPI    string             `yaml:"openapi"`
	Swagger    string             `yaml:"swagger"`
	Info       rawInfo            `yaml:"info"`
	Tags       []rawTag           `yaml:"tags"`
	Paths      map[string]rawPath `yaml:"paths"`
	Components rawComponents      `yaml:"components"`
}

type rawComponents struct {
	Schemas    map[string]rawSchema    `yaml:"schemas"`
	Parameters map[string]rawParameter `yaml:"parameters"`
}

type rawInfo struct {
	Title   string `yaml:"title"`
	Version string `yaml:"version"`
}

type rawTag struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type rawPath struct {
	Get     rawOperation `yaml:"get"`
	Put     rawOperation `yaml:"put"`
	Post    rawOperation `yaml:"post"`
	Delete  rawOperation `yaml:"delete"`
	Patch   rawOperation `yaml:"patch"`
	Head    rawOperation `yaml:"head"`
	Options rawOperation `yaml:"options"`
	Trace   rawOperation `yaml:"trace"`
}

type rawOperation struct {
	Tags        []string       `yaml:"tags"`
	OperationID string         `yaml:"operationId"`
	Summary     string         `yaml:"summary"`
	Parameters  []rawParameter `yaml:"parameters"`
	RequestBody rawRequestBody `yaml:"requestBody"`
}

type rawParameter struct {
	Ref         string             `yaml:"$ref"`
	Name        string             `yaml:"name"`
	In          string             `yaml:"in"`
	Required    bool               `yaml:"required"`
	Description string             `yaml:"description"`
	Schema      rawParameterSchema `yaml:"schema"`
	Type        string             `yaml:"type"`
}

type rawParameterSchema struct {
	Type string `yaml:"type"`
}

type rawRequestBody struct {
	Required bool                    `yaml:"required"`
	Content  map[string]rawMediaType `yaml:"content"`
}

type rawMediaType struct {
	Schema rawSchema `yaml:"schema"`
}

type rawSchema struct {
	Ref         string               `yaml:"$ref"`
	Type        string               `yaml:"type"`
	Description string               `yaml:"description"`
	Required    []string             `yaml:"required"`
	Properties  map[string]rawSchema `yaml:"properties"`
	Items       *rawSchema           `yaml:"items"`
	AnyOf       []rawSchema          `yaml:"anyOf"`
	OneOf       []rawSchema          `yaml:"oneOf"`
	AllOf       []rawSchema          `yaml:"allOf"`
}

func normalizeDocument(raw rawDocument) Document {
	doc := Document{
		Title:   strings.TrimSpace(raw.Info.Title),
		Version: strings.TrimSpace(raw.Info.Version),
		Tags:    normalizeTags(raw.Tags),
	}

	for _, path := range sortedKeys(raw.Paths) {
		item := raw.Paths[path]
		doc.Operations = append(doc.Operations, normalizeOperations(path, item, raw.Components)...)
	}

	return doc
}

func normalizeTags(tags []rawTag) []Tag {
	out := make([]Tag, 0, len(tags))
	for _, tag := range tags {
		out = append(out, Tag{
			Name:        strings.TrimSpace(tag.Name),
			Description: strings.TrimSpace(tag.Description),
		})
	}
	return out
}

func normalizeOperations(path string, item rawPath, components rawComponents) []Operation {
	operations := []struct {
		method string
		raw    rawOperation
	}{
		{method: "get", raw: item.Get},
		{method: "put", raw: item.Put},
		{method: "post", raw: item.Post},
		{method: "delete", raw: item.Delete},
		{method: "patch", raw: item.Patch},
		{method: "head", raw: item.Head},
		{method: "options", raw: item.Options},
		{method: "trace", raw: item.Trace},
	}

	out := make([]Operation, 0, len(operations))
	for _, op := range operations {
		if isEmptyOperation(op.raw) {
			continue
		}
		out = append(out, normalizeOperation(path, op.method, op.raw, components))
	}

	return out
}

func normalizeOperation(path, method string, raw rawOperation, components rawComponents) Operation {
	op := Operation{
		Method:      strings.ToUpper(strings.TrimSpace(method)),
		Path:        strings.TrimSpace(path),
		OperationID: strings.TrimSpace(raw.OperationID),
		Summary:     strings.TrimSpace(raw.Summary),
		Parameters:  normalizeParameters(raw.Parameters, components.Parameters),
		RequestBody: normalizeRequestBody(raw.RequestBody, components.Schemas),
	}
	if len(raw.Tags) > 0 {
		op.Tag = strings.TrimSpace(raw.Tags[0])
	}
	return op
}

func normalizeParameters(parameters []rawParameter, refs map[string]rawParameter) []Parameter {
	out := make([]Parameter, 0, len(parameters))
	for _, parameter := range parameters {
		parameter = resolveParameter(parameter, refs, nil)
		out = append(out, Parameter{
			Name:        strings.TrimSpace(parameter.Name),
			In:          strings.TrimSpace(parameter.In),
			Required:    parameter.Required,
			Description: strings.TrimSpace(parameter.Description),
			Type:        parameterType(parameter),
		})
	}
	return out
}

func resolveParameter(parameter rawParameter, refs map[string]rawParameter, seen map[string]bool) rawParameter {
	ref := strings.TrimSpace(parameter.Ref)
	if ref == "" {
		return parameter
	}
	const prefix = "#/components/parameters/"
	if !strings.HasPrefix(ref, prefix) {
		return parameter
	}
	name := strings.TrimPrefix(ref, prefix)
	if name == "" {
		return parameter
	}
	if seen == nil {
		seen = map[string]bool{}
	}
	if seen[name] {
		return parameter
	}
	referenced, ok := refs[name]
	if !ok {
		return parameter
	}
	seen[name] = true
	return resolveParameter(referenced, refs, seen)
}

func parameterType(parameter rawParameter) string {
	if strings.TrimSpace(parameter.Type) != "" {
		return strings.TrimSpace(parameter.Type)
	}
	return strings.TrimSpace(parameter.Schema.Type)
}

func normalizeRequestBody(body rawRequestBody, schemas map[string]rawSchema) RequestBody {
	requestBody := RequestBody{Required: body.Required}
	if len(body.Content) == 0 {
		return requestBody
	}

	requestBody.ContentTypes = make([]string, 0, len(body.Content))
	for contentType := range body.Content {
		requestBody.ContentTypes = append(requestBody.ContentTypes, contentType)
	}
	sort.Strings(requestBody.ContentTypes)
	if mediaType, ok := body.Content["application/json"]; ok {
		requestBody.HasJSONSchema = true
		requestBody.IsSimpleJSON, requestBody.JSONFields = normalizeSimpleJSONFields(resolveSchema(mediaType.Schema, schemas, nil), schemas)
	}
	return requestBody
}

func normalizeSimpleJSONFields(schema rawSchema, schemas map[string]rawSchema) (bool, []BodyField) {
	if strings.TrimSpace(schema.Type) != "object" {
		return false, nil
	}
	if len(schema.AnyOf) > 0 || len(schema.OneOf) > 0 || len(schema.AllOf) > 0 {
		return false, nil
	}
	if len(schema.Properties) == 0 || len(schema.Properties) > MaxSimpleJSONFields {
		return false, nil
	}

	required := make(map[string]bool, len(schema.Required))
	for _, name := range schema.Required {
		required[strings.TrimSpace(name)] = true
	}

	keys := sortedKeys(schema.Properties)
	fields := make([]BodyField, 0, len(keys))
	for _, key := range keys {
		property := resolveSchema(schema.Properties[key], schemas, nil)
		switch strings.TrimSpace(property.Type) {
		case "string", "integer", "number", "boolean":
		default:
			return false, nil
		}
		if property.Items != nil || len(property.Properties) > 0 || len(property.AnyOf) > 0 || len(property.OneOf) > 0 || len(property.AllOf) > 0 {
			return false, nil
		}
		fields = append(fields, BodyField{
			Name:        strings.TrimSpace(key),
			Description: strings.TrimSpace(property.Description),
			Required:    required[strings.TrimSpace(key)],
			Type:        strings.TrimSpace(property.Type),
		})
	}
	return true, fields
}

func resolveSchema(schema rawSchema, schemas map[string]rawSchema, seen map[string]bool) rawSchema {
	ref := strings.TrimSpace(schema.Ref)
	if ref == "" {
		return schema
	}
	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(ref, prefix) {
		return schema
	}
	name := strings.TrimSpace(strings.TrimPrefix(ref, prefix))
	if name == "" {
		return schema
	}
	if seen == nil {
		seen = make(map[string]bool)
	}
	if seen[name] {
		return schema
	}
	resolved, ok := schemas[name]
	if !ok {
		return schema
	}
	seen[name] = true
	return resolveSchema(resolved, schemas, seen)
}

func isEmptyOperation(raw rawOperation) bool {
	return len(raw.Tags) == 0 &&
		strings.TrimSpace(raw.OperationID) == "" &&
		strings.TrimSpace(raw.Summary) == "" &&
		len(raw.Parameters) == 0 &&
		!raw.RequestBody.Required &&
		len(raw.RequestBody.Content) == 0
}

func sortedKeys[T any](values map[string]T) []string {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
