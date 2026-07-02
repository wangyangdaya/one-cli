package render

import (
	"fmt"
	"strings"

	"one-cli/internal/model"
)

// Project writes the generated project for the requested target(s).
// Target normalization happens here so callers can pass raw --target values.
func Project(outputDir, module string, app model.App, targets ...string) error {
	if err := validateProjectInputs(outputDir, module, app); err != nil {
		return err
	}
	target := "go"
	if len(targets) > 0 {
		target = strings.TrimSpace(targets[0])
	}
	skillLang := "en"
	if len(targets) > 1 {
		skillLang = strings.TrimSpace(targets[1])
	}
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
		return writeGoProject(outputDir, module, app, skillLang)
	case "rust":
		return writeRustProject(outputDir, module, app, skillLang)
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
