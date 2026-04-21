package integration_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"one-cli/internal/app"
)

func TestGeneratedCLIHeaderFlagSendsRequestHeaders(t *testing.T) {
	dir := t.TempDir()
	if err := app.RunGenerate(filepath.Join("..", "..", "examples", "openapi.json"), "", dir, "github.com/acme/generated", "openapi-cli", ""); err != nil {
		t.Fatalf("run generate: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("expected Authorization header, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}))
	defer server.Close()

	cmd := exec.Command("go", "run", "./cmd/openapi-cli", "auth", "me", "--header", "Authorization: Bearer token")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"OPENCLI_BASE_URL="+server.URL,
		"GOTOOLCHAIN=local",
		"GOCACHE="+filepath.Join(dir, ".gocache"),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected auth me to succeed, got %v, output: %s", err, string(out))
	}
	if !strings.Contains(string(out), "ok") {
		t.Fatalf("expected success output, got %s", string(out))
	}
}

func TestGeneratedCLIHeaderFlagRejectsMalformedValues(t *testing.T) {
	dir := t.TempDir()
	if err := app.RunGenerate(filepath.Join("..", "..", "examples", "openapi.json"), "", dir, "github.com/acme/generated", "openapi-cli", ""); err != nil {
		t.Fatalf("run generate: %v", err)
	}

	cmd := exec.Command("go", "run", "./cmd/openapi-cli", "auth", "me", "--header", "Authorization")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"OPENCLI_BASE_URL=http://127.0.0.1:1",
		"GOTOOLCHAIN=local",
		"GOCACHE="+filepath.Join(dir, ".gocache"),
	)
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), `invalid --header value "Authorization": expected "Name: Value"`) {
		t.Fatalf("expected malformed header error, got err=%v output=%s", err, string(out))
	}
}
