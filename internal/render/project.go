package render

import (
	"fmt"
	"strings"

	"one-cli/internal/model"
	"one-cli/internal/runtimeconfig"
)

type ProjectOptions struct {
	Target        string
	SkillLang     string
	RuntimeBundle *runtimeconfig.Bundle
}

// Project writes the generated project for the requested target(s).
// Target normalization happens here so callers can pass raw --target values.
func Project(outputDir, module string, app model.App, targets ...string) error {
	options := ProjectOptions{}
	if len(targets) > 0 {
		options.Target = targets[0]
	}
	if len(targets) > 1 {
		options.SkillLang = targets[1]
	}
	return ProjectWithOptions(outputDir, module, app, options)
}

func ProjectWithOptions(outputDir, module string, app model.App, options ProjectOptions) error {
	if err := validateProjectInputs(outputDir, module, app); err != nil {
		return err
	}
	target := strings.TrimSpace(options.Target)
	if target == "" {
		target = "go"
	}
	skillLang := strings.TrimSpace(options.SkillLang)
	if skillLang == "" {
		skillLang = "en"
	}
	switch strings.ToLower(skillLang) {
	case "en", "zh":
		skillLang = strings.ToLower(skillLang)
	default:
		return fmt.Errorf("unsupported skill language %q: expected en or zh", skillLang)
	}
	switch strings.ToLower(target) {
	case "", "go":
		return writeGoProject(outputDir, module, app, skillLang, options.RuntimeBundle)
	case "rust":
		return writeRustProject(outputDir, module, app, skillLang, options.RuntimeBundle)
	default:
		return fmt.Errorf("unsupported target %q: expected go or rust", target)
	}
}

func validateProjectInputs(outputDir, module string, app model.App) error {
	if strings.TrimSpace(outputDir) == "" {
		return fmt.Errorf("missing output directory")
	}
	if strings.TrimSpace(module) == "" {
		return fmt.Errorf("missing module path")
	}
	if strings.TrimSpace(app.Name) == "" {
		return fmt.Errorf("missing app name")
	}
	return nil
}
