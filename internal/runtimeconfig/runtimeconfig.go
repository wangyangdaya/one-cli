package runtimeconfig

import (
	"crypto/aes"
	"crypto/cipher"
	cryptorand "crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	EnvelopePrefix = "ENC[v1:"
	EnvelopeSuffix = "]"
	AADPrefix      = "opencli:v1:"
)

type SealOptions struct {
	AuthMode string
	OAuth2   OAuth2Defaults
	Getenv   func(string) string
	Random   io.Reader
}

type OAuth2Defaults struct {
	GrantType string
	Scheme    string
	TokenURL  string
	Placement string
	Scopes    []string
}

type Bundle struct {
	YAML      []byte
	KeyShareA [32]byte
	KeyShareB [32]byte
	HasSecret bool
}

type sourceConfig struct {
	Version string      `yaml:"version,omitempty"`
	BaseURL string      `yaml:"base_url,omitempty"`
	Auth    *sourceAuth `yaml:"auth,omitempty"`
}

type sourceAuth struct {
	Type       string           `yaml:"type"`
	Header     string           `yaml:"header,omitempty"`
	GrantType  string           `yaml:"grant_type,omitempty"`
	Scheme     string           `yaml:"scheme,omitempty"`
	TokenURL   string           `yaml:"token_url,omitempty"`
	ClientID   string           `yaml:"client_id,omitempty"`
	ClientAuth sourceClientAuth `yaml:"client_auth,omitempty"`
	Scopes     []string         `yaml:"scopes,omitempty"`
}

type sourceClientAuth struct {
	Method    string `yaml:"method,omitempty"`
	Placement string `yaml:"placement,omitempty"`
}

type sealedConfig struct {
	Version string      `yaml:"version,omitempty"`
	BaseURL string      `yaml:"base_url,omitempty"`
	Auth    *sealedAuth `yaml:"auth,omitempty"`
}

type sealedAuth struct {
	Type           string           `yaml:"type"`
	Header         string           `yaml:"header,omitempty"`
	GrantType      string           `yaml:"grant_type,omitempty"`
	Scheme         string           `yaml:"scheme,omitempty"`
	TokenURL       string           `yaml:"token_url,omitempty"`
	ClientID       string           `yaml:"client_id,omitempty"`
	ClientAuth     sourceClientAuth `yaml:"client_auth,omitempty"`
	Scopes         []string         `yaml:"scopes,omitempty"`
	EncryptedValue string           `yaml:"encrypted_value"`
}

func LoadAndSeal(path string, opts SealOptions) (Bundle, error) {
	file, err := os.Open(path)
	if err != nil {
		return Bundle{}, fmt.Errorf("open runtime config %s: %w", path, err)
	}
	defer file.Close()

	var source sourceConfig
	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)
	if err := decoder.Decode(&source); err != nil {
		return Bundle{}, fmt.Errorf("parse runtime config %s: %w", path, err)
	}
	applyOAuth2Defaults(&source, opts.OAuth2)
	if err := validateSource(source, opts.AuthMode); err != nil {
		return Bundle{}, fmt.Errorf("validate runtime config %s: %w", path, err)
	}

	output := sealedConfig{
		Version: strings.TrimSpace(source.Version),
		BaseURL: strings.TrimSpace(source.BaseURL),
	}
	if source.Auth == nil {
		rendered, err := yaml.Marshal(output)
		if err != nil {
			return Bundle{}, fmt.Errorf("encode runtime config: %w", err)
		}
		return Bundle{YAML: rendered}, nil
	}

	getenv := opts.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	envName := "OPENCLI_AUTH_TOKEN"
	switch strings.TrimSpace(source.Auth.Type) {
	case "api_key":
		envName = "OPENCLI_API_KEY"
	case "oauth2":
		envName = "OPENCLI_OAUTH_CLIENT_SECRET"
	}
	credential := strings.TrimSpace(getenv(envName))
	if credential == "" {
		return Bundle{}, fmt.Errorf("%s is required to seal runtime auth", envName)
	}

	random := opts.Random
	if random == nil {
		random = cryptorand.Reader
	}
	key := make([]byte, 32)
	if _, err := io.ReadFull(random, key); err != nil {
		return Bundle{}, fmt.Errorf("generate runtime seal key: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return Bundle{}, fmt.Errorf("create runtime cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return Bundle{}, fmt.Errorf("create runtime cipher mode: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(random, nonce); err != nil {
		return Bundle{}, fmt.Errorf("generate runtime nonce: %w", err)
	}
	authType := strings.TrimSpace(source.Auth.Type)
	header := strings.TrimSpace(source.Auth.Header)
	aadContext := header
	if authType == "oauth2" {
		aadContext = oauth2AADContext(*source.Auth)
	}
	sealed := gcm.Seal(nil, nonce, []byte(credential), additionalData(authType, aadContext))
	payload := append(append([]byte(nil), nonce...), sealed...)

	var bundle Bundle
	if _, err := io.ReadFull(random, bundle.KeyShareA[:]); err != nil {
		return Bundle{}, fmt.Errorf("generate runtime key share: %w", err)
	}
	for i := range bundle.KeyShareB {
		bundle.KeyShareB[i] = key[i] ^ bundle.KeyShareA[i]
	}
	bundle.HasSecret = true
	output.Auth = &sealedAuth{
		Type:      authType,
		Header:    header,
		GrantType: strings.TrimSpace(source.Auth.GrantType),
		Scheme:    strings.TrimSpace(source.Auth.Scheme),
		TokenURL:  strings.TrimSpace(source.Auth.TokenURL),
		ClientID:  strings.TrimSpace(source.Auth.ClientID),
		ClientAuth: sourceClientAuth{
			Method:    strings.TrimSpace(source.Auth.ClientAuth.Method),
			Placement: strings.TrimSpace(source.Auth.ClientAuth.Placement),
		},
		Scopes:         append([]string(nil), source.Auth.Scopes...),
		EncryptedValue: EnvelopePrefix + base64.RawURLEncoding.EncodeToString(payload) + EnvelopeSuffix,
	}
	bundle.YAML, err = yaml.Marshal(output)
	if err != nil {
		return Bundle{}, fmt.Errorf("encode runtime config: %w", err)
	}
	return bundle, nil
}

func validateSource(source sourceConfig, authMode string) error {
	if version := strings.TrimSpace(source.Version); version != "" && version != "v1" {
		return fmt.Errorf("version must be v1")
	}
	mode := strings.TrimSpace(authMode)
	if mode == "" {
		mode = "none"
	}
	if source.Auth == nil {
		if mode == "api_key" || mode == "oauth2" {
			return fmt.Errorf("auth metadata is required for generated auth mode %s", mode)
		}
		return nil
	}

	authType := strings.TrimSpace(source.Auth.Type)
	expected := mode
	if mode == "token" {
		expected = "bearer"
	}
	if authType != expected {
		return fmt.Errorf("auth type %q is incompatible with generated auth mode %q", authType, mode)
	}
	switch authType {
	case "bearer":
		if strings.TrimSpace(source.Auth.Header) != "" {
			return fmt.Errorf("bearer auth must not define header")
		}
	case "api_key":
		if strings.TrimSpace(source.Auth.Header) == "" {
			return fmt.Errorf("api_key auth requires header")
		}
	case "oauth2":
		if strings.TrimSpace(source.Auth.GrantType) != "client_credentials" {
			return fmt.Errorf("oauth2 auth requires grant_type client_credentials")
		}
		if strings.TrimSpace(source.Auth.TokenURL) == "" {
			return fmt.Errorf("oauth2 auth requires token_url or one clientCredentials security scheme in OpenAPI")
		}
		if strings.TrimSpace(source.Auth.ClientID) == "" {
			return fmt.Errorf("oauth2 auth requires client_id")
		}
		if strings.TrimSpace(source.Auth.ClientAuth.Method) != "client_secret" {
			return fmt.Errorf("oauth2 client_auth.method must be client_secret")
		}
		switch strings.TrimSpace(source.Auth.ClientAuth.Placement) {
		case "basic", "body", "query":
		default:
			return fmt.Errorf("oauth2 client_auth.placement must be one of basic, body, or query")
		}
	default:
		return fmt.Errorf("unsupported runtime auth type %q", authType)
	}
	return nil
}

func applyOAuth2Defaults(source *sourceConfig, defaults OAuth2Defaults) {
	if source == nil || source.Auth == nil || strings.TrimSpace(source.Auth.Type) != "oauth2" {
		return
	}
	if strings.TrimSpace(source.Auth.GrantType) == "" {
		source.Auth.GrantType = strings.TrimSpace(defaults.GrantType)
	}
	if strings.TrimSpace(source.Auth.Scheme) == "" {
		source.Auth.Scheme = strings.TrimSpace(defaults.Scheme)
	}
	if strings.TrimSpace(source.Auth.TokenURL) == "" {
		source.Auth.TokenURL = strings.TrimSpace(defaults.TokenURL)
	}
	if strings.TrimSpace(source.Auth.ClientAuth.Method) == "" {
		source.Auth.ClientAuth.Method = "client_secret"
	}
	if strings.TrimSpace(source.Auth.ClientAuth.Placement) == "" {
		source.Auth.ClientAuth.Placement = strings.TrimSpace(defaults.Placement)
	}
	if source.Auth.ClientAuth.Placement == "" {
		source.Auth.ClientAuth.Placement = "basic"
	}
	if len(source.Auth.Scopes) == 0 {
		source.Auth.Scopes = append([]string(nil), defaults.Scopes...)
	}
}

func oauth2AADContext(auth sourceAuth) string {
	return strings.Join([]string{
		strings.TrimSpace(auth.GrantType),
		strings.TrimSpace(auth.Scheme),
		strings.TrimSpace(auth.TokenURL),
		strings.TrimSpace(auth.ClientID),
		strings.TrimSpace(auth.ClientAuth.Method),
		strings.TrimSpace(auth.ClientAuth.Placement),
	}, "|")
}

func additionalData(authType, header string) []byte {
	return []byte(AADPrefix + strings.TrimSpace(authType) + ":" + strings.ToLower(strings.TrimSpace(header)))
}
