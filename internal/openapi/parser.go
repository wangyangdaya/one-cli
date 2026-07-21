package openapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"slices"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"github.com/getkin/kin-openapi/openapi3"
	yamljson "github.com/oasdiff/yaml"
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
		if err := yamljson.Unmarshal(data, &doc2); err != nil {
			return nil, fmt.Errorf("decode swagger 2.0: %w", err)
		}
		normalizeSwagger2MultipleBodyParameters(&doc2)
		ensureSwagger2DefinitionRefs(&doc2)
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

func normalizeSwagger2MultipleBodyParameters(doc *openapi2.T) {
	if doc == nil {
		return
	}
	for _, pathItem := range doc.Paths {
		if pathItem == nil {
			continue
		}
		for _, op := range pathItem.Operations() {
			if op == nil {
				continue
			}
			seenBody := false
			for _, param := range op.Parameters {
				if param == nil || strings.TrimSpace(param.In) != "body" {
					continue
				}
				if !seenBody {
					seenBody = true
					continue
				}
				param.In = "query"
				copySwagger2SchemaToParameter(param)
				param.Schema = nil
			}
		}
	}
}

func copySwagger2SchemaToParameter(param *openapi2.Parameter) {
	if param == nil || param.Schema == nil || param.Schema.Value == nil {
		return
	}
	schema := param.Schema.Value
	if schema.Type != nil {
		param.Type = schema.Type
	}
	param.Format = schema.Format
	param.Default = schema.Default
	param.Enum = append([]any(nil), schema.Enum...)
	param.Items = schema.Items
	param.Pattern = schema.Pattern
	param.MultipleOf = schema.MultipleOf
	param.Minimum = schema.Min
	param.Maximum = schema.Max
	param.MinLength = schema.MinLength
	param.MaxLength = schema.MaxLength
	param.MinItems = schema.MinItems
	param.MaxItems = schema.MaxItems
	param.UniqueItems = schema.UniqueItems
}

func ensureSwagger2DefinitionRefs(doc *openapi2.T) {
	if doc == nil {
		return
	}
	refs := make(map[string]struct{})
	for _, schema := range doc.Definitions {
		collectSwagger2SchemaRefs(schema, refs)
	}
	for _, param := range doc.Parameters {
		collectSwagger2ParameterRefs(param, refs)
	}
	for _, response := range doc.Responses {
		collectSwagger2ResponseRefs(response, refs)
	}
	for _, pathItem := range doc.Paths {
		if pathItem == nil {
			continue
		}
		for _, param := range pathItem.Parameters {
			collectSwagger2ParameterRefs(param, refs)
		}
		for _, op := range pathItem.Operations() {
			if op == nil {
				continue
			}
			for _, param := range op.Parameters {
				collectSwagger2ParameterRefs(param, refs)
			}
			for _, response := range op.Responses {
				collectSwagger2ResponseRefs(response, refs)
			}
		}
	}
	if len(refs) == 0 {
		return
	}
	if doc.Definitions == nil {
		doc.Definitions = make(map[string]*openapi2.SchemaRef)
	}
	for name := range refs {
		if _, ok := doc.Definitions[name]; ok {
			continue
		}
		doc.Definitions[name] = &openapi2.SchemaRef{
			Value: &openapi2.Schema{Type: &openapi3.Types{"object"}},
		}
	}
}

func collectSwagger2ParameterRefs(param *openapi2.Parameter, refs map[string]struct{}) {
	if param == nil {
		return
	}
	collectSwagger2SchemaRefs(param.Schema, refs)
	collectSwagger2SchemaRefs(param.Items, refs)
}

func collectSwagger2ResponseRefs(response *openapi2.Response, refs map[string]struct{}) {
	if response == nil {
		return
	}
	collectSwagger2DefinitionRef(response.Ref, refs)
	collectSwagger2SchemaRefs(response.Schema, refs)
	for _, header := range response.Headers {
		if header != nil {
			collectSwagger2ParameterRefs(&header.Parameter, refs)
		}
	}
}

func collectSwagger2SchemaRefs(ref *openapi2.SchemaRef, refs map[string]struct{}) {
	if ref == nil {
		return
	}
	collectSwagger2DefinitionRef(ref.Ref, refs)
	if ref.Value == nil {
		return
	}
	schema := ref.Value
	for _, child := range schema.AllOf {
		collectSwagger2SchemaRefs(child, refs)
	}
	collectSwagger2SchemaRefs(schema.Not, refs)
	collectSwagger2SchemaRefs(schema.Items, refs)
	for _, child := range schema.Properties {
		collectSwagger2SchemaRefs(child, refs)
	}
}

func collectSwagger2DefinitionRef(ref string, refs map[string]struct{}) {
	name, ok := strings.CutPrefix(strings.TrimSpace(ref), "#/definitions/")
	if !ok || strings.TrimSpace(name) == "" {
		return
	}
	refs[name] = struct{}{}
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
	for _, path := range slices.Sorted(maps.Keys(pathMap)) {
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
		Responses:   convertResponses(op.Responses),
	}
	if len(op.Tags) > 0 {
		result.Tag = strings.TrimSpace(op.Tags[0])
	}
	return result
}

func convertResponses(responses *openapi3.Responses) []Response {
	if responses == nil {
		return nil
	}
	responseMap := responses.Map()
	var out []Response
	for _, status := range slices.Sorted(maps.Keys(responseMap)) {
		ref := responseMap[status]
		if ref == nil || ref.Value == nil {
			continue
		}
		description := ""
		if ref.Value.Description != nil {
			description = strings.TrimSpace(*ref.Value.Description)
		}
		if len(ref.Value.Content) == 0 {
			out = append(out, Response{Status: status, Description: description})
			continue
		}
		for _, contentType := range slices.Sorted(maps.Keys(ref.Value.Content)) {
			mediaType := ref.Value.Content[contentType]
			response := Response{
				Status:      status,
				ContentType: strings.TrimSpace(contentType),
				Description: description,
			}
			if mediaType != nil && mediaType.Schema != nil && mediaType.Schema.Value != nil {
				response.Schemas = collectResponseSchemas(mediaType.Schema)
			}
			out = append(out, response)
		}
	}
	return out
}

func collectResponseSchemas(root *openapi3.SchemaRef) []Schema {
	var schemas []Schema
	visited := make(map[string]bool)
	collectResponseSchema(root, "response", visited, &schemas)
	return schemas
}

func collectResponseSchema(ref *openapi3.SchemaRef, fallbackName string, visited map[string]bool, schemas *[]Schema) {
	if ref == nil || ref.Value == nil {
		return
	}
	name := schemaRefName(ref)
	if name == "" {
		name = strings.TrimSpace(ref.Value.Title)
	}
	if name == "" {
		name = fallbackName
	}
	key := name
	if visited[key] {
		return
	}
	visited[key] = true

	index := len(*schemas)
	*schemas = append(*schemas, Schema{})
	schema := Schema{
		Name:        name,
		Description: schemaDescription(ref.Value),
		Type:        schemaType(ref.Value),
	}

	properties, required := responseSchemaProperties(ref.Value)
	requiredSet := make(map[string]bool, len(required))
	for _, fieldName := range required {
		requiredSet[strings.TrimSpace(fieldName)] = true
	}
	for _, fieldName := range slices.Sorted(maps.Keys(properties)) {
		fieldRef := properties[fieldName]
		if fieldRef == nil || fieldRef.Value == nil {
			continue
		}
		schema.Fields = append(schema.Fields, SchemaField{
			Name:        strings.TrimSpace(fieldName),
			Description: schemaDescription(fieldRef.Value),
			Required:    requiredSet[strings.TrimSpace(fieldName)],
			Type:        responseSchemaType(fieldRef),
		})
		collectReferencedResponseSchemas(fieldRef, visited, schemas)
	}
	(*schemas)[index] = schema
}

func collectReferencedResponseSchemas(ref *openapi3.SchemaRef, visited map[string]bool, schemas *[]Schema) {
	if ref == nil || ref.Value == nil {
		return
	}
	if schemaRefName(ref) != "" {
		collectResponseSchema(ref, "", visited, schemas)
		return
	}
	if ref.Value.Items != nil {
		collectReferencedResponseSchemas(ref.Value.Items, visited, schemas)
	}
}

func responseSchemaProperties(schema *openapi3.Schema) (openapi3.Schemas, []string) {
	if len(schema.AllOf) > 0 {
		return flattenAllOf(schema)
	}
	return schema.Properties, schema.Required
}

func responseSchemaType(ref *openapi3.SchemaRef) string {
	if name := schemaRefName(ref); name != "" {
		return name
	}
	if ref == nil || ref.Value == nil {
		return ""
	}
	if ref.Value.Items != nil {
		itemType := responseSchemaType(ref.Value.Items)
		if itemType == "" {
			itemType = "unknown"
		}
		return itemType + "[]"
	}
	return schemaType(ref.Value)
}

func schemaRefName(ref *openapi3.SchemaRef) string {
	if ref == nil {
		return ""
	}
	value := strings.TrimSpace(ref.Ref)
	if index := strings.LastIndex(value, "/"); index >= 0 {
		value = value[index+1:]
	}
	return strings.TrimSpace(value)
}

func schemaDescription(schema *openapi3.Schema) string {
	if schema == nil {
		return ""
	}
	if description := strings.TrimSpace(schema.Description); description != "" {
		return description
	}
	return strings.TrimSpace(schema.Title)
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
	for _, contentType := range rb.ContentTypes {
		mediaType := content[contentType]
		if mediaType == nil || mediaType.Schema == nil || mediaType.Schema.Value == nil {
			continue
		}
		if isBinarySchema(mediaType.Schema.Value) {
			rb.FileFields = appendUniqueBodyFields(rb.FileFields, BodyField{
				Name: "file", Required: rb.Required, Type: "string", Format: "binary",
			})
		}
	}

	if mediaType, ok := content["application/json"]; ok && mediaType != nil && mediaType.Schema != nil && mediaType.Schema.Value != nil {
		rb.HasJSONSchema = true
		rb.JSONSchemaFields = collectJSONSchemaFields(mediaType.Schema.Value)
		rb.FileFields = binaryBodyFields(rb.JSONSchemaFields)
		rb.IsSimpleJSON, rb.JSONFields = classifySimpleJSON(mediaType.Schema.Value)
		if len(rb.JSONSchemaFields) == 0 {
			if fields := inferJSONFieldsFromExample(mediaType.Example); len(fields) > 0 {
				rb.JSONSchemaFields = fields
				rb.IsSimpleJSON = true
				rb.JSONFields = fields
			}
		}
	}
	if mediaType, ok := content["multipart/form-data"]; ok && mediaType != nil && mediaType.Schema != nil && mediaType.Schema.Value != nil {
		rb.FileFields = appendUniqueBodyFields(rb.FileFields, binaryBodyFields(collectJSONSchemaFields(mediaType.Schema.Value))...)
	}
	return rb
}

func binaryBodyFields(fields []BodyField) []BodyField {
	result := make([]BodyField, 0)
	for _, field := range fields {
		if strings.EqualFold(strings.TrimSpace(field.Format), "binary") {
			result = append(result, field)
		}
	}
	return result
}

func isBinarySchema(schema *openapi3.Schema) bool {
	if schema == nil || schema.Type == nil || !slices.Contains(schema.Type.Slice(), "string") {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(schema.Format), "binary")
}

func appendUniqueBodyFields(fields []BodyField, additions ...BodyField) []BodyField {
	seen := make(map[string]bool, len(fields))
	for _, field := range fields {
		seen[strings.TrimSpace(field.Name)] = true
	}
	for _, field := range additions {
		name := strings.TrimSpace(field.Name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		fields = append(fields, field)
	}
	return fields
}

func inferJSONFieldsFromExample(example any) []BodyField {
	values := exampleObjectFields(example)
	if len(values) == 0 || len(values) > MaxSimpleJSONFields {
		return nil
	}

	fields := make([]BodyField, 0, len(values))
	for _, value := range values {
		fieldType := exampleValueType(value.Value)
		if fieldType == "" {
			return nil
		}
		fields = append(fields, BodyField{
			Name:            strings.TrimSpace(value.Name),
			RequiredUnknown: true,
			Type:            fieldType,
		})
	}
	return fields
}

type exampleField struct {
	Name  string
	Value any
}

func exampleObjectFields(example any) []exampleField {
	switch value := example.(type) {
	case string:
		return exampleObjectStringFields(value)
	case map[string]any:
		keys := slices.Sorted(maps.Keys(value))
		fields := make([]exampleField, 0, len(keys))
		for _, key := range keys {
			fields = append(fields, exampleField{Name: key, Value: value[key]})
		}
		return fields
	default:
		return nil
	}
}

func exampleObjectStringFields(raw string) []exampleField {
	decoder := json.NewDecoder(strings.NewReader(strings.TrimSpace(raw)))
	decoder.UseNumber()

	token, err := decoder.Token()
	if err != nil {
		return nil
	}
	if delim, ok := token.(json.Delim); !ok || delim != '{' {
		return nil
	}

	var fields []exampleField
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil
		}
		name, ok := token.(string)
		if !ok {
			return nil
		}
		var value any
		if err := decoder.Decode(&value); err != nil {
			return nil
		}
		fields = append(fields, exampleField{Name: name, Value: value})
	}

	token, err = decoder.Token()
	if err != nil {
		return nil
	}
	if delim, ok := token.(json.Delim); !ok || delim != '}' {
		return nil
	}
	return fields
}

func exampleValueType(value any) string {
	switch v := value.(type) {
	case string:
		return "string"
	case bool:
		return "boolean"
	case json.Number:
		if _, err := v.Int64(); err == nil {
			return "integer"
		}
		if _, err := v.Float64(); err == nil {
			return "number"
		}
	case float64:
		if math.Trunc(v) == v {
			return "integer"
		}
		return "number"
	case float32:
		if math.Trunc(float64(v)) == float64(v) {
			return "integer"
		}
		return "number"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return "integer"
	}
	return ""
}

func collectJSONSchemaFields(schema *openapi3.Schema) []BodyField {
	if !schemaIsObject(schema) || len(schema.OneOf) > 0 || len(schema.AnyOf) > 0 {
		return nil
	}

	var properties openapi3.Schemas
	var required []string
	if len(schema.AllOf) > 0 {
		properties, required = flattenAllOf(schema)
	} else {
		properties = schema.Properties
		required = schema.Required
	}
	if len(properties) == 0 {
		return nil
	}

	requiredSet := make(map[string]bool, len(required))
	for _, name := range required {
		requiredSet[strings.TrimSpace(name)] = true
	}

	keys := slices.Sorted(maps.Keys(properties))
	fields := make([]BodyField, 0, len(keys))
	for _, key := range keys {
		propRef := properties[key]
		if propRef == nil || propRef.Value == nil {
			continue
		}
		prop := propRef.Value
		fields = append(fields, BodyField{
			Name:        strings.TrimSpace(key),
			Description: strings.TrimSpace(prop.Description),
			Required:    requiredSet[strings.TrimSpace(key)],
			Type:        schemaType(prop),
			Format:      strings.TrimSpace(prop.Format),
		})
	}
	return fields
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

	keys := slices.Sorted(maps.Keys(properties))
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
			Format:      strings.TrimSpace(prop.Format),
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
