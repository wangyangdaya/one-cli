package openapi

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v3"
)

// MaxSimpleJSONFields is the maximum number of properties a JSON schema can have
// to be treated as "simple JSON" (individual CLI flags). Both the OpenAPI parser
// and the MCP converter reference this constant to stay in sync.
const MaxSimpleJSONFields = 5

// Parse parses raw bytes (YAML or JSON) into a Document.
// It supports OpenAPI 2.0 (Swagger), 3.0, and 3.1.
func Parse(data []byte) (Document, error) {
	if len(data) == 0 {
		return Document{}, nil
	}

	version, err := detectVersion(data)
	if err != nil {
		return Document{}, err
	}

	doc3, err := loadAsOpenAPI3(data, version)
	if err != nil {
		return Document{}, err
	}

	return convertDocument(doc3), nil
}

// versionHeader is used to extract the version identifier from raw bytes.
type versionHeader struct {
	OpenAPI string `yaml:"openapi" json:"openapi"`
	Swagger string `yaml:"swagger" json:"swagger"`
}

// detectVersion extracts the OpenAPI/Swagger version from raw bytes.
// Returns "2.0", "3.0", or "3.1".
func detectVersion(data []byte) (string, error) {
	var header versionHeader
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(false)
	if err := dec.Decode(&header); err != nil {
		return "", fmt.Errorf("decode openapi: %w", err)
	}

	if strings.TrimSpace(header.Swagger) == "2.0" {
		return "2.0", nil
	}
	if strings.HasPrefix(strings.TrimSpace(header.OpenAPI), "3.1") {
		return "3.1", nil
	}
	if strings.HasPrefix(strings.TrimSpace(header.OpenAPI), "3.0") {
		return "3.0", nil
	}
	return "", fmt.Errorf("unsupported or missing OpenAPI version")
}

// loadAsOpenAPI3 loads raw bytes into an openapi3.T document.
// For OpenAPI 3.x it uses the kin-openapi loader directly.
// For Swagger 2.0 it unmarshals into openapi2.T then converts via openapi2conv.
func loadAsOpenAPI3(data []byte, version string) (*openapi3.T, error) {
	if version == "2.0" {
		var doc2 openapi2.T
		if err := yaml.Unmarshal(data, &doc2); err != nil {
			return nil, fmt.Errorf("decode swagger 2.0: %w", err)
		}
		doc3, err := openapi2conv.ToV3(&doc2)
		if err != nil {
			return nil, fmt.Errorf("convert swagger 2.0 to openapi 3.0: %w", err)
		}
		return doc3, nil
	}

	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(data)
	if err != nil {
		return nil, fmt.Errorf("decode openapi: %w", err)
	}
	return doc, nil
}

// convertDocument converts an openapi3.T into our internal Document model.
func convertDocument(doc *openapi3.T) Document {
	d := Document{}
	if doc.Info != nil {
		d.Title = strings.TrimSpace(doc.Info.Title)
		d.Version = strings.TrimSpace(doc.Info.Version)
	}
	d.Tags = convertTags(doc.Tags)
	d.Operations = convertOperations(doc.Paths)
	return d
}

// convertTags maps kin-openapi Tags to our internal Tag slice.
func convertTags(tags openapi3.Tags) []Tag {
	out := make([]Tag, 0, len(tags))
	for _, tag := range tags {
		if tag == nil {
			continue
		}
		out = append(out, Tag{
			Name:        strings.TrimSpace(tag.Name),
			Description: strings.TrimSpace(tag.Description),
		})
	}
	return out
}

// convertOperations iterates paths in sorted order and methods in a fixed order,
// producing a deterministic list of Operations.
func convertOperations(paths *openapi3.Paths) []Operation {
	if paths == nil {
		return nil
	}
	var ops []Operation
	pathMap := paths.Map()
	for _, path := range sortedKeys(pathMap) {
		item := pathMap[path]
		methods := []struct {
			name string
			op   *openapi3.Operation
		}{
			{"GET", item.Get},
			{"PUT", item.Put},
			{"POST", item.Post},
			{"DELETE", item.Delete},
			{"PATCH", item.Patch},
			{"HEAD", item.Head},
			{"OPTIONS", item.Options},
			{"TRACE", item.Trace},
		}
		for _, m := range methods {
			if m.op == nil {
				continue
			}
			ops = append(ops, convertOperation(path, m.name, m.op))
		}
	}
	return ops
}

// convertOperation maps a single kin-openapi Operation to our internal Operation.
func convertOperation(path, method string, op *openapi3.Operation) Operation {
	result := Operation{
		Method:      strings.TrimSpace(method),
		Path:        strings.TrimSpace(path),
		OperationID: strings.TrimSpace(op.OperationID),
		Summary:     strings.TrimSpace(op.Summary),
		Parameters:  convertParameters(op.Parameters),
		RequestBody: convertRequestBody(op.RequestBody),
	}
	if len(op.Tags) > 0 {
		result.Tag = strings.TrimSpace(op.Tags[0])
	}
	return result
}

// convertParameters maps kin-openapi Parameters to our internal Parameter slice.
func convertParameters(params openapi3.Parameters) []Parameter {
	out := make([]Parameter, 0, len(params))
	for _, ref := range params {
		if ref == nil || ref.Value == nil {
			continue
		}
		out = append(out, convertParameter(ref.Value))
	}
	return out
}

// convertParameter maps a single kin-openapi Parameter to our internal Parameter.
func convertParameter(param *openapi3.Parameter) Parameter {
	return Parameter{
		Name:        strings.TrimSpace(param.Name),
		In:          strings.TrimSpace(param.In),
		Required:    param.Required,
		Description: strings.TrimSpace(param.Description),
		Type:        parameterType(param),
	}
}

// parameterType extracts the type string from a kin-openapi Parameter's schema.
func parameterType(param *openapi3.Parameter) string {
	if param.Schema == nil || param.Schema.Value == nil {
		return ""
	}
	if param.Schema.Value.Type != nil {
		types := param.Schema.Value.Type.Slice()
		if len(types) > 0 {
			return strings.TrimSpace(types[0])
		}
	}
	return ""
}

// convertRequestBody maps a kin-openapi RequestBodyRef to our internal RequestBody.
func convertRequestBody(body *openapi3.RequestBodyRef) RequestBody {
	if body == nil || body.Value == nil {
		return RequestBody{}
	}
	rb := RequestBody{Required: body.Value.Required}
	content := body.Value.Content
	if len(content) == 0 {
		return rb
	}

	rb.ContentTypes = make([]string, 0, len(content))
	for ct := range content {
		rb.ContentTypes = append(rb.ContentTypes, ct)
	}
	sort.Strings(rb.ContentTypes)

	if mediaType, ok := content["application/json"]; ok && mediaType != nil && mediaType.Schema != nil && mediaType.Schema.Value != nil {
		rb.HasJSONSchema = true
		rb.IsSimpleJSON, rb.JSONFields = classifySimpleJSON(mediaType.Schema.Value)
	}
	return rb
}

// flattenAllOf merges all sub-schemas in an allOf array into a single set of
// properties and required fields.
func flattenAllOf(schema *openapi3.Schema) (openapi3.Schemas, []string) {
	properties := make(openapi3.Schemas)
	var required []string

	// Merge the top-level schema's own properties and required.
	for name, prop := range schema.Properties {
		properties[name] = prop
	}
	required = append(required, schema.Required...)

	// Merge each allOf sub-schema.
	for _, ref := range schema.AllOf {
		if ref.Value == nil {
			continue
		}
		sub := ref.Value
		for name, prop := range sub.Properties {
			properties[name] = prop
		}
		required = append(required, sub.Required...)
	}

	return properties, required
}

// classifySimpleJSON determines whether a schema qualifies as "simple JSON"
// (object with ≤ MaxSimpleJSONFields primitive-typed properties).
func classifySimpleJSON(schema *openapi3.Schema) (bool, []BodyField) {
	// Must be object type.
	if !schemaIsObject(schema) {
		return false, nil
	}

	// Top-level oneOf or anyOf → complex.
	if len(schema.OneOf) > 0 || len(schema.AnyOf) > 0 {
		return false, nil
	}

	var properties openapi3.Schemas
	var required []string

	if len(schema.AllOf) > 0 {
		properties, required = flattenAllOf(schema)
	} else {
		properties = schema.Properties
		required = schema.Required
	}

	if len(properties) == 0 || len(properties) > MaxSimpleJSONFields {
		return false, nil
	}

	requiredSet := make(map[string]bool, len(required))
	for _, name := range required {
		requiredSet[strings.TrimSpace(name)] = true
	}

	keys := sortedKeys(properties)
	fields := make([]BodyField, 0, len(keys))
	for _, key := range keys {
		propRef := properties[key]
		if propRef == nil || propRef.Value == nil {
			return false, nil
		}
		prop := propRef.Value

		// Check for complex sub-structures.
		if prop.Items != nil || len(prop.Properties) > 0 || len(prop.AnyOf) > 0 || len(prop.OneOf) > 0 || len(prop.AllOf) > 0 {
			return false, nil
		}

		propType := schemaType(prop)
		switch propType {
		case "string", "integer", "number", "boolean":
		default:
			return false, nil
		}

		fields = append(fields, BodyField{
			Name:        strings.TrimSpace(key),
			Description: strings.TrimSpace(prop.Description),
			Required:    requiredSet[strings.TrimSpace(key)],
			Type:        propType,
		})
	}
	return true, fields
}

// schemaIsObject checks whether a schema's type includes "object".
func schemaIsObject(schema *openapi3.Schema) bool {
	if schema.Type == nil {
		return false
	}
	for _, t := range schema.Type.Slice() {
		if t == "object" {
			return true
		}
	}
	return false
}

// schemaType returns the first type string from a schema's Type field.
func schemaType(schema *openapi3.Schema) string {
	if schema.Type == nil {
		return ""
	}
	types := schema.Type.Slice()
	if len(types) > 0 {
		return strings.TrimSpace(types[0])
	}
	return ""
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
