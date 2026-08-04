package model

// Backend constants identify the transport layer for a Group.
const (
	BackendHTTP     = ""
	BackendMCPHTTP  = "mcp-streamable-http"
	BackendMCPStdio = "mcp-stdio"
)

// Auth type constants identify generated runtime authentication behavior.
const (
	AuthTypeNone   = "none"
	AuthTypeToken  = "token"
	AuthTypeAPIKey = "api_key"
	AuthTypeAKSK   = "ak_sk"
	AuthTypeOAuth2 = "oauth2"
)

// Signer profile constants identify concrete AK/SK signing contracts.
const (
	SignerProfileSupplierEDI = "supplier_edi"
)

// SignerAlgorithm constants identify supported signing algorithms.
const (
	SignerAlgorithmSHA512Hex = "sha512_hex"
)

// BodyMode constants identify how request bodies are rendered.
const (
	BodyModeSimpleJSON     = "simple-json"
	BodyModeFormURLEncoded = "form-urlencoded"
	BodyModeFileOrData     = "file-or-data"
	BodyModeFlags          = "flags"
)

// CloneStringMap returns a shallow copy of a string map, or nil for empty maps.
func CloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

type App struct {
	Name    string
	Version string
	Auth    Auth
	Groups  []Group
}

type Auth struct {
	Type          string
	SignerProfile string
	Signer        Signer
}

type Signer struct {
	Profile           string
	Algorithm         string
	AccessKeyHeader   string
	SignatureHeader   string
	TimestampHeader   string
	NonceHeader       string
	PathStripPrefix   string
	BodyOrder         string
	CanonicalTemplate string
}

type Group struct {
	Name        string
	PackageName string
	Description string
	RenamedFrom string
	Backend     string
	Endpoint    string
	Headers     map[string]string
	Command     string
	Args        []string
	Env         map[string]string
	Operations  []Operation
}

type Operation struct {
	Method           string
	Path             string
	Servers          []string
	CommandName      string
	RemoteName       string
	Summary          string
	AuthRequired     bool
	BodyMode         string
	BodyRequired     bool
	BodyFields       []BodyField
	FileFields       []BodyField
	BodySchemaFields []BodyField
	Parameters       []Parameter
	Responses        []Response
}

type Parameter struct {
	Name              string
	FieldName         string
	FlagName          string
	PreferredFlagName string
	In                string
	Required          bool
	Description       string
	Type              string
	Example           string
	JSONSchemaName    string
	JSONFields        []BodyField
}

type BodyField struct {
	Name            string
	FieldName       string
	FlagName        string
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
