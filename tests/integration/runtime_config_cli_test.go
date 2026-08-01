package integration_test

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
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
			prependLegacyRuntimeVersion(t, dir)
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

func TestGeneratedGoOAuth2ClientCredentials(t *testing.T) {
	const (
		clientID     = "fssc-opencli"
		clientSecret = "sealed-client-secret"
		accessToken  = "issued-access-token"
	)
	var tokenRequests atomic.Int32
	var businessRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			tokenRequests.Add(1)
			if got := r.Header.Get("Authorization"); got != "" {
				t.Errorf("token request Authorization = %q, want empty", got)
			}
			if got := r.URL.Query().Get("client_id"); got != clientID {
				t.Errorf("client_id = %q, want %q", got, clientID)
			}
			if got := r.URL.Query().Get("client_secret"); got != clientSecret {
				t.Errorf("client_secret = %q, want sealed credential", got)
			}
			if got := r.URL.Query().Get("grant_type"); got != "client_credentials" {
				t.Errorf("grant_type = %q, want client_credentials", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"` + accessToken + `","token_type":"bearer","expires_in":3600}`))
		case "/items":
			businessRequests.Add(1)
			if got := r.Header.Get("Authorization"); got != "Bearer "+accessToken {
				t.Errorf("business Authorization = %q, want Bearer token", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"message":"ok"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	specPath := filepath.Join(t.TempDir(), "oauth.yaml")
	spec := fmt.Sprintf(`
openapi: 3.0.3
info: {title: OAuth API, version: 1.0.0}
paths:
  /oauth/token:
    post:
      tags: [auth]
      operationId: getToken
      security: []
      parameters:
        - {name: client_id, in: query, required: true, schema: {type: string}}
        - {name: client_secret, in: query, required: true, schema: {type: string}}
        - {name: grant_type, in: query, required: true, schema: {type: string}}
      responses:
        "200": {description: ok}
  /items:
    get:
      tags: [items]
      operationId: listItems
      security:
        - vendorOAuth: []
      responses:
        "200": {description: ok}
components:
  securitySchemes:
    vendorOAuth:
      type: oauth2
      flows:
        clientCredentials:
          tokenUrl: %s/oauth/token
          scopes: {}
`, server.URL)
	if err := os.WriteFile(specPath, []byte(spec), 0o600); err != nil {
		t.Fatalf("write OAuth spec: %v", err)
	}
	runtimeSource := filepath.Join(t.TempDir(), "runtime.yaml")
	if err := os.WriteFile(runtimeSource, []byte(fmt.Sprintf("base_url: %s\nauth:\n  type: oauth2\n  client_id: %s\n", server.URL, clientID)), 0o600); err != nil {
		t.Fatalf("write OAuth runtime source: %v", err)
	}
	t.Setenv("OPENCLI_OAUTH_CLIENT_SECRET", clientSecret)
	dir := t.TempDir()
	if err := app.RunGenerate(app.GenerateOptions{
		Input:             specPath,
		Output:            dir,
		Module:            "github.com/acme/generated",
		AppName:           "oauthcli",
		Auth:              "oauth2",
		Target:            "go",
		RuntimeConfigPath: runtimeSource,
	}); err != nil {
		t.Fatalf("generate OAuth CLI: %v", err)
	}

	cmd := exec.Command("go", "run", "./cmd/oauthcli", "items", "list")
	cmd.Dir = dir
	cmd.Env = append(runtimeTestEnv(dir), "OPENCLI_CONFIG="+filepath.Join(dir, "config", "runtime.yaml"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run generated OAuth CLI: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "ok") {
		t.Fatalf("generated OAuth CLI output = %s", out)
	}
	if tokenRequests.Load() != 1 || businessRequests.Load() != 1 {
		t.Fatalf("requests: token=%d business=%d, want 1 each", tokenRequests.Load(), businessRequests.Load())
	}
}

func TestGeneratedGoOAuth2AuthorizationCode(t *testing.T) {
	const (
		clientID    = "business-cli"
		accessToken = "business-access-token"
	)
	var loginRequests atomic.Int32
	var businessRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/idp/oauth2/authorize":
			if got := r.URL.Query().Get("client_id"); got != clientID {
				t.Errorf("authorize client_id = %q, want %q", got, clientID)
			}
			if got := r.URL.Query().Get("response_type"); got != "code" {
				t.Errorf("response_type = %q, want code", got)
			}
			redirectURI := r.URL.Query().Get("redirect_uri")
			state := r.URL.Query().Get("state")
			http.Redirect(w, r, redirectURI+"?code=iam-code&state="+url.QueryEscape(state), http.StatusFound)
		case "/cli-auth/login":
			loginRequests.Add(1)
			if err := r.ParseForm(); err != nil {
				t.Errorf("parse login form: %v", err)
			}
			if got := r.Form.Get("grant_type"); got != "authorization_code" {
				t.Errorf("grant_type = %q, want authorization_code", got)
			}
			if got := r.Form.Get("code"); got != "iam-code" {
				t.Errorf("code = %q, want iam-code", got)
			}
			if got := r.Form.Get("client_id"); got != clientID {
				t.Errorf("login client_id = %q, want %q", got, clientID)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"` + accessToken + `","token_type":"bearer","expires_in":3600}`))
		case "/items":
			businessRequests.Add(1)
			if got := r.Header.Get("Authorization"); got != "Bearer "+accessToken {
				t.Errorf("business Authorization = %q, want Bearer token", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"message":"ok"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	specPath := filepath.Join(t.TempDir(), "oauth.yaml")
	spec := fmt.Sprintf(`
openapi: 3.0.3
info: {title: OAuth API, version: 1.0.0}
paths:
  /cli-auth/login:
    post:
      tags: [auth]
      operationId: login
      security: []
      responses:
        "200": {description: ok}
  /items:
    get:
      tags: [items]
      operationId: listItems
      security:
        - userOAuth: []
      responses:
        "200": {description: ok}
components:
  securitySchemes:
    userOAuth:
      type: oauth2
      flows:
        authorizationCode:
          authorizationUrl: %s/idp/oauth2/authorize
          tokenUrl: %s/cli-auth/login
          scopes: {}
`, server.URL, server.URL)
	if err := os.WriteFile(specPath, []byte(spec), 0o600); err != nil {
		t.Fatalf("write OAuth spec: %v", err)
	}
	runtimeSource := filepath.Join(t.TempDir(), "runtime.yaml")
	runtimeYAML := fmt.Sprintf(`
base_url: %s
auth:
  type: oauth2
  grant_type: authorization_code
  client_id: %s
  authorization_url: %s/idp/oauth2/authorize
  token_url: %s/cli-auth/login
`, server.URL, clientID, server.URL, server.URL)
	if err := os.WriteFile(runtimeSource, []byte(runtimeYAML), 0o600); err != nil {
		t.Fatalf("write OAuth runtime source: %v", err)
	}
	dir := t.TempDir()
	if err := app.RunGenerate(app.GenerateOptions{
		Input:             specPath,
		Output:            dir,
		Module:            "github.com/acme/generated",
		AppName:           "oauthcli",
		Auth:              "oauth2",
		Target:            "go",
		RuntimeConfigPath: runtimeSource,
	}); err != nil {
		t.Fatalf("generate OAuth CLI: %v", err)
	}

	tokenFile := filepath.Join(t.TempDir(), "oauth-token.json")
	login := exec.Command("go", "run", "./cmd/oauthcli", "auth", "login", "--no-browser")
	login.Dir = dir
	login.Env = append(runtimeTestEnv(dir),
		"OPENCLI_CONFIG="+filepath.Join(dir, "config", "runtime.yaml"),
		"OPENCLI_OAUTH_TOKEN_FILE="+tokenFile,
	)
	stdout, err := login.StdoutPipe()
	if err != nil {
		t.Fatalf("login stdout: %v", err)
	}
	var stderr bytes.Buffer
	login.Stderr = &stderr
	if err := login.Start(); err != nil {
		t.Fatalf("start login: %v", err)
	}
	reader := bufio.NewReader(stdout)
	authorizeURL, readErr := reader.ReadString('\n')
	if readErr != nil {
		_ = login.Wait()
		t.Fatalf("read authorize URL: %v; stderr=%s", readErr, stderr.String())
	}
	authorizeURL = strings.TrimSpace(authorizeURL)
	browserResponse, err := http.Get(authorizeURL)
	if err != nil {
		_ = login.Process.Kill()
		t.Fatalf("complete browser login: %v", err)
	}
	_ = browserResponse.Body.Close()
	if err := login.Wait(); err != nil {
		t.Fatalf("login failed: %v; stderr=%s", err, stderr.String())
	}
	if info, err := os.Stat(tokenFile); err != nil {
		t.Fatalf("stat OAuth token file: %v", err)
	} else if info.Mode().Perm() != 0o600 {
		t.Fatalf("OAuth token file mode = %o, want 600", info.Mode().Perm())
	}

	call := exec.Command("go", "run", "./cmd/oauthcli", "items", "list")
	call.Dir = dir
	call.Env = append(runtimeTestEnv(dir),
		"OPENCLI_CONFIG="+filepath.Join(dir, "config", "runtime.yaml"),
		"OPENCLI_OAUTH_TOKEN_FILE="+tokenFile,
	)
	out, err := call.CombinedOutput()
	if err != nil {
		t.Fatalf("run generated OAuth CLI: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "ok") {
		t.Fatalf("generated OAuth CLI output = %s", out)
	}
	if loginRequests.Load() != 1 || businessRequests.Load() != 1 {
		t.Fatalf("requests: login=%d business=%d, want 1 each", loginRequests.Load(), businessRequests.Load())
	}
}

func TestOAuth2AuthorizationCodeRejectsRustTarget(t *testing.T) {
	runtimeSource := filepath.Join(t.TempDir(), "runtime.yaml")
	if err := os.WriteFile(runtimeSource, []byte(`
base_url: https://business-api.example.com
auth:
  type: oauth2
  grant_type: authorization_code
  client_id: business-cli
  authorization_url: https://iam.example.com/idp/oauth2/authorize
  token_url: https://business.example.com/cli-auth/login
`), 0o600); err != nil {
		t.Fatalf("write OAuth runtime source: %v", err)
	}
	err := app.RunGenerate(app.GenerateOptions{
		Input:             filepath.Join("..", "..", "examples", "petstore.yaml"),
		Output:            t.TempDir(),
		Module:            "github.com/acme/generated",
		AppName:           "oauthcli",
		Auth:              "oauth2",
		Target:            "rust",
		RuntimeConfigPath: runtimeSource,
	})
	if err == nil || !strings.Contains(err.Error(), "authorization_code currently supports target go") {
		t.Fatalf("error = %v, want Go-only authorization_code error", err)
	}
}

func TestGeneratedGoRequiredCredentialDoesNotLoadRuntimeConfigFromCWD(t *testing.T) {
	tests := []struct {
		name         string
		authMode     string
		fileAuthType string
		header       string
		credential   string
		wantError    string
	}{
		{
			name:         "bearer",
			authMode:     "token",
			fileAuthType: "bearer",
			credential:   "sealed-token",
			wantError:    "missing bearer token: set OPENCLI_AUTH_TOKEN or configure runtime auth",
		},
		{
			name:         "api key",
			authMode:     "api_key",
			fileAuthType: "api_key",
			header:       "X-API-Key",
			credential:   "sealed-api-key",
			wantError:    "missing API key: set OPENCLI_API_KEY or configure runtime auth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := generateRuntimeConfigCLI(t, "go", tt.authMode, tt.fileAuthType, tt.header, "http://127.0.0.1:1", tt.credential)
			cmd := exec.Command("go", "run", "./cmd/petcli", "pet", "list", "--limit", "10")
			cmd.Dir = dir
			cmd.Env = append(runtimeTestEnv(dir),
				"OPENCLI_BASE_URL=http://127.0.0.1:1",
				"XDG_CONFIG_HOME="+t.TempDir(),
			)
			out, err := cmd.CombinedOutput()
			if err == nil || !strings.Contains(string(out), tt.wantError) {
				t.Fatalf("error = %v output = %s, want %q", err, out, tt.wantError)
			}
			if strings.Contains(string(out), tt.credential) {
				t.Fatalf("error output leaked credential %q: %s", tt.credential, out)
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

func TestGeneratedRustBaseURLConfigUsesRuntimeToken(t *testing.T) {
	if _, err := exec.LookPath("cargo"); err != nil {
		t.Skip("cargo not installed")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer runtime-token" {
			t.Errorf("Authorization = %q, want runtime token", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}))
	defer server.Close()

	sourcePath := filepath.Join(t.TempDir(), "runtime.yaml")
	if err := os.WriteFile(sourcePath, []byte("base_url: "+server.URL+"\n"), 0o600); err != nil {
		t.Fatalf("write runtime source: %v", err)
	}
	t.Setenv("OPENCLI_AUTH_TOKEN", "")
	dir := t.TempDir()
	if err := app.RunGenerate(app.GenerateOptions{
		Input:             filepath.Join("..", "..", "examples", "petstore.yaml"),
		Output:            dir,
		Module:            "github.com/acme/generated",
		AppName:           "petcli",
		Auth:              "token",
		Target:            "rust",
		RuntimeConfigPath: sourcePath,
	}); err != nil {
		t.Fatalf("generate Rust CLI: %v", err)
	}

	cmd := exec.Command("cargo", "run", "--", "pet", "list", "--limit", "10")
	cmd.Dir = dir
	cmd.Env = append(runtimeTestEnv(dir),
		"OPENCLI_CONFIG="+filepath.Join(dir, "config", "runtime.yaml"),
		"OPENCLI_AUTH_TOKEN=runtime-token",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		output := string(out)
		if strings.Contains(output, "Could not resolve host") || strings.Contains(output, "failed to download from") {
			t.Skipf("cargo run skipped due to dependency network restriction:\n%s", output)
		}
		t.Fatalf("run generated Rust CLI: %v\n%s", err, output)
	}
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
	prependLegacyRuntimeVersion(t, dir)
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
	source := fmt.Sprintf("base_url: %s\n%s", baseURL, authMetadata)
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

func prependLegacyRuntimeVersion(t *testing.T, dir string) {
	t.Helper()
	path := filepath.Join(dir, "config", "runtime.yaml")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated runtime config: %v", err)
	}
	if err := os.WriteFile(path, append([]byte("version: v1\n"), content...), 0o600); err != nil {
		t.Fatalf("add legacy runtime version: %v", err)
	}
}

func runtimeTestEnv(dir string) []string {
	env := make([]string, 0, len(os.Environ())+2)
	for _, value := range os.Environ() {
		name, _, _ := strings.Cut(value, "=")
		switch name {
		case "OPENCLI_BASE_URL", "OPENCLI_AUTH_TOKEN", "OPENCLI_API_KEY", "OPENCLI_OAUTH_CLIENT_SECRET", "OPENCLI_CONFIG":
			continue
		}
		env = append(env, value)
	}
	return append(env,
		"GOTOOLCHAIN=local",
		"GOCACHE="+filepath.Join(dir, ".gocache"),
	)
}
