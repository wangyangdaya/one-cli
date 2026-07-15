package configgen

import (
	"strings"
	"testing"
)

func TestLoadBytesRejectsInvalidAuthType(t *testing.T) {
	_, err := LoadBytes([]byte("auth:\n  type: bearer\n"))
	if err == nil || !strings.Contains(err.Error(), "auth.type") {
		t.Fatalf("expected auth.type validation error, got %v", err)
	}
}

func TestLoadBytesRejectsInvalidBodyMode(t *testing.T) {
	_, err := LoadBytes([]byte("overrides:\n  body_mode:\n    pet.create: json\n"))
	if err == nil || !strings.Contains(err.Error(), "overrides.body_mode[pet.create]") {
		t.Fatalf("expected body mode validation error, got %v", err)
	}
}

func TestLoadBytesAcceptsSupportedValues(t *testing.T) {
	_, err := LoadBytes([]byte("auth:\n  type: ak_sk\noverrides:\n  body_mode:\n    pet.create: flags\n"))
	if err != nil {
		t.Fatalf("expected supported values to load: %v", err)
	}
}
