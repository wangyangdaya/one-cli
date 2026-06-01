package configgen

type Config struct {
	App       AppConfig      `yaml:"app"`
	Naming    NamingConfig   `yaml:"naming"`
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

type OverrideConfig struct {
	BodyMode map[string]string `yaml:"body_mode"`
}
