package openapi

type Document struct {
	Title           string
	Version         string
	Tags            []Tag
	SecuritySchemes []SecurityScheme
	Operations      []Operation
}

type SecurityScheme struct {
	Name                      string
	Type                      string
	ClientCredentialsTokenURL string
	ClientCredentialsScopes   []string
}

type Tag struct {
	Name        string
	Description string
}

type Operation struct {
	Method      string
	Path        string
	Servers     []string
	Tag         string
	OperationID string
	Summary     string
	Security    []string
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
	Name           string
	FlagName       string
	In             string
	Required       bool
	Description    string
	Type           string
	Example        string
	JSONSchemaName string
	JSONFields     []BodyField
}

type RequestBody struct {
	Required         bool
	ContentTypes     []string
	HasJSONSchema    bool
	IsSimpleJSON     bool
	JSONFields       []BodyField
	JSONSchemaFields []BodyField
	FormFields       []BodyField
	FileFields       []BodyField
}

type BodyField struct {
	Name            string
	Description     string
	Required        bool
	RequiredUnknown bool
	Type            string
	Format          string
	Example         string
	JSONSchemaName  string
	JSONFields      []BodyField
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
