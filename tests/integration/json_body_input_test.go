package integration_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"one-cli/internal/app"
)

func TestGeneratedSimpleJSONCommandBuildsBodyAndRejectsConflicts(t *testing.T) {
	projectDir := t.TempDir()
	if err := app.RunGenerate(filepath.Join("..", "..", "examples", "openapi.json"), "", projectDir, "github.com/acme/generated", "openapi-cli", ""); err != nil {
		t.Fatalf("generate: %v", err)
	}

	received := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = true
		if r.URL.Path != "/nodus/api/v1/auth/login" {
			t.Fatalf("path = %q want %q", r.URL.Path, "/nodus/api/v1/auth/login")
		}
		payload, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if string(payload) != `{"email":"you@example.com","password":"secret","remember":true}` {
			t.Fatalf("body = %s", payload)
		}
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}))
	defer server.Close()

	env := append(os.Environ(),
		"GOCACHE="+filepath.Join(t.TempDir(), "gocache"),
		"GOTOOLCHAIN=local",
		"OPENCLI_BASE_URL="+server.URL,
	)

	cmd := exec.Command("go", "run", "./cmd/openapi-cli", "auth", "login", "--email", "you@example.com", "--password", "secret", "--remember")
	cmd.Dir = projectDir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("login command failed: %v output=%s", err, out)
	}
	if !received {
		t.Fatal("expected generated command to issue an HTTP request")
	}

	conflict := exec.Command("go", "run", "./cmd/openapi-cli", "auth", "login", "--email", "you@example.com", "--password", "secret", "--data", `{"email":"override"}`)
	conflict.Dir = projectDir
	conflict.Env = env
	conflictOut, conflictErr := conflict.CombinedOutput()
	if conflictErr == nil || !strings.Contains(string(conflictOut), "body input flags cannot be combined with --data or --file") {
		t.Fatalf("expected body conflict error, got err=%v output=%s", conflictErr, conflictOut)
	}
}
