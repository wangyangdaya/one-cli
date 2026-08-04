package runtimeconfig

import (
	"crypto/aes"
	"crypto/cipher"
	cryptorand "crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
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
	YAML            []byte
	KeyShareA       [32]byte
	KeyShareB       [32]byte
	HasSecret       bool
	OAuth2GrantType string
	OAuth2TokenURL  string
}

type sourceConfig struct {
	Version string      `yaml:"version,omitempty"`
	BaseURL string      `yaml:"base_url,omitempty"`
	Auth    *sourceAuth `yaml:"auth,omitempty"`
}

type sourceAuth struct {
	Type             string           `yaml:"type"`
	Header           string           `yaml:"header,omitempty"`
	GrantType        string           `yaml:"grant_type,omitempty"`
	Scheme           string           `yaml:"scheme,omitempty"`
	AuthorizationURL string           `yaml:"authorization_url,omitempty"`
	TokenURL         string           `yaml:"token_url,omitempty"`
	ClientID         string           `yaml:"client_id,omitempty"`
	RedirectURI      string           `yaml:"redirect_uri,omitempty"`
	ClientAuth       sourceClientAuth `yaml:"client_auth,omitempty"`
	Scopes           []string         `yaml:"scopes,omitempty"`
	TokenExchange    *tokenExchange   `yaml:"token_exchange,omitempty"`
}

type tokenExchange struct {
	Method     string                   `yaml:"method,omitempty"`
	BodyFormat string                   `yaml:"body_format,omitempty"`
	Parameters []tokenExchangeParameter `yaml:"parameters,omitempty"`
	Response   *tokenExchangeResponse   `yaml:"response,omitempty"`
}

type tokenExchangeParameter struct {
	Source   string `yaml:"source"`
	Name     string `yaml:"name"`
	In       string `yaml:"in"`
	Required bool   `yaml:"required,omitempty"`
	Value    string `yaml:"value,omitempty"`
}

type tokenExchangeResponse struct {
	AccessToken *tokenExchangeResult `yaml:"access_token,omitempty"`
	TokenType   *tokenExchangeResult `yaml:"token_type,omitempty"`
	ExpiresIn   *tokenExchangeResult `yaml:"expires_in,omitempty"`
}

type tokenExchangeResult struct {
	In   string `yaml:"in"`
	Path string `yaml:"path"`
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
	Type             string           `yaml:"type"`
	Header           string           `yaml:"header,omitempty"`
	GrantType        string           `yaml:"grant_type,omitempty"`
	Scheme           string           `yaml:"scheme,omitempty"`
	AuthorizationURL string           `yaml:"authorization_url,omitempty"`
	TokenURL         string           `yaml:"token_url,omitempty"`
	ClientID         string           `yaml:"client_id,omitempty"`
	RedirectURI      string           `yaml:"redirect_uri,omitempty"`
	ClientAuth       sourceClientAuth `yaml:"client_auth,omitempty"`
	Scopes           []string         `yaml:"scopes,omitempty"`
	TokenExchange    *tokenExchange   `yaml:"token_exchange,omitempty"`
	EncryptedValue   string           `yaml:"encrypted_value,omitempty"`
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
	if strings.TrimSpace(source.Auth.Type) == "oauth2" && strings.TrimSpace(source.Auth.GrantType) == "authorization_code" {
		output.Auth = sealedAuthFromSource(*source.Auth)
		rendered, err := yaml.Marshal(output)
		if err != nil {
			return Bundle{}, fmt.Errorf("encode runtime config: %w", err)
		}
		return Bundle{
			YAML:            rendered,
			OAuth2GrantType: "authorization_code",
			OAuth2TokenURL:  strings.TrimSpace(source.Auth.TokenURL),
		}, nil
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
	output.Auth = sealedAuthFromSource(*source.Auth)
	output.Auth.EncryptedValue = EnvelopePrefix + base64.RawURLEncoding.EncodeToString(payload) + EnvelopeSuffix
	bundle.OAuth2GrantType = strings.TrimSpace(source.Auth.GrantType)
	bundle.OAuth2TokenURL = strings.TrimSpace(source.Auth.TokenURL)
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
		if source.Auth.TokenExchange != nil && strings.TrimSpace(source.Auth.GrantType) != "authorization_code" {
			return fmt.Errorf("oauth2 token_exchange is only supported for authorization_code")
		}
		switch strings.TrimSpace(source.Auth.GrantType) {
		case "authorization_code":
			if strings.TrimSpace(source.Auth.AuthorizationURL) == "" {
				return fmt.Errorf("oauth2 authorization_code requires authorization_url")
			}
			if strings.TrimSpace(source.Auth.TokenURL) == "" {
				return fmt.Errorf("oauth2 authorization_code requires token_url")
			}
			if strings.TrimSpace(source.Auth.ClientID) == "" {
				return fmt.Errorf("oauth2 authorization_code requires client_id")
			}
			if redirectURI := strings.TrimSpace(source.Auth.RedirectURI); redirectURI != "" {
				if err := validateLoopbackRedirectURI(redirectURI); err != nil {
					return err
				}
			}
			if strings.TrimSpace(source.Auth.ClientAuth.Method) != "" || strings.TrimSpace(source.Auth.ClientAuth.Placement) != "" {
				return fmt.Errorf("oauth2 authorization_code must not define client_auth")
			}
			if err := validateTokenExchange(source.Auth.TokenExchange); err != nil {
				return err
			}
			return nil
		case "client_credentials":
		default:
			return fmt.Errorf("oauth2 grant_type must be client_credentials or authorization_code")
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

func validateLoopbackRedirectURI(value string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "http" || parsed.Hostname() != "127.0.0.1" || parsed.Port() == "" || parsed.Path == "" || parsed.Path == "/" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil {
		return fmt.Errorf("oauth2 redirect_uri must be an HTTP 127.0.0.1 URL with a port and callback path")
	}
	return nil
}

func validateTokenExchange(exchange *tokenExchange) error {
	if exchange == nil {
		return nil
	}
	method := strings.ToUpper(strings.TrimSpace(exchange.Method))
	if method == "" {
		method = "POST"
	}
	switch method {
	case "POST", "PUT", "PATCH":
	default:
		return fmt.Errorf("oauth2 token_exchange method must be POST, PUT, or PATCH")
	}
	bodyFormat := strings.ToLower(strings.TrimSpace(exchange.BodyFormat))
	if bodyFormat != "" && bodyFormat != "form" && bodyFormat != "json" {
		return fmt.Errorf("oauth2 token_exchange body_format must be form or json")
	}
	hasBody := false
	hasCode := false
	for i, parameter := range exchange.Parameters {
		source := strings.ToLower(strings.TrimSpace(parameter.Source))
		name := strings.TrimSpace(parameter.Name)
		location := strings.ToLower(strings.TrimSpace(parameter.In))
		if name == "" {
			return fmt.Errorf("oauth2 token_exchange parameter %d requires name", i+1)
		}
		switch source {
		case "code":
			hasCode = true
		case "state", "client_id", "redirect_uri", "scope", "grant_type":
		case "literal":
			if strings.TrimSpace(parameter.Value) == "" {
				return fmt.Errorf("oauth2 token_exchange literal parameter %q requires value", name)
			}
		default:
			return fmt.Errorf("oauth2 token_exchange parameter %q has unsupported source %q", name, parameter.Source)
		}
		switch location {
		case "body":
			hasBody = true
		case "query", "header", "cookie":
		default:
			return fmt.Errorf("oauth2 token_exchange parameter %q has unsupported location %q", name, parameter.In)
		}
	}
	if !hasCode {
		return fmt.Errorf("oauth2 token_exchange requires one code parameter")
	}
	if hasBody && bodyFormat == "" {
		return fmt.Errorf("oauth2 token_exchange body_format is required for body parameters")
	}
	if exchange.Response != nil {
		for name, result := range map[string]*tokenExchangeResult{
			"access_token": exchange.Response.AccessToken,
			"token_type":   exchange.Response.TokenType,
			"expires_in":   exchange.Response.ExpiresIn,
		} {
			if err := validateTokenExchangeResult(name, result); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateTokenExchangeResult(name string, result *tokenExchangeResult) error {
	if result == nil {
		return nil
	}
	location := strings.ToLower(strings.TrimSpace(result.In))
	if location != "body" && location != "header" {
		return fmt.Errorf("oauth2 token_exchange response %s location must be body or header", name)
	}
	if strings.TrimSpace(result.Path) == "" {
		return fmt.Errorf("oauth2 token_exchange response %s requires path", name)
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
	if strings.TrimSpace(source.Auth.GrantType) == "client_credentials" && strings.TrimSpace(source.Auth.ClientAuth.Method) == "" {
		source.Auth.ClientAuth.Method = "client_secret"
	}
	if strings.TrimSpace(source.Auth.GrantType) == "client_credentials" && strings.TrimSpace(source.Auth.ClientAuth.Placement) == "" {
		source.Auth.ClientAuth.Placement = strings.TrimSpace(defaults.Placement)
	}
	if strings.TrimSpace(source.Auth.GrantType) == "client_credentials" && source.Auth.ClientAuth.Placement == "" {
		source.Auth.ClientAuth.Placement = "basic"
	}
	if len(source.Auth.Scopes) == 0 {
		source.Auth.Scopes = append([]string(nil), defaults.Scopes...)
	}
}

func sealedAuthFromSource(auth sourceAuth) *sealedAuth {
	return &sealedAuth{
		Type:             strings.TrimSpace(auth.Type),
		Header:           strings.TrimSpace(auth.Header),
		GrantType:        strings.TrimSpace(auth.GrantType),
		Scheme:           strings.TrimSpace(auth.Scheme),
		AuthorizationURL: strings.TrimSpace(auth.AuthorizationURL),
		TokenURL:         strings.TrimSpace(auth.TokenURL),
		ClientID:         strings.TrimSpace(auth.ClientID),
		RedirectURI:      strings.TrimSpace(auth.RedirectURI),
		ClientAuth: sourceClientAuth{
			Method:    strings.TrimSpace(auth.ClientAuth.Method),
			Placement: strings.TrimSpace(auth.ClientAuth.Placement),
		},
		Scopes:        append([]string(nil), auth.Scopes...),
		TokenExchange: normalizeTokenExchange(auth.TokenExchange),
	}
}

func normalizeTokenExchange(exchange *tokenExchange) *tokenExchange {
	if exchange == nil {
		return nil
	}
	result := &tokenExchange{
		Method:     strings.ToUpper(strings.TrimSpace(exchange.Method)),
		BodyFormat: strings.ToLower(strings.TrimSpace(exchange.BodyFormat)),
		Response:   exchange.Response,
	}
	if result.Method == "" {
		result.Method = "POST"
	}
	for _, parameter := range exchange.Parameters {
		result.Parameters = append(result.Parameters, tokenExchangeParameter{
			Source:   strings.ToLower(strings.TrimSpace(parameter.Source)),
			Name:     strings.TrimSpace(parameter.Name),
			In:       strings.ToLower(strings.TrimSpace(parameter.In)),
			Required: parameter.Required,
			Value:    parameter.Value,
		})
	}
	return result
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
