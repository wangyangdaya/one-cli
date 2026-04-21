package render

import (
	"fmt"
	"strings"

	"one-cli/internal/model"
)

func Project(outputDir, module string, app model.App, targets ...string) error {
	target, err := normalizeRenderTarget(targets)
	if err != nil {
		return err
	}
	if err := validateProjectInputs(outputDir, module, app); err != nil {
		return err
	}

	switch target {
	case "go":
		return writeGoProject(outputDir, module, app)
	case "rust":
		return writeRustProject(outputDir, module, app)
	default:
		return fmt.Errorf("unsupported render target %q", target)
	}
}

func normalizeRenderTarget(targets []string) (string, error) {
	target := "go"
	if len(targets) > 0 {
		target = strings.TrimSpace(targets[0])
	}

	switch strings.ToLower(target) {
	case "", "go":
		return "go", nil
	case "rust":
		return "rust", nil
	default:
		return "", fmt.Errorf("unsupported render target %q", target)
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
