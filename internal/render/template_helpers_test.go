package render

import (
	"testing"

	"one-cli/internal/model"
)

func TestAppVersionFallbacksMatchAcrossTargets(t *testing.T) {
	app := model.App{}

	if got := goAppVersion(app); got != "0.1.0" {
		t.Fatalf("goAppVersion() = %q, want 0.1.0", got)
	}
	if got := rustAppVersion(app); got != "0.1.0" {
		t.Fatalf("rustAppVersion() = %q, want 0.1.0", got)
	}
}
