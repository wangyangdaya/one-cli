package runtimeconfig

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoadAndSealBearer(t *testing.T) {
	path := writeRuntimeSource(t, `
base_url: https://api.example.com
auth:
  type: bearer
`)
	credential := "bearer-file-secret"

	bundle, err := LoadAndSeal(path, SealOptions{
		AuthMode: "token",
		Getenv: func(key string) string {
			if key == "OPENCLI_AUTH_TOKEN" {
				return credential
			}
			return ""
		},
		Random: bytes.NewReader(bytes.Repeat([]byte{0x3c}, 128)),
	})
	if err != nil {
		t.Fatalf("LoadAndSeal: %v", err)
	}
	if !bundle.HasSecret {
		t.Fatal("expected sealed secret")
	}
	if bytes.Contains(bundle.YAML, []byte(credential)) {
		t.Fatalf("sealed YAML contains plaintext credential: %s", bundle.YAML)
	}

	var generated struct {
		BaseURL string `yaml:"base_url"`
		Auth    struct {
			Type           string `yaml:"type"`
			Header         string `yaml:"header"`
			EncryptedValue string `yaml:"encrypted_value"`
		} `yaml:"auth"`
	}
	if err := yaml.Unmarshal(bundle.YAML, &generated); err != nil {
		t.Fatalf("unmarshal generated YAML: %v", err)
	}
	if generated.BaseURL != "https://api.example.com" || generated.Auth.Type != "bearer" {
		t.Fatalf("unexpected generated config: %+v", generated)
	}
	if got := decryptForTest(t, generated.Auth.EncryptedValue, generated.Auth.Type, generated.Auth.Header, bundle); got != credential {
		t.Fatalf("decrypted credential = %q, want %q", got, credential)
	}
}

func TestLoadAndSealTokenWithoutAuthUsesRuntimeEnvironment(t *testing.T) {
	path := writeRuntimeSource(t, "base_url: https://api.example.com\n")
	bundle, err := LoadAndSeal(path, SealOptions{
		AuthMode: "token",
		Getenv: func(key string) string {
			t.Fatalf("unexpected credential lookup for %s", key)
			return ""
		},
	})
	if err != nil {
		t.Fatalf("LoadAndSeal: %v", err)
	}
	if bundle.HasSecret {
		t.Fatal("base-URL-only token config unexpectedly has a secret")
	}
	if !bytes.Contains(bundle.YAML, []byte("base_url: https://api.example.com")) {
		t.Fatalf("generated YAML missing base URL: %s", bundle.YAML)
	}
	if bytes.Contains(bundle.YAML, []byte("auth:")) {
		t.Fatalf("generated YAML unexpectedly contains auth metadata: %s", bundle.YAML)
	}
}

func TestLoadAndSealAcceptsLegacyVersionV1(t *testing.T) {
	path := writeRuntimeSource(t, `
version: v1
base_url: https://api.example.com
`)

	bundle, err := LoadAndSeal(path, SealOptions{AuthMode: "none"})
	if err != nil {
		t.Fatalf("LoadAndSeal legacy version: %v", err)
	}
	if !bytes.Contains(bundle.YAML, []byte("base_url: https://api.example.com")) {
		t.Fatalf("generated YAML missing base URL: %s", bundle.YAML)
	}
}

func TestLoadAndSealAPIKey(t *testing.T) {
	path := writeRuntimeSource(t, `
base_url: https://api.example.com
auth:
  type: api_key
  header: X-API-Key
`)

	bundle, err := LoadAndSeal(path, SealOptions{
		AuthMode: "api_key",
		Getenv: func(key string) string {
			if key == "OPENCLI_API_KEY" {
				return "api-key-secret"
			}
			return ""
		},
		Random: bytes.NewReader(bytes.Repeat([]byte{0x5a}, 128)),
	})
	if err != nil {
		t.Fatalf("LoadAndSeal: %v", err)
	}
	if !bytes.Contains(bundle.YAML, []byte("header: X-API-Key")) {
		t.Fatalf("generated YAML missing API-key header: %s", bundle.YAML)
	}
	if bytes.Contains(bundle.YAML, []byte("api-key-secret")) {
		t.Fatalf("generated YAML contains API key: %s", bundle.YAML)
	}
}

func TestLoadAndSealOAuth2ClientCredentials(t *testing.T) {
	path := writeRuntimeSource(t, `
base_url: https://api.example.com
auth:
  type: oauth2
  client_id: fssc-opencli
`)
	const credential = "oauth-client-secret"
	bundle, err := LoadAndSeal(path, SealOptions{
		AuthMode: "oauth2",
		OAuth2: OAuth2Defaults{
			GrantType: "client_credentials",
			Scheme:    "fsscOAuth",
			TokenURL:  "https://identity.example.com/oauth/token",
			Placement: "query",
		},
		Getenv: func(key string) string {
			if key == "OPENCLI_OAUTH_CLIENT_SECRET" {
				return credential
			}
			return ""
		},
		Random: bytes.NewReader(bytes.Repeat([]byte{0x6b}, 128)),
	})
	if err != nil {
		t.Fatalf("LoadAndSeal: %v", err)
	}
	if bytes.Contains(bundle.YAML, []byte(credential)) {
		t.Fatalf("generated YAML contains plaintext client secret:\n%s", bundle.YAML)
	}
	if !bytes.Contains(bundle.YAML, []byte("client_id: fssc-opencli")) ||
		!bytes.Contains(bundle.YAML, []byte("token_url: https://identity.example.com/oauth/token")) ||
		!bytes.Contains(bundle.YAML, []byte("placement: query")) {
		t.Fatalf("generated YAML is missing OAuth metadata:\n%s", bundle.YAML)
	}

	var generated struct {
		Auth struct {
			Type           string `yaml:"type"`
			GrantType      string `yaml:"grant_type"`
			Scheme         string `yaml:"scheme"`
			TokenURL       string `yaml:"token_url"`
			ClientID       string `yaml:"client_id"`
			EncryptedValue string `yaml:"encrypted_value"`
			ClientAuth     struct {
				Method    string `yaml:"method"`
				Placement string `yaml:"placement"`
			} `yaml:"client_auth"`
		} `yaml:"auth"`
	}
	if err := yaml.Unmarshal(bundle.YAML, &generated); err != nil {
		t.Fatalf("unmarshal generated YAML: %v", err)
	}
	aad := strings.Join([]string{
		generated.Auth.GrantType,
		generated.Auth.Scheme,
		generated.Auth.TokenURL,
		generated.Auth.ClientID,
		generated.Auth.ClientAuth.Method,
		generated.Auth.ClientAuth.Placement,
	}, "|")
	if got := decryptForTest(t, generated.Auth.EncryptedValue, generated.Auth.Type, aad, bundle); got != credential {
		t.Fatalf("decrypted credential = %q, want %q", got, credential)
	}
}

func TestLoadAndSealRejectsInvalidSources(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		authMode string
		getenv   func(string) string
		want     string
	}{
		{
			name:     "unknown field",
			source:   "base_urll: https://api.example.com\n",
			authMode: "none",
			want:     "base_urll",
		},
		{
			name:     "unsupported version",
			source:   "version: v2\nbase_url: https://api.example.com\n",
			authMode: "none",
			want:     "version must be v1",
		},
		{
			name:     "plaintext token",
			source:   "auth:\n  type: bearer\n  token: exposed\n",
			authMode: "token",
			getenv:   func(string) string { return "secret" },
			want:     "token",
		},
		{
			name:     "missing credential",
			source:   "auth:\n  type: bearer\n",
			authMode: "token",
			want:     "OPENCLI_AUTH_TOKEN",
		},
		{
			name:     "auth mismatch",
			source:   "auth:\n  type: api_key\n  header: X-API-Key\n",
			authMode: "token",
			getenv:   func(string) string { return "secret" },
			want:     "api_key",
		},
		{
			name:     "api key missing header",
			source:   "auth:\n  type: api_key\n",
			authMode: "api_key",
			getenv:   func(string) string { return "secret" },
			want:     "header",
		},
		{
			name:     "api key missing auth metadata",
			source:   "base_url: https://api.example.com\n",
			authMode: "api_key",
			want:     "auth metadata",
		},
		{
			name:     "oauth2 missing auth metadata",
			source:   "base_url: https://api.example.com\n",
			authMode: "oauth2",
			want:     "auth metadata",
		},
		{
			name:     "unsupported oauth grant type",
			source:   "auth:\n  type: oauth2\n  grant_type: authorization_code\n",
			authMode: "oauth2",
			want:     "grant_type client_credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeRuntimeSource(t, tt.source)
			_, err := LoadAndSeal(path, SealOptions{
				AuthMode: tt.authMode,
				Getenv:   tt.getenv,
				Random:   bytes.NewReader(bytes.Repeat([]byte{0x22}, 128)),
			})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestLoadAndSealWithoutAuthDoesNotRequireCredential(t *testing.T) {
	path := writeRuntimeSource(t, "base_url: http://localhost:8080\n")
	bundle, err := LoadAndSeal(path, SealOptions{
		AuthMode: "none",
		Random:   bytes.NewReader(bytes.Repeat([]byte{0x11}, 128)),
	})
	if err != nil {
		t.Fatalf("LoadAndSeal: %v", err)
	}
	if bundle.HasSecret {
		t.Fatal("no-auth config unexpectedly has a secret")
	}
}

func writeRuntimeSource(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "runtime.yaml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o600); err != nil {
		t.Fatalf("write runtime source: %v", err)
	}
	return path
}

func decryptForTest(t *testing.T, envelope, authType, header string, bundle Bundle) string {
	t.Helper()
	if !strings.HasPrefix(envelope, "ENC[v1:") || !strings.HasSuffix(envelope, "]") {
		t.Fatalf("invalid envelope %q", envelope)
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSuffix(strings.TrimPrefix(envelope, "ENC[v1:"), "]"))
	if err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	var key [32]byte
	for i := range key {
		key[i] = bundle.KeyShareA[i] ^ bundle.KeyShareB[i]
	}
	block, err := aes.NewCipher(key[:])
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("new GCM: %v", err)
	}
	if len(raw) <= gcm.NonceSize() {
		t.Fatalf("sealed payload too short: %d", len(raw))
	}
	plaintext, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], []byte("opencli:v1:"+authType+":"+strings.ToLower(header)))
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	return string(plaintext)
}
