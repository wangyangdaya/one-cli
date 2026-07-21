package openapi

type Document struct {
	Title      string
	Version    string
	Tags       []Tag
	Operations []Operation
}

type Tag struct {
	Name        string
	Description string
}

type Operation struct {
	Method      string
	Path        string
	Tag         string
	OperationID string
	Summary     string
	Backend     string
	Endpoint    string
	Headers     map[string]string
	Command     string
	Args        []string
	Env         map[string]string
	Parameters  []Parameter
	RequestBody RequestBody
	Responses   []Response
}

type Parameter struct {
	Name        string
	In          string
	Required    bool
	Description string
	Type        string
}

type RequestBody struct {
	Required         bool
	ContentTypes     []string
	HasJSONSchema    bool
	IsSimpleJSON     bool
	JSONFields       []BodyField
	JSONSchemaFields []BodyField
	FileFields       []BodyField
}

type BodyField struct {
	Name            string
	Description     string
	Required        bool
	RequiredUnknown bool
	Type            string
	Format          string
}

type Response struct {
	Status      string
	ContentType string
	Description string
	Schemas     []Schema
}

type Schema struct {
	Name        string
	Description string
	Type        string
	Fields      []SchemaField
}

type SchemaField struct {
	Name        string
	Description string
	Required    bool
	Type        string
}
