package integration_test

import (
	"encoding/json"
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
	if err := app.RunGenerate(app.GenerateOptions{
		Input:   filepath.Join("..", "..", "examples", "openapi.json"),
		Output:  dir,
		Module:  "github.com/acme/generated",
		AppName: "openapi-cli",
	}); err != nil {
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

func TestGeneratedCLIAuthTokenEnvSendsBearerHeader(t *testing.T) {
	dir := t.TempDir()
	if err := app.RunGenerate(app.GenerateOptions{
		Input:   filepath.Join("..", "..", "examples", "petstore.yaml"),
		Output:  dir,
		Module:  "github.com/acme/generated",
		AppName: "petcli",
	}); err != nil {
		t.Fatalf("run generate: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer env-token" {
			t.Fatalf("expected Authorization header from OPENCLI_AUTH_TOKEN, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}))
	defer server.Close()

	cmd := exec.Command("go", "run", "./cmd/petcli", "pet", "list", "--limit", "10")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"OPENCLI_BASE_URL="+server.URL,
		"OPENCLI_AUTH_TOKEN=env-token",
		"GOTOOLCHAIN=local",
		"GOCACHE="+filepath.Join(dir, ".gocache"),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected pet listPets to succeed, got %v, output: %s", err, string(out))
	}
	if !strings.Contains(string(out), "ok") {
		t.Fatalf("expected success output, got %s", string(out))
	}
}

func TestGeneratedCLIJSONFlagWrapsResponse(t *testing.T) {
	dir := t.TempDir()
	if err := app.RunGenerate(app.GenerateOptions{
		Input:   filepath.Join("..", "..", "examples", "openapi.json"),
		Output:  dir,
		Module:  "github.com/acme/generated",
		AppName: "openapi-cli",
	}); err != nil {
		t.Fatalf("run generate: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"ok","user":{"id":"u1"}}`))
	}))
	defer server.Close()

	cmd := exec.Command("go", "run", "./cmd/openapi-cli", "--json", "auth", "me")
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

	var envelope struct {
		OK      bool   `json:"ok"`
		Command string `json:"command"`
		Message string `json:"message"`
		Data    struct {
			Message string         `json:"message"`
			Raw     map[string]any `json:"raw"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &envelope); err != nil {
		t.Fatalf("expected JSON envelope, got error %v output=%s", err, out)
	}
	if !envelope.OK || envelope.Command != "openapi-cli auth me" || envelope.Data.Message != "ok" || envelope.Data.Raw["message"] != "ok" {
		t.Fatalf("unexpected JSON envelope: %+v", envelope)
	}
}

func TestGeneratedCLIHeaderFlagRejectsMalformedValues(t *testing.T) {
	dir := t.TempDir()
	if err := app.RunGenerate(app.GenerateOptions{
		Input:   filepath.Join("..", "..", "examples", "openapi.json"),
		Output:  dir,
		Module:  "github.com/acme/generated",
		AppName: "openapi-cli",
	}); err != nil {
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
