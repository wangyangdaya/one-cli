package integration_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"one-cli/internal/app"
)

func TestGeneratedGoRuntimeConfig(t *testing.T) {
	tests := []struct {
		name       string
		env        []string
		extraArgs  []string
		wantHeader string
	}{
		{name: "sealed file fallback", wantHeader: "Bearer file-token"},
		{name: "plaintext environment override", env: []string{"OPENCLI_AUTH_TOKEN=env-token"}, wantHeader: "Bearer env-token"},
		{
			name:       "explicit header override",
			env:        []string{"OPENCLI_AUTH_TOKEN=env-token"},
			extraArgs:  []string{"--header", "Authorization: Bearer explicit-token"},
			wantHeader: "Bearer explicit-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("Authorization"); got != tt.wantHeader {
					t.Errorf("Authorization = %q, want %q", got, tt.wantHeader)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"message":"ok"}`))
			}))
			defer server.Close()

			dir := generateRuntimeConfigCLI(t, "go", "token", "bearer", "", server.URL, "file-token")
			args := append([]string{"pet", "list", "--limit", "10"}, tt.extraArgs...)
			cmd := exec.Command("go", append([]string{"run", "./cmd/petcli"}, args...)...)
			cmd.Dir = dir
			cmd.Env = append(runtimeTestEnv(dir), "OPENCLI_CONFIG="+filepath.Join(dir, "config", "runtime.yaml"))
			cmd.Env = append(cmd.Env, tt.env...)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("run generated Go CLI: %v\n%s", err, out)
			}
			if !strings.Contains(string(out), "ok") {
				t.Fatalf("generated Go CLI output = %s", out)
			}
		})
	}
}

func TestGeneratedGoAPIKeyRuntimeConfig(t *testing.T) {
	tests := []struct {
		name       string
		env        []string
		extraArgs  []string
		wantHeader string
	}{
		{name: "sealed file fallback", wantHeader: "file-api-key"},
		{name: "plaintext environment override", env: []string{"OPENCLI_API_KEY=env-api-key"}, wantHeader: "env-api-key"},
		{
			name:       "explicit header override",
			env:        []string{"OPENCLI_API_KEY=env-api-key"},
			extraArgs:  []string{"--header", "X-API-Key: explicit-api-key"},
			wantHeader: "explicit-api-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("X-API-Key"); got != tt.wantHeader {
					t.Errorf("X-API-Key = %q, want %q", got, tt.wantHeader)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"message":"ok"}`))
			}))
			defer server.Close()

			dir := generateRuntimeConfigCLI(t, "go", "api_key", "api_key", "X-API-Key", server.URL, "file-api-key")
			args := append([]string{"pet", "list", "--limit", "10"}, tt.extraArgs...)
			cmd := exec.Command("go", append([]string{"run", "./cmd/petcli"}, args...)...)
			cmd.Dir = dir
			cmd.Env = append(runtimeTestEnv(dir), "OPENCLI_CONFIG="+filepath.Join(dir, "config", "runtime.yaml"))
			cmd.Env = append(cmd.Env, tt.env...)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("run generated Go CLI: %v\n%s", err, out)
			}
		})
	}
}

func TestGeneratedRustRuntimeConfig(t *testing.T) {
	testGeneratedRustRuntimeConfig(t, "token", "bearer", "", "Authorization", "file-token", []struct {
		name       string
		env        []string
		extraArgs  []string
		wantHeader string
	}{
		{name: "sealed file fallback", wantHeader: "Bearer file-token"},
		{name: "plaintext environment override", env: []string{"OPENCLI_AUTH_TOKEN=env-token"}, wantHeader: "Bearer env-token"},
		{name: "explicit header override", env: []string{"OPENCLI_AUTH_TOKEN=env-token"}, extraArgs: []string{"--header", "Authorization: Bearer explicit-token"}, wantHeader: "Bearer explicit-token"},
	})
}

func TestGeneratedRustAPIKeyRuntimeConfig(t *testing.T) {
	testGeneratedRustRuntimeConfig(t, "api_key", "api_key", "X-API-Key", "X-API-Key", "file-api-key", []struct {
		name       string
		env        []string
		extraArgs  []string
		wantHeader string
	}{
		{name: "sealed file fallback", wantHeader: "file-api-key"},
		{name: "plaintext environment override", env: []string{"OPENCLI_API_KEY=env-api-key"}, wantHeader: "env-api-key"},
		{name: "explicit header override", env: []string{"OPENCLI_API_KEY=env-api-key"}, extraArgs: []string{"--header", "X-API-Key: explicit-api-key"}, wantHeader: "explicit-api-key"},
	})
}

func TestPackagedRustRuntimeConfigLoadsRelativeToExecutable(t *testing.T) {
	if _, err := exec.LookPath("cargo"); err != nil {
		t.Skip("cargo not installed")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer packaged-rust-token" {
			t.Errorf("Authorization = %q, want packaged Rust token", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}))
	defer server.Close()

	project := generateRuntimeConfigCLI(t, "rust", "token", "bearer", "", server.URL, "packaged-rust-token")
	output := filepath.Join(t.TempDir(), "pet-skills")
	packageCommand := app.NewRootCommand()
	packageCommand.SetArgs([]string{"package", "--project", project, "--output", output})
	if err := packageCommand.Execute(); err != nil {
		message := err.Error()
		if strings.Contains(message, "Could not resolve host") || strings.Contains(message, "failed to download from") {
			t.Skipf("cargo build skipped due to dependency network restriction: %v", err)
		}
		t.Fatalf("package generated Rust project: %v", err)
	}

	run := exec.Command(filepath.Join(output, "bin", "petcli"), "pet", "list", "--limit", "10")
	run.Dir = t.TempDir()
	run.Env = runtimeTestEnv(run.Dir)
	result, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("run packaged Rust CLI outside bundle directory: %v\n%s", err, result)
	}
	if !strings.Contains(string(result), "ok") {
		t.Fatalf("packaged Rust CLI output = %s", result)
	}
}

func testGeneratedRustRuntimeConfig(t *testing.T, authMode, fileAuthType, configHeader, observedHeader, fileCredential string, tests []struct {
	name       string
	env        []string
	extraArgs  []string
	wantHeader string
}) {
	t.Helper()
	if _, err := exec.LookPath("cargo"); err != nil {
		t.Skip("cargo not installed")
	}

	wantHeader := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get(observedHeader); got != wantHeader {
			t.Errorf("%s = %q, want %q", observedHeader, got, wantHeader)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}))
	defer server.Close()

	dir := generateRuntimeConfigCLI(t, "rust", authMode, fileAuthType, configHeader, server.URL, fileCredential)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantHeader = tt.wantHeader
			args := append([]string{"run", "--"}, tt.extraArgs...)
			args = append(args, "pet", "list", "--limit", "10")
			cmd := exec.Command("cargo", args...)
			cmd.Dir = dir
			cmd.Env = append(runtimeTestEnv(dir), "OPENCLI_CONFIG="+filepath.Join(dir, "config", "runtime.yaml"))
			cmd.Env = append(cmd.Env, tt.env...)
			out, err := cmd.CombinedOutput()
			if err != nil {
				output := string(out)
				if strings.Contains(output, "Could not resolve host") || strings.Contains(output, "failed to download from") {
					t.Skipf("cargo run skipped due to dependency network restriction:\n%s", output)
				}
				t.Fatalf("run generated Rust CLI: %v\n%s", err, output)
			}
		})
	}
}

func generateRuntimeConfigCLI(t *testing.T, target, authMode, fileAuthType, header, baseURL, credential string) string {
	t.Helper()
	dir := t.TempDir()
	sourcePath := filepath.Join(t.TempDir(), "runtime.yaml")
	authMetadata := fmt.Sprintf("auth:\n  type: %s\n", fileAuthType)
	if header != "" {
		authMetadata += "  header: " + header + "\n"
	}
	source := fmt.Sprintf("version: v1\nbase_url: %s\n%s", baseURL, authMetadata)
	if err := os.WriteFile(sourcePath, []byte(source), 0o600); err != nil {
		t.Fatalf("write runtime source: %v", err)
	}
	envName := "OPENCLI_AUTH_TOKEN"
	if authMode == "api_key" {
		envName = "OPENCLI_API_KEY"
	}
	t.Setenv(envName, credential)
	if err := app.RunGenerate(app.GenerateOptions{
		Input:             filepath.Join("..", "..", "examples", "petstore.yaml"),
		Output:            dir,
		Module:            "github.com/acme/generated",
		AppName:           "petcli",
		Auth:              authMode,
		Target:            target,
		RuntimeConfigPath: sourcePath,
	}); err != nil {
		t.Fatalf("generate %s runtime-config CLI: %v", target, err)
	}
	return dir
}

func runtimeTestEnv(dir string) []string {
	env := make([]string, 0, len(os.Environ())+2)
	for _, value := range os.Environ() {
		name, _, _ := strings.Cut(value, "=")
		switch name {
		case "OPENCLI_BASE_URL", "OPENCLI_AUTH_TOKEN", "OPENCLI_API_KEY", "OPENCLI_CONFIG":
			continue
		}
		env = append(env, value)
	}
	return append(env,
		"GOTOOLCHAIN=local",
		"GOCACHE="+filepath.Join(dir, ".gocache"),
	)
}
