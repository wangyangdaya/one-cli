package model

type App struct {
	Name   string
	Groups []Group
}

type Group struct {
	Name        string
	Description string
	Operations  []Operation
}

type Operation struct {
	Method       string
	Path         string
	CommandName  string
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
