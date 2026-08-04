package integration_test

import (
	"bufio"
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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
	for _, customExchange := range []bool{false, true} {
		name := "standard form"
		if customExchange {
			name = "custom JSON"
		}
		t.Run(name, func(t *testing.T) {
			testGeneratedGoOAuth2AuthorizationCode(t, customExchange)
		})
	}
}

func TestGeneratedGoOAuth2AuthorizationCodeWithPKCEAndOIDC(t *testing.T) {
	testGeneratedOAuth2AuthorizationCodeWithPKCEAndOIDC(t, "go")
}

func TestGeneratedRustOAuth2AuthorizationCodeWithPKCEAndOIDC(t *testing.T) {
	if _, err := exec.LookPath("cargo"); err != nil {
		t.Skip("cargo not installed")
	}
	testGeneratedOAuth2AuthorizationCodeWithPKCEAndOIDC(t, "rust")
}

func testGeneratedOAuth2AuthorizationCodeWithPKCEAndOIDC(t *testing.T, target string) {
	t.Helper()
	const clientID = "oidc-cli"
	callbackListener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve callback port: %v", err)
	}
	redirectURI := "http://" + callbackListener.Addr().String() + "/oauth/callback"
	_ = callbackListener.Close()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate OIDC signing key: %v", err)
	}
	invalidSigningKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate invalid OIDC signing key: %v", err)
	}

	var issuer string
	var expectedChallenge string
	var expectedNonce string
	var tokenCase atomic.Value
	tokenCase.Store("valid")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/authorize":
			if got := r.URL.Query().Get("code_challenge_method"); got != "S256" {
				t.Errorf("code_challenge_method = %q, want S256", got)
			}
			expectedChallenge = r.URL.Query().Get("code_challenge")
			expectedNonce = r.URL.Query().Get("nonce")
			if expectedChallenge == "" || expectedNonce == "" {
				t.Errorf("authorize query is missing PKCE/OIDC values: %v", r.URL.Query())
			}
			state := r.URL.Query().Get("state")
			http.Redirect(w, r, redirectURI+"?code=oidc-code&state="+url.QueryEscape(state), http.StatusFound)
		case "/token":
			if err := r.ParseForm(); err != nil {
				t.Errorf("parse token form: %v", err)
			}
			verifier := r.Form.Get("code_verifier")
			challenge := sha256.Sum256([]byte(verifier))
			if got := base64.RawURLEncoding.EncodeToString(challenge[:]); verifier == "" || got != expectedChallenge {
				t.Errorf("PKCE verifier does not match challenge")
			}
			mode := tokenCase.Load().(string)
			if mode == "missing_id_token" {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"access_token":"access","refresh_token":"refresh","token_type":"Bearer","expires_in":3600,"refresh_token_expires_in":86400}`))
				return
			}
			now := time.Now().Unix()
			claims := map[string]any{"iss": issuer, "aud": clientID, "exp": now + 300, "iat": now, "nonce": expectedNonce}
			signingKey := privateKey
			switch mode {
			case "invalid_signature":
				signingKey = invalidSigningKey
			case "issuer_mismatch":
				claims["iss"] = issuer + "/other"
			case "audience_mismatch":
				claims["aud"] = "another-client"
			case "invalid_azp":
				claims["aud"] = []string{clientID, "another-client"}
				claims["azp"] = "another-client"
			case "expired":
				claims["exp"] = now - 120
			case "future_iat":
				claims["iat"] = now + 120
			case "nonce_mismatch":
				claims["nonce"] = expectedNonce + "-other"
			}
			idToken := signOIDCTestToken(t, signingKey, claims)
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"access_token":"access","refresh_token":"refresh","token_type":"Bearer","expires_in":3600,"refresh_token_expires_in":86400,"id_token":%q}`, idToken)
		case "/.well-known/openid-configuration":
			jwksURI := issuer + "/jwks"
			if tokenCase.Load().(string) == "unsafe_jwks_uri" {
				jwksURI = "http://example.com/jwks"
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"issuer":%q,"jwks_uri":%q}`, issuer, jwksURI)
		case "/jwks":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"keys":[{"kid":"test-key","kty":"RSA","use":"sig","alg":"RS256","n":%q,"e":%q}]}`,
				base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
				base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.PublicKey.E)).Bytes()))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	issuer = server.URL

	specPath := filepath.Join(t.TempDir(), "oidc.yaml")
	spec := fmt.Sprintf(`
openapi: 3.0.3
info: {title: OIDC API, version: 1.0.0}
paths:
  /items:
    get:
      tags: [items]
      operationId: listItems
      security: [{userOAuth: []}]
      responses: {"200": {description: ok}}
components:
  securitySchemes:
    userOAuth:
      type: oauth2
      flows:
        authorizationCode:
          authorizationUrl: %s/authorize
          tokenUrl: %s/token
          scopes: {openid: OIDC}
`, issuer, issuer)
	if err := os.WriteFile(specPath, []byte(spec), 0o600); err != nil {
		t.Fatalf("write OIDC spec: %v", err)
	}
	runtimeSource := filepath.Join(t.TempDir(), "runtime.yaml")
	runtimeYAML := fmt.Sprintf(`
base_url: %s
auth:
  type: oauth2
  grant_type: authorization_code
  client_id: %s
  authorization_url: %s/authorize
  token_url: %s/token
  redirect_uri: %s
  scopes: [openid]
  pkce: {enabled: true}
  oidc: {enabled: true, issuer: %s}
`, issuer, clientID, issuer, issuer, redirectURI, issuer)
	if err := os.WriteFile(runtimeSource, []byte(runtimeYAML), 0o600); err != nil {
		t.Fatalf("write OIDC runtime config: %v", err)
	}
	dir := t.TempDir()
	if err := app.RunGenerate(app.GenerateOptions{
		Input: specPath, Output: dir, Module: "github.com/acme/oidccli", AppName: "oidccli",
		Auth: "oauth2", Target: target, RuntimeConfigPath: runtimeSource,
	}); err != nil {
		t.Fatalf("generate OIDC CLI: %v", err)
	}
	tokenFile := filepath.Join(t.TempDir(), "oauth-token.json")
	for _, mode := range []string{
		"valid",
		"missing_id_token",
		"invalid_signature",
		"issuer_mismatch",
		"audience_mismatch",
		"invalid_azp",
		"expired",
		"future_iat",
		"nonce_mismatch",
		"unsafe_jwks_uri",
	} {
		t.Run(mode, func(t *testing.T) {
			tokenCase.Store(mode)
			_ = os.Remove(tokenFile)
			login := exec.Command("go", "run", "./cmd/oidccli", "login", "--no-browser")
			if target == "rust" {
				login = exec.Command("cargo", "run", "--quiet", "--", "login", "--no-browser")
			}
			login.Dir = dir
			login.Env = append(runtimeTestEnv(dir), "OPENCLI_CONFIG="+filepath.Join(dir, "config", "runtime.yaml"), "OPENCLI_OAUTH_TOKEN_FILE="+tokenFile)
			stdout, err := login.StdoutPipe()
			if err != nil {
				t.Fatalf("login stdout: %v", err)
			}
			var stderr bytes.Buffer
			login.Stderr = &stderr
			if err := login.Start(); err != nil {
				t.Fatalf("start login: %v", err)
			}
			authorizeURL, err := bufio.NewReader(stdout).ReadString('\n')
			if err != nil {
				_ = login.Process.Kill()
				t.Fatalf("read authorize URL: %v", err)
			}
			response, err := http.Get(strings.TrimSpace(authorizeURL))
			if err != nil {
				_ = login.Process.Kill()
				t.Fatalf("complete OIDC login: %v", err)
			}
			_ = response.Body.Close()
			loginErr := login.Wait()
			if mode == "valid" {
				if loginErr != nil {
					t.Fatalf("OIDC login failed: %v; stderr=%s", loginErr, stderr.String())
				}
				if raw, err := os.ReadFile(tokenFile); err != nil || !bytes.Contains(raw, []byte(`"access_token":"access"`)) {
					t.Fatalf("OIDC token was not stored: %s, err=%v", raw, err)
				}
				return
			}
			if loginErr == nil {
				t.Fatalf("OIDC %s unexpectedly succeeded", mode)
			}
			if _, err := os.Stat(tokenFile); !os.IsNotExist(err) {
				t.Fatalf("OIDC %s persisted OAuth credentials: %v", mode, err)
			}
		})
	}

	unsafeRuntime := strings.Replace(runtimeYAML, redirectURI, "http://0.0.0.0:18081/oauth/callback", 1)
	if err := os.WriteFile(filepath.Join(dir, "config", "runtime.yaml"), []byte(unsafeRuntime), 0o600); err != nil {
		t.Fatalf("write unsafe OIDC runtime config: %v", err)
	}
	unsafeLogin := exec.Command("go", "run", "./cmd/oidccli", "login", "--no-browser")
	if target == "rust" {
		unsafeLogin = exec.Command("cargo", "run", "--quiet", "--", "login", "--no-browser")
	}
	unsafeLogin.Dir = dir
	unsafeLogin.Env = append(runtimeTestEnv(dir), "OPENCLI_CONFIG="+filepath.Join(dir, "config", "runtime.yaml"), "OPENCLI_OAUTH_TOKEN_FILE="+tokenFile)
	unsafeOutput, unsafeErr := unsafeLogin.CombinedOutput()
	if unsafeErr == nil || !strings.Contains(string(unsafeOutput), "invalid OAuth redirect_uri") {
		t.Fatalf("unsafe runtime redirect result = %s, err=%v", unsafeOutput, unsafeErr)
	}
}

func signOIDCTestToken(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()
	header, _ := json.Marshal(map[string]any{"alg": "RS256", "kid": "test-key", "typ": "JWT"})
	claimsJSON, _ := json.Marshal(claims)
	encodedHeader := base64.RawURLEncoding.EncodeToString(header)
	encodedClaims := base64.RawURLEncoding.EncodeToString(claimsJSON)
	signingInput := encodedHeader + "." + encodedClaims
	digest := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("sign OIDC test token: %v", err)
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func testGeneratedGoOAuth2AuthorizationCode(t *testing.T, customExchange bool) {
	const (
		clientID             = "business-cli"
		accessToken          = "business-access-token"
		refreshToken         = "business-refresh-token"
		refreshedAccessToken = "refreshed-business-access-token"
		refreshedRefresh     = "refreshed-business-refresh-token"
	)
	callbackListener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve callback port: %v", err)
	}
	redirectURI := "http://" + callbackListener.Addr().String() + "/oauth/callback"
	_ = callbackListener.Close()
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
			if got := r.URL.Query().Get("redirect_uri"); got != redirectURI {
				t.Errorf("redirect_uri = %q, want %q", got, redirectURI)
			}
			state := r.URL.Query().Get("state")
			http.Redirect(w, r, redirectURI+"?code=iam-code&state="+url.QueryEscape(state), http.StatusFound)
		case "/cli-auth/login":
			loginRequests.Add(1)
			if customExchange {
				var request struct {
					Code  string `json:"authorizationCode"`
					State string `json:"oidcState"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Errorf("decode custom token request: %v", err)
				}
				if request.Code != "iam-code" || request.State == "" {
					t.Errorf("custom token request = %+v", request)
				}
			} else {
				if err := r.ParseForm(); err != nil {
					t.Errorf("parse login form: %v", err)
				}
				if r.Form.Get("client_id") != clientID {
					t.Errorf("standard token form client_id = %v", r.Form)
				}
				if r.Form.Get("grant_type") == "refresh_token" {
					if r.Form.Get("refresh_token") != refreshToken {
						t.Errorf("refresh token form = %v", r.Form)
					}
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"access_token":"` + refreshedAccessToken + `","refresh_token":"` + refreshedRefresh + `","token_type":"Bearer","scope":"offline_access","expires_in":3600,"refresh_token_expires_in":604800}`))
					return
				}
				if r.Form.Get("grant_type") != "authorization_code" || r.Form.Get("code") != "iam-code" || r.Form.Get("redirect_uri") != redirectURI {
					t.Errorf("authorization code form = %v", r.Form)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			if customExchange {
				w.Header().Set("X-Token-Type", "bearer")
				_, _ = w.Write([]byte(`{"data":{"gatewayToken":"` + accessToken + `","expireSeconds":"3600"}}`))
			} else {
				_, _ = w.Write([]byte(`{"access_token":"` + accessToken + `","refresh_token":"` + refreshToken + `","token_type":"Bearer","scope":"offline_access","expires_in":3600,"refresh_token_expires_in":604800}`))
			}
		case "/items":
			businessRequests.Add(1)
			expectedToken := accessToken
			if !customExchange && loginRequests.Load() > 1 {
				expectedToken = refreshedAccessToken
			}
			if got := r.Header.Get("Authorization"); got != "Bearer "+expectedToken {
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
  redirect_uri: %s
`, server.URL, clientID, server.URL, server.URL, redirectURI)
	if customExchange {
		runtimeYAML += `  token_exchange:
    method: POST
    body_format: json
    parameters:
      - {source: code, name: authorizationCode, in: body, required: true}
      - {source: state, name: oidcState, in: body, required: true}
    response:
      access_token: {in: body, path: data.gatewayToken}
      token_type: {in: header, path: X-Token-Type}
      expires_in: {in: body, path: data.expireSeconds}
`
	}
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
	login := exec.Command("go", "run", "./cmd/oauthcli", "--trace", "login", "--no-browser")
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
	if trace := stderr.String(); !strings.Contains(trace, "[opencli][oauth] token_exchange") ||
		!strings.Contains(trace, "method: POST") || !strings.Contains(trace, "status: 200") {
		t.Fatalf("OAuth trace is missing safe exchange metadata: %s", trace)
	} else if strings.Contains(trace, "iam-code") || strings.Contains(trace, accessToken) {
		t.Fatalf("OAuth trace leaked code or token: %s", trace)
	}
	if info, err := os.Stat(tokenFile); err != nil {
		t.Fatalf("stat OAuth token file: %v", err)
	} else if info.Mode().Perm() != 0o600 {
		t.Fatalf("OAuth token file mode = %o, want 600", info.Mode().Perm())
	}
	runStatus := func() ([]byte, error) {
		status := exec.Command("go", "run", "./cmd/oauthcli", "status")
		status.Dir = dir
		status.Env = append(runtimeTestEnv(dir),
			"OPENCLI_CONFIG="+filepath.Join(dir, "config", "runtime.yaml"),
			"OPENCLI_OAUTH_TOKEN_FILE="+tokenFile,
		)
		return status.CombinedOutput()
	}
	statusOut, err := runStatus()
	if err != nil || !strings.Contains(string(statusOut), "valid") {
		t.Fatalf("OAuth status = %s, err=%v", statusOut, err)
	}
	if !customExchange {
		raw, readErr := os.ReadFile(tokenFile)
		if readErr != nil {
			t.Fatalf("read OAuth token file: %v", readErr)
		}
		var stored map[string]any
		if err := json.Unmarshal(raw, &stored); err != nil {
			t.Fatalf("decode OAuth token file: %v", err)
		}
		if stored["refresh_token"] != refreshToken || stored["scope"] != "offline_access" {
			t.Fatalf("stored OAuth token = %#v", stored)
		}
		stored["expires_at"] = time.Now().Add(-time.Minute).UTC().Format(time.RFC3339Nano)
		updated, _ := json.Marshal(stored)
		if err := os.WriteFile(tokenFile, updated, 0o600); err != nil {
			t.Fatalf("expire OAuth token: %v", err)
		}
		statusOut, err = runStatus()
		if err != nil || !strings.Contains(string(statusOut), "needs_refresh") {
			t.Fatalf("OAuth refresh status = %s, err=%v", statusOut, err)
		}
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
	wantLoginRequests := int32(1)
	if !customExchange {
		wantLoginRequests = 2
		raw, readErr := os.ReadFile(tokenFile)
		if readErr != nil || !bytes.Contains(raw, []byte(refreshedRefresh)) {
			t.Fatalf("rotated refresh token was not stored: %s, err=%v", raw, readErr)
		}
	}
	if loginRequests.Load() != wantLoginRequests || businessRequests.Load() != 1 {
		t.Fatalf("requests: login=%d business=%d, want login=%d business=1", loginRequests.Load(), businessRequests.Load(), wantLoginRequests)
	}

}

func TestOAuth2AuthorizationCodeGeneratesRustTarget(t *testing.T) {
	specPath := filepath.Join(t.TempDir(), "oauth-with-business-auth.yaml")
	if err := os.WriteFile(specPath, []byte(`
openapi: 3.0.3
info: {title: OAuth API, version: 1.0.0}
paths:
  /api/login:
    post:
      tags: [auth]
      operationId: businessLogin
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
          authorizationUrl: https://iam.example.com/idp/oauth2/authorize
          tokenUrl: https://business.example.com/cli-auth/login
          scopes: {}
`), 0o600); err != nil {
		t.Fatalf("write OAuth spec: %v", err)
	}
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
	dir := t.TempDir()
	err := app.RunGenerate(app.GenerateOptions{
		Input:             specPath,
		Output:            dir,
		Module:            "github.com/acme/generated",
		AppName:           "oauthcli",
		Auth:              "oauth2",
		Target:            "rust",
		SkillLang:         "zh",
		RuntimeConfigPath: runtimeSource,
	})
	if err != nil {
		t.Fatalf("generate Rust authorization_code CLI: %v", err)
	}
	for _, rel := range []string{"src/oauth_auth.rs", "src/cli.rs", "src/runtime_config.rs", "skills/cli-auth/SKILL.md"} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Fatalf("missing generated %s: %v", rel, err)
		}
	}
	cliSource, err := os.ReadFile(filepath.Join(dir, "src", "cli.rs"))
	if err != nil {
		t.Fatalf("read generated Rust cli.rs: %v", err)
	}
	cliText := string(cliSource)
	if !strings.Contains(cliText, `#[command(name = "login")]`) ||
		!strings.Contains(cliText, `#[command(name = "status")]`) ||
		!strings.Contains(cliText, `#[command(name = "logout")]`) ||
		!strings.Contains(cliText, "GroupCommand::Login") ||
		!strings.Contains(cliText, "GroupCommand::Logout") {
		t.Fatalf("generated Rust CLI is missing built-in top-level login/status/logout commands:\n%s", cliSource)
	}
	if got := strings.Count(cliText, `#[command(name = "auth")]`); got != 1 {
		t.Fatalf("generated Rust CLI has %d auth root commands, want only the business auth group:\n%s", got, cliSource)
	}
	readmeSource, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatalf("read generated Rust README.md: %v", err)
	}
	for _, want := range []string{"oauthcli login", "oauthcli status", "oauthcli logout", "authorization_code"} {
		if !strings.Contains(string(readmeSource), want) {
			t.Fatalf("generated Rust README is missing %q:\n%s", want, readmeSource)
		}
	}
	loginSkill, err := os.ReadFile(filepath.Join(dir, "skills", "cli-auth", "SKILL.md"))
	if err != nil {
		t.Fatalf("read generated cli-auth skill: %v", err)
	}
	for _, want := range []string{"oauthcli login", "oauthcli login --no-browser", "oauthcli status", "oauthcli logout", "login_required"} {
		if !strings.Contains(string(loginSkill), want) {
			t.Fatalf("generated cli-auth skill is missing %q:\n%s", want, loginSkill)
		}
	}
	businessSkill, err := os.ReadFile(filepath.Join(dir, "skills", "auth", "SKILL.md"))
	if err != nil {
		t.Fatalf("read generated business auth skill: %v", err)
	}
	if !strings.Contains(string(businessSkill), "oauthcli login") {
		t.Fatalf("business auth skill does not declare the CLI login prerequisite:\n%s", businessSkill)
	}
	routerSkill, err := os.ReadFile(filepath.Join(dir, "skills", "SKILL.md"))
	if err != nil {
		t.Fatalf("read generated root skill router: %v", err)
	}
	for _, want := range []string{"cli-auth/SKILL.md", "login_required", "不要把业务 `auth login` 当作 CLI 登录"} {
		if !strings.Contains(string(routerSkill), want) {
			t.Fatalf("generated root skill router is missing %q:\n%s", want, routerSkill)
		}
	}
}

func TestGeneratedRustOAuth2AuthorizationCode(t *testing.T) {
	for _, customExchange := range []bool{false, true} {
		name := "standard form"
		if customExchange {
			name = "custom JSON"
		}
		t.Run(name, func(t *testing.T) {
			testGeneratedRustOAuth2AuthorizationCode(t, customExchange)
		})
	}
}

func testGeneratedRustOAuth2AuthorizationCode(t *testing.T, customExchange bool) {
	if _, err := exec.LookPath("cargo"); err != nil {
		t.Skip("cargo not installed")
	}
	const (
		clientID             = "business-cli"
		accessToken          = "rust-business-access-token"
		refreshToken         = "rust-business-refresh-token"
		refreshedAccessToken = "rust-refreshed-access-token"
		refreshedRefresh     = "rust-refreshed-refresh-token"
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
			if customExchange {
				var request struct {
					Code  string `json:"authorizationCode"`
					State string `json:"oidcState"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Errorf("decode custom token request: %v", err)
				}
				if request.Code != "iam-code" || request.State == "" {
					t.Errorf("custom token request = %+v", request)
				}
			} else {
				if err := r.ParseForm(); err != nil {
					t.Errorf("parse login form: %v", err)
				}
				if got := r.Form.Get("grant_type"); got == "refresh_token" {
					if r.Form.Get("refresh_token") != refreshToken {
						t.Errorf("refresh token form = %v", r.Form)
					}
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"access_token":"` + refreshedAccessToken + `","refresh_token":"` + refreshedRefresh + `","token_type":"Bearer","scope":"offline_access","expires_in":3600,"refresh_token_expires_in":604800}`))
					return
				} else if got != "authorization_code" {
					t.Errorf("grant_type = %q, want authorization_code", got)
				}
				if got := r.Form.Get("code"); got != "iam-code" {
					t.Errorf("code = %q, want iam-code", got)
				}
				if got := r.Form.Get("client_id"); got != clientID {
					t.Errorf("login client_id = %q, want %q", got, clientID)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			if customExchange {
				w.Header().Set("X-Token-Type", "bearer")
				_, _ = w.Write([]byte(`{"data":{"gatewayToken":"` + accessToken + `","expireSeconds":"3600"}}`))
			} else {
				_, _ = w.Write([]byte(`{"access_token":"` + accessToken + `","refresh_token":"` + refreshToken + `","token_type":"Bearer","scope":"offline_access","expires_in":3600,"refresh_token_expires_in":604800}`))
			}
		case "/items":
			businessRequests.Add(1)
			expectedToken := accessToken
			if !customExchange && loginRequests.Load() > 1 {
				expectedToken = refreshedAccessToken
			}
			if got := r.Header.Get("Authorization"); got != "Bearer "+expectedToken {
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
	if customExchange {
		runtimeYAML += `  token_exchange:
    method: POST
    body_format: json
    parameters:
      - {source: code, name: authorizationCode, in: body, required: true}
      - {source: state, name: oidcState, in: body, required: true}
    response:
      access_token: {in: body, path: data.gatewayToken}
      token_type: {in: header, path: X-Token-Type}
      expires_in: {in: body, path: data.expireSeconds}
`
	}
	if err := os.WriteFile(runtimeSource, []byte(runtimeYAML), 0o600); err != nil {
		t.Fatalf("write OAuth runtime source: %v", err)
	}
	dir := t.TempDir()
	if err := app.RunGenerate(app.GenerateOptions{
		Input:             specPath,
		Output:            dir,
		Module:            "oauthcli",
		AppName:           "oauthcli",
		Auth:              "oauth2",
		Target:            "rust",
		RuntimeConfigPath: runtimeSource,
	}); err != nil {
		t.Fatalf("generate Rust OAuth CLI: %v", err)
	}

	tokenFile := filepath.Join(t.TempDir(), "oauth-token.json")
	login := exec.Command("cargo", "run", "--quiet", "--", "--trace", "login", "--no-browser")
	login.Dir = dir
	login.Env = append(runtimeTestEnv(dir),
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
		output := stderr.String()
		if strings.Contains(output, "Could not resolve host") || strings.Contains(output, "failed to download from") {
			t.Skipf("cargo run skipped due to dependency network restriction:\n%s", output)
		}
		t.Fatalf("read authorize URL: %v; stderr=%s", readErr, output)
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
	if trace := stderr.String(); !strings.Contains(trace, "[opencli][oauth] token_exchange") ||
		!strings.Contains(trace, "method: POST") || !strings.Contains(trace, "status: 200") {
		t.Fatalf("Rust OAuth trace is missing safe exchange metadata: %s", trace)
	} else if strings.Contains(trace, "iam-code") || strings.Contains(trace, accessToken) {
		t.Fatalf("Rust OAuth trace leaked code or token: %s", trace)
	}
	if info, err := os.Stat(tokenFile); err != nil {
		t.Fatalf("stat OAuth token file: %v", err)
	} else if info.Mode().Perm() != 0o600 {
		t.Fatalf("OAuth token file mode = %o, want 600", info.Mode().Perm())
	}
	runStatus := func() ([]byte, error) {
		status := exec.Command("cargo", "run", "--quiet", "--", "status")
		status.Dir = dir
		status.Env = append(runtimeTestEnv(dir), "OPENCLI_OAUTH_TOKEN_FILE="+tokenFile)
		return status.CombinedOutput()
	}
	statusOut, err := runStatus()
	if err != nil || !strings.Contains(string(statusOut), "valid") {
		t.Fatalf("Rust OAuth status = %s, err=%v", statusOut, err)
	}
	if !customExchange {
		raw, readErr := os.ReadFile(tokenFile)
		if readErr != nil {
			t.Fatalf("read Rust OAuth token: %v", readErr)
		}
		var stored map[string]any
		if err := json.Unmarshal(raw, &stored); err != nil {
			t.Fatalf("decode Rust OAuth token: %v", err)
		}
		if stored["refresh_token"] != refreshToken {
			t.Fatalf("stored Rust OAuth token = %#v", stored)
		}
		stored["expires_at"] = time.Now().Add(-time.Minute).Unix()
		updated, _ := json.Marshal(stored)
		if err := os.WriteFile(tokenFile, updated, 0o600); err != nil {
			t.Fatalf("expire Rust OAuth token: %v", err)
		}
		statusOut, err = runStatus()
		if err != nil || !strings.Contains(string(statusOut), "needs_refresh") {
			t.Fatalf("Rust OAuth refresh status = %s, err=%v", statusOut, err)
		}
	}

	call := exec.Command("cargo", "run", "--quiet", "--", "items", "list")
	call.Dir = dir
	call.Env = append(runtimeTestEnv(dir),
		"OPENCLI_OAUTH_TOKEN_FILE="+tokenFile,
	)
	out, err := call.CombinedOutput()
	if err != nil {
		t.Fatalf("run generated Rust OAuth CLI: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "ok") {
		t.Fatalf("generated Rust OAuth CLI output = %s", out)
	}
	wantLoginRequests := int32(1)
	if !customExchange {
		wantLoginRequests = 2
		raw, readErr := os.ReadFile(tokenFile)
		if readErr != nil || !bytes.Contains(raw, []byte(refreshedRefresh)) {
			t.Fatalf("rotated Rust refresh token was not stored: %s, err=%v", raw, readErr)
		}
	}
	if loginRequests.Load() != wantLoginRequests || businessRequests.Load() != 1 {
		t.Fatalf("requests: login=%d business=%d, want login=%d business=1", loginRequests.Load(), businessRequests.Load(), wantLoginRequests)
	}

	logout := exec.Command("cargo", "run", "--quiet", "--", "logout")
	logout.Dir = dir
	logout.Env = append(runtimeTestEnv(dir),
		"OPENCLI_OAUTH_TOKEN_FILE="+tokenFile,
	)
	if out, err := logout.CombinedOutput(); err != nil {
		t.Fatalf("logout generated Rust OAuth CLI: %v\n%s", err, out)
	}
	if _, err := os.Stat(tokenFile); !os.IsNotExist(err) {
		t.Fatalf("OAuth token file still exists after logout: %v", err)
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
		case "OPENCLI_BASE_URL", "OPENCLI_AUTH_TOKEN", "OPENCLI_API_KEY", "OPENCLI_OAUTH_CLIENT_SECRET", "OPENCLI_OAUTH_TOKEN_FILE", "OPENCLI_CONFIG":
			continue
		}
		env = append(env, value)
	}
	return append(env,
		"GOTOOLCHAIN=local",
		"GOCACHE="+filepath.Join(dir, ".gocache"),
	)
}
