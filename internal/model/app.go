package model

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
