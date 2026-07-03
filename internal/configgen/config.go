package configgen

type Config struct {
	App       AppConfig      `yaml:"app"`
	Auth      AuthConfig     `yaml:"auth"`
	Naming    NamingConfig   `yaml:"naming"`
	Runtime   RuntimeConfig  `yaml:"runtime"`
	Overrides OverrideConfig `yaml:"overrides"`
}

type AppConfig struct {
	Binary      string `yaml:"binary"`
	RootCommand string `yaml:"root_command"`
	Version     string `yaml:"version"`
}

type AuthConfig struct {
	Type   string       `yaml:"type"`
	Signer SignerConfig `yaml:"signer"`
}

type SignerConfig struct {
	Profile   string              `yaml:"profile"`
	Algorithm string              `yaml:"algorithm"`
	Headers   SignerHeadersConfig `yaml:"headers"`
	Path      SignerPathConfig    `yaml:"path"`
	Body      SignerBodyConfig    `yaml:"body"`
	Canonical SignerCanonical     `yaml:"canonical"`
}

type SignerHeadersConfig struct {
	AccessKey string `yaml:"access_key"`
	Signature string `yaml:"signature"`
	Timestamp string `yaml:"timestamp"`
	Nonce     string `yaml:"nonce"`
}

type SignerPathConfig struct {
	StripPrefix string `yaml:"strip_prefix"`
}

type SignerBodyConfig struct {
	Order string `yaml:"order"`
}

type SignerCanonical struct {
	Template string `yaml:"template"`
}

type NamingConfig struct {
	TagAlias       map[string]string `yaml:"tag_alias"`
	OperationAlias map[string]string `yaml:"operation_alias"`
}

type RuntimeConfig struct {
	AuthHeader    string `yaml:"auth_header"`
	DefaultOutput string `yaml:"default_output"`
}

type OverrideConfig struct {
	BodyMode   map[string]string      `yaml:"body_mode"`
	BodyFields map[string][]BodyField `yaml:"body_fields"`
}

type BodyField struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Required    *bool  `yaml:"required"`
	Type        string `yaml:"type"`
}
