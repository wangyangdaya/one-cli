package configgen

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"one-cli/internal/model"

	"gopkg.in/yaml.v3"
)

func Load(path string) (Config, error) {
	if path == "" {
		return Config{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	return LoadBytes(data)
}

func LoadBytes(data []byte) (Config, error) {
	if len(data) == 0 {
		return Config{}, nil
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if err := validate(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func validate(cfg Config) error {
	switch strings.TrimSpace(cfg.Auth.Type) {
	case "", model.AuthTypeNone, model.AuthTypeToken, model.AuthTypeAKSK:
	default:
		return fmt.Errorf("auth.type must be one of none, token, or ak_sk, got %q", cfg.Auth.Type)
	}

	for _, key := range slices.Sorted(maps.Keys(cfg.Overrides.BodyMode)) {
		value := cfg.Overrides.BodyMode[key]
		switch strings.TrimSpace(value) {
		case model.BodyModeSimpleJSON, model.BodyModeFileOrData, model.BodyModeFlags:
		default:
			return fmt.Errorf("overrides.body_mode[%s] must be one of simple-json, file-or-data, or flags, got %q", key, value)
		}
	}
	return nil
}
