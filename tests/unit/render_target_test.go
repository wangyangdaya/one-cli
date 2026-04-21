package unit_test

import (
	"strings"
	"testing"

	"one-cli/internal/model"
	"one-cli/internal/render"
)

func TestRenderProjectAcceptsExplicitGoTarget(t *testing.T) {
	dir := t.TempDir()
	app := model.App{
		Name: "one",
		Groups: []model.Group{
			{
				Name: "leave",
				Operations: []model.Operation{
					{CommandName: "list", Method: "GET", Path: "/leaves"},
				},
			},
		},
	}

	if err := render.Project(dir, "github.com/acme/one-cli", app, "go"); err != nil {
		t.Fatalf("render with go target: %v", err)
	}
}

func TestRenderProjectRejectsUnknownTarget(t *testing.T) {
	err := render.Project(t.TempDir(), "github.com/acme/one-cli", model.App{Name: "one"}, "java")
	if err == nil || !strings.Contains(err.Error(), "unsupported render target") {
		t.Fatalf("expected unsupported render target error, got %v", err)
	}
}
