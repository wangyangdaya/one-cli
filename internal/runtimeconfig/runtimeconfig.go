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
	envelopePrefix = "ENC[v1:"
	envelopeSuffix = "]"
)

type SealOptions struct {
	AuthMode string
	Getenv   func(string) string
	Random   io.Reader
}

type Bundle struct {
	YAML      []byte
	KeyShareA [32]byte
	KeyShareB [32]byte
	HasSecret bool
}

type sourceConfig struct {
	Version string      `yaml:"version"`
	BaseURL string      `yaml:"base_url,omitempty"`
	Auth    *sourceAuth `yaml:"auth,omitempty"`
}

type sourceAuth struct {
	Type   string `yaml:"type"`
	Header string `yaml:"header,omitempty"`
}

type sealedConfig struct {
	Version string      `yaml:"version"`
	BaseURL string      `yaml:"base_url,omitempty"`
	Auth    *sealedAuth `yaml:"auth,omitempty"`
}

type sealedAuth struct {
	Type           string `yaml:"type"`
	Header         string `yaml:"header,omitempty"`
	EncryptedValue string `yaml:"encrypted_value"`
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
	if err := validateSource(source, opts.AuthMode); err != nil {
		return Bundle{}, fmt.Errorf("validate runtime config %s: %w", path, err)
	}

	output := sealedConfig{
		Version: "v1",
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
	if strings.TrimSpace(source.Auth.Type) == "api_key" {
		envName = "OPENCLI_API_KEY"
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
	sealed := gcm.Seal(nil, nonce, []byte(credential), additionalData(authType, header))
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
		Type:           authType,
		Header:         header,
		EncryptedValue: envelopePrefix + base64.RawURLEncoding.EncodeToString(payload) + envelopeSuffix,
	}
	bundle.YAML, err = yaml.Marshal(output)
	if err != nil {
		return Bundle{}, fmt.Errorf("encode runtime config: %w", err)
	}
	return bundle, nil
}

func validateSource(source sourceConfig, authMode string) error {
	if strings.TrimSpace(source.Version) != "v1" {
		return fmt.Errorf("version must be v1")
	}
	mode := strings.TrimSpace(authMode)
	if mode == "" {
		mode = "none"
	}
	if source.Auth == nil {
		if mode == "token" || mode == "api_key" {
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
	default:
		return fmt.Errorf("unsupported runtime auth type %q", authType)
	}
	return nil
}

func additionalData(authType, header string) []byte {
	return []byte("opencli:v1:" + strings.TrimSpace(authType) + ":" + strings.ToLower(strings.TrimSpace(header)))
}
