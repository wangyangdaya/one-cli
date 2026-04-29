package model

// Backend constants identify the transport layer for a Group.
const (
	BackendHTTP     = ""
	BackendMCPHTTP  = "mcp-streamable-http"
	BackendMCPStdio = "mcp-stdio"
)

// BodyMode constants identify how request bodies are rendered.
const (
	BodyModeSimpleJSON = "simple-json"
	BodyModeFileOrData = "file-or-data"
	BodyModeFlags      = "flags"
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
	Name   string
	Groups []Group
}

type Group struct {
	Name        string
	PackageName string
	Description string
	Backend     string
	Endpoint    string
	Headers     map[string]string
	Command     string
	Args        []string
	Env         map[string]string
	Operations  []Operation
}

type Operation struct {
	Method       string
	Path         string
	CommandName  string
	RemoteName   string
	Summary      string
	BodyMode     string
	BodyRequired bool
	BodyFields   []BodyField
	Parameters   []Parameter
}

type Parameter struct {
	Name        string
	In          string
	Required    bool
	Description string
	Type        string
}

type BodyField struct {
	Name        string
	Description string
	Required    bool
	Type        string
}
