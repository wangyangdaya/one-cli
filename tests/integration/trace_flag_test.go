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

func TestGeneratedCLITraceFlagControlsHTTPLogging(t *testing.T) {
	projectDir := t.TempDir()
	if err := app.RunGenerate(filepath.Join("..", "..", "examples", "openapi.json"), "", projectDir, "github.com/acme/generated", "openapi-cli", ""); err != nil {
		t.Fatalf("generate: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}))
	defer server.Close()

	env := append(os.Environ(),
		"GOCACHE="+filepath.Join(t.TempDir(), "gocache"),
		"GOTOOLCHAIN=local",
		"OPENCLI_BASE_URL="+server.URL,
	)

	noTrace := exec.Command("go", "run", "./cmd/openapi-cli", "auth", "login", "--email", "you@example.com", "--password", "secret")
	noTrace.Dir = projectDir
	noTrace.Env = env
	noTraceOut, noTraceErr := noTrace.CombinedOutput()
	if noTraceErr != nil {
		t.Fatalf("no-trace command failed: %v output=%s", noTraceErr, noTraceOut)
	}
	if strings.Contains(string(noTraceOut), "[opencli][http]") {
		t.Fatalf("expected no trace logs without flag, got %s", noTraceOut)
	}

	withTrace := exec.Command("go", "run", "./cmd/openapi-cli", "--trace", "auth", "login", "--email", "you@example.com", "--password", "secret")
	withTrace.Dir = projectDir
	withTrace.Env = env
	withTraceOut, withTraceErr := withTrace.CombinedOutput()
	if withTraceErr != nil {
		t.Fatalf("trace command failed: %v output=%s", withTraceErr, withTraceOut)
	}
	text := string(withTraceOut)
	for _, want := range []string{"[opencli][http] request", "[opencli][http] response", `"password": "secret"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing trace fragment %q in %s", want, text)
		}
	}
}
