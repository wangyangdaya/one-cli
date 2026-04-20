package configgen

type Config struct {
	App       AppConfig      `yaml:"app"`
	Naming    NamingConfig   `yaml:"naming"`
	Runtime   RuntimeConfig  `yaml:"runtime"`
	Overrides OverrideConfig `yaml:"overrides"`
}

type AppConfig struct {
	Binary      string `yaml:"binary"`
	RootCommand string `yaml:"root_command"`
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
	BodyMode map[string]string `yaml:"body_mode"`
}
