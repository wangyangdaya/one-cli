package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveInputSources(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "body.json")
	if err := os.WriteFile(path, []byte("  {\"from\":\"file\"}  \n"), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}

	tests := []struct {
		name  string
		raw   string
		stdin string
		want  string
	}{
		{name: "inline", raw: `{"from":"inline"}`, want: `{"from":"inline"}`},
		{name: "file", raw: "@" + path, want: `{"from":"file"}`},
		{name: "stdin", raw: "-", stdin: "  {\"from\":\"stdin\"} \n", want: `{"from":"stdin"}`},
		{name: "escaped at", raw: "@@literal", want: "@literal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveInput(tt.raw, strings.NewReader(tt.stdin))
			if err != nil {
				t.Fatalf("ResolveInput: %v", err)
			}
			if got != tt.want {
				t.Fatalf("resolved = %q want %q", got, tt.want)
			}
		})
	}
}

func TestResolveInputRejectsEmptySources(t *testing.T) {
	for _, raw := range []string{"@", "-"} {
		if _, err := ResolveInput(raw, strings.NewReader("")); err == nil {
			t.Fatalf("ResolveInput(%q) should fail", raw)
		}
	}
}
