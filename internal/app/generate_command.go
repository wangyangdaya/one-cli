package app

import (
	"fmt"
	"net/url"
	"strings"

	"one-cli/internal/configgen"
	"one-cli/internal/loaders"
	"one-cli/internal/mcp"
	"one-cli/internal/model"
	"one-cli/internal/openapi"
	outjson "one-cli/internal/output"
	"one-cli/internal/planner"
	"one-cli/internal/render"
	"one-cli/internal/runtimeconfig"

	"github.com/spf13/cobra"
)

type GenerateOptions struct {
	Input             string
	MCPConfig         string
	Output            string
	Module            string
	AppName           string
	AppVersion        string
	ConfigPath        string
	SkillLang         string
	Auth              string
	Signer            string
	Target            string
	RuntimeConfigPath string
}

func NewGenerateCommand() *cobra.Command {
	var input string
	var mcpConfig string
	var output string
	var module string
	var appName string
	var appVersion string
	var configPath string
	var target string
	var skillLang string
	var auth string
	var signer string
	var runtimeConfigPath string

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate a Go or Rust CLI project from Swagger/OpenAPI or MCP",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := RunGenerate(GenerateOptions{
				Input:             input,
				MCPConfig:         mcpConfig,
				Output:            output,
				Module:            module,
				AppName:           appName,
				AppVersion:        appVersion,
				ConfigPath:        configPath,
				SkillLang:         skillLang,
				Auth:              auth,
				Signer:            signer,
				Target:            target,
				RuntimeConfigPath: runtimeConfigPath,
			}); err != nil {
				return err
			}
			if JSONEnabled(cmd) {
				selectedTarget := strings.TrimSpace(target)
				if selectedTarget == "" {
					selectedTarget = "go"
				}
				rendered, err := outjson.JSONSuccess(cmd.CommandPath(), "generated project", map[string]string{
					"output": strings.TrimSpace(output),
					"module": strings.TrimSpace(module),
					"app":    strings.TrimSpace(appName),
					"target": selectedTarget,
				})
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(cmd.OutOrStdout(), rendered)
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&input, "input", "", "Path or URL to the OpenAPI document")
	cmd.Flags().StringVar(&mcpConfig, "mcp-config", "", "Path to the MCP server config file")
	cmd.Flags().StringVar(&output, "output", "", "Output directory")
	cmd.Flags().StringVar(&module, "module", "", "Go module path for the generated project")
	cmd.Flags().StringVar(&appName, "app", "", "Binary/root command name for the generated project")
	cmd.Flags().StringVar(&appVersion, "app-version", "", "Version for the generated CLI project")
	cmd.Flags().StringVar(&configPath, "config", "", "Path to opencli YAML config")
	cmd.Flags().StringVar(&target, "target", "go", "Generation target: go or rust")
	cmd.Flags().StringVar(&skillLang, "skill-lang", "en", "Generated skill language: en or zh")
	cmd.Flags().StringVar(&auth, "auth", "", "Generated auth mode: token, api_key, ak_sk, oauth2, or none (default token)")
	cmd.Flags().StringVar(&signer, "signer", "", "AK/SK signer profile, for example supplier_edi")
	cmd.Flags().StringVar(&runtimeConfigPath, "runtime-config", "", "Runtime YAML metadata; seals OPENCLI_AUTH_TOKEN, OPENCLI_API_KEY, or OPENCLI_OAUTH_CLIENT_SECRET for credential modes; authorization_code uses no build-time secret")
	_ = cmd.MarkFlagRequired("output")
	_ = cmd.MarkFlagRequired("module")
	_ = cmd.MarkFlagRequired("app")
	return cmd
}

func validateGenerateSources(input, mcpConfig string) error {
	hasInput := strings.TrimSpace(input) != ""
	hasMCPConfig := strings.TrimSpace(mcpConfig) != ""
	if hasInput == hasMCPConfig {
		return fmt.Errorf("exactly one of --input or --mcp-config is required")
	}
	return nil
}

func RunGenerate(opts GenerateOptions) error {
	if err := validateGenerateSources(opts.Input, opts.MCPConfig); err != nil {
		return err
	}

	cfg, err := configgen.Load(strings.TrimSpace(opts.ConfigPath))
	if err != nil {
		return err
	}
	if trimmed := strings.TrimSpace(opts.AppVersion); trimmed != "" {
		cfg.App.Version = trimmed
	}
	auth := resolveAuth(opts.Auth, cfg)
	if auth == model.AuthTypeAPIKey && strings.TrimSpace(opts.RuntimeConfigPath) == "" {
		return fmt.Errorf("--auth api_key requires --runtime-config to declare the API-key header")
	}
	if auth == model.AuthTypeOAuth2 && strings.TrimSpace(opts.RuntimeConfigPath) == "" {
		return fmt.Errorf("--auth oauth2 requires --runtime-config to declare OAuth2 settings")
	}
	signer := resolveSigner(opts.Signer, cfg)
	signerConfig, err := resolveSignerConfig(auth, signer, cfg)
	if err != nil {
		return err
	}

	var doc openapi.Document
	if strings.TrimSpace(opts.MCPConfig) != "" {
		doc, err = mcp.DiscoverDocument(strings.TrimSpace(opts.MCPConfig))
		if err != nil {
			return err
		}
	} else {
		raw, err := loaders.Load(strings.TrimSpace(opts.Input))
		if err != nil {
			return err
		}

		doc, err = openapi.Parse(raw)
		if err != nil {
			return err
		}
	}

	oauth2RuntimeDefaults := oauth2Defaults(doc)
	var runtimeBundle *runtimeconfig.Bundle
	if path := strings.TrimSpace(opts.RuntimeConfigPath); path != "" {
		bundle, err := runtimeconfig.LoadAndSeal(path, runtimeconfig.SealOptions{
			AuthMode: auth,
			OAuth2:   oauth2RuntimeDefaults,
		})
		if err != nil {
			return err
		}
		runtimeBundle = &bundle
	}
	if auth == model.AuthTypeOAuth2 {
		tokenURL := oauth2RuntimeDefaults.TokenURL
		if runtimeBundle != nil && strings.TrimSpace(runtimeBundle.OAuth2TokenURL) != "" {
			tokenURL = runtimeBundle.OAuth2TokenURL
		}
		doc = withoutOAuthTokenOperation(doc, tokenURL)
	}
	plan := planner.Build(doc, cfg)
	plan.Name = strings.TrimSpace(opts.AppName)
	plan.Auth.Type = auth
	plan.Auth.SignerProfile = signerConfig.Profile
	plan.Auth.Signer = signerConfig
	return render.ProjectWithOptions(strings.TrimSpace(opts.Output), strings.TrimSpace(opts.Module), plan, render.ProjectOptions{
		Target:        strings.TrimSpace(opts.Target),
		SkillLang:     strings.TrimSpace(opts.SkillLang),
		RuntimeBundle: runtimeBundle,
	})
}

func resolveAuth(flag string, cfg configgen.Config) string {
	if trimmed := strings.TrimSpace(flag); trimmed != "" {
		return trimmed
	}
	if configured := strings.TrimSpace(cfg.Auth.Type); configured != "" {
		return configured
	}
	return model.AuthTypeToken
}

func resolveSigner(flag string, cfg configgen.Config) string {
	if trimmed := strings.TrimSpace(flag); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(cfg.Auth.Signer.Profile)
}

func validateAuthAndSigner(auth, signer string) error {
	switch auth {
	case "", model.AuthTypeToken, model.AuthTypeAPIKey, model.AuthTypeAKSK, model.AuthTypeOAuth2, model.AuthTypeNone:
	default:
		return fmt.Errorf("unsupported auth %q: expected token, api_key, ak_sk, oauth2, or none", auth)
	}
	if strings.TrimSpace(signer) != "" && auth != model.AuthTypeAKSK {
		return fmt.Errorf("--signer requires --auth ak_sk")
	}
	return nil
}

func withoutOAuthTokenOperation(doc openapi.Document, tokenURL string) openapi.Document {
	operations := make([]openapi.Operation, 0, len(doc.Operations))
	for _, operation := range doc.Operations {
		if matchesOAuthTokenOperation(operation, tokenURL) {
			continue
		}
		operations = append(operations, operation)
	}
	doc.Operations = operations
	return doc
}

func matchesOAuthTokenOperation(operation openapi.Operation, tokenURL string) bool {
	parsedTokenURL, err := url.Parse(strings.TrimSpace(tokenURL))
	if err != nil || strings.TrimSpace(parsedTokenURL.Path) == "" || !strings.EqualFold(operation.Method, "POST") || operation.Path != parsedTokenURL.Path {
		return false
	}
	if !parsedTokenURL.IsAbs() || strings.TrimSpace(parsedTokenURL.Host) == "" {
		return true
	}

	hasAbsoluteServer := false
	for _, rawServer := range operation.Servers {
		server, err := url.Parse(strings.TrimSpace(rawServer))
		if err != nil || !server.IsAbs() || strings.TrimSpace(server.Host) == "" {
			continue
		}
		hasAbsoluteServer = true
		if strings.EqualFold(server.Scheme, parsedTokenURL.Scheme) && strings.EqualFold(server.Host, parsedTokenURL.Host) {
			return true
		}
	}
	return !hasAbsoluteServer
}

func oauth2Defaults(doc openapi.Document) runtimeconfig.OAuth2Defaults {
	var matches []openapi.SecurityScheme
	for _, scheme := range doc.SecuritySchemes {
		if strings.EqualFold(strings.TrimSpace(scheme.Type), "oauth2") && strings.TrimSpace(scheme.ClientCredentialsTokenURL) != "" {
			matches = append(matches, scheme)
		}
	}
	if len(matches) != 1 {
		return runtimeconfig.OAuth2Defaults{}
	}
	selected := matches[0]
	defaults := runtimeconfig.OAuth2Defaults{
		GrantType: "client_credentials",
		Scheme:    strings.TrimSpace(selected.Name),
		TokenURL:  strings.TrimSpace(selected.ClientCredentialsTokenURL),
		Placement: "basic",
		Scopes:    append([]string(nil), selected.ClientCredentialsScopes...),
	}
	for _, operation := range doc.Operations {
		if !matchesOAuthTokenOperation(operation, defaults.TokenURL) {
			continue
		}
		queryFields := make(map[string]bool)
		for _, parameter := range operation.Parameters {
			if strings.EqualFold(parameter.In, "query") {
				queryFields[strings.TrimSpace(parameter.Name)] = true
			}
		}
		if queryFields["client_id"] && queryFields["client_secret"] {
			defaults.Placement = "query"
		}
		break
	}
	return defaults
}

func resolveSignerConfig(auth, signer string, cfg configgen.Config) (model.Signer, error) {
	if err := validateAuthAndSigner(auth, signer); err != nil {
		return model.Signer{}, err
	}
	if auth != model.AuthTypeAKSK {
		return model.Signer{}, nil
	}

	resolved := signerFromConfig(cfg.Auth.Signer)
	if signer != "" {
		resolved.Profile = signer
	}
	if resolved.Profile == "" {
		resolved.Profile = model.SignerProfileSupplierEDI
	}
	if resolved.Profile == model.SignerProfileSupplierEDI {
		resolved = withSupplierEDIDefaults(resolved)
		if !supportedSignerAlgorithm(resolved.Algorithm) {
			return model.Signer{}, fmt.Errorf("signer %q uses unsupported algorithm %q: expected %s", resolved.Profile, resolved.Algorithm, model.SignerAlgorithmSHA512Hex)
		}
		return resolved, nil
	}
	if resolved.Algorithm == "" || resolved.CanonicalTemplate == "" || resolved.SignatureHeader == "" {
		return model.Signer{}, fmt.Errorf("custom signer %q requires auth.signer.algorithm, auth.signer.canonical.template, and auth.signer.headers.signature", resolved.Profile)
	}
	if !supportedSignerAlgorithm(resolved.Algorithm) {
		return model.Signer{}, fmt.Errorf("custom signer %q uses unsupported algorithm %q: expected %s", resolved.Profile, resolved.Algorithm, model.SignerAlgorithmSHA512Hex)
	}
	return withSignerDefaults(resolved), nil
}

func supportedSignerAlgorithm(value string) bool {
	return strings.TrimSpace(value) == model.SignerAlgorithmSHA512Hex
}

func signerFromConfig(cfg configgen.SignerConfig) model.Signer {
	return model.Signer{
		Profile:           strings.TrimSpace(cfg.Profile),
		Algorithm:         strings.TrimSpace(cfg.Algorithm),
		AccessKeyHeader:   strings.TrimSpace(cfg.Headers.AccessKey),
		SignatureHeader:   strings.TrimSpace(cfg.Headers.Signature),
		TimestampHeader:   strings.TrimSpace(cfg.Headers.Timestamp),
		NonceHeader:       strings.TrimSpace(cfg.Headers.Nonce),
		PathStripPrefix:   strings.TrimSpace(cfg.Path.StripPrefix),
		BodyOrder:         strings.TrimSpace(cfg.Body.Order),
		CanonicalTemplate: strings.TrimSpace(cfg.Canonical.Template),
	}
}

func withSupplierEDIDefaults(signer model.Signer) model.Signer {
	signer = withSignerDefaults(signer)
	if signer.Algorithm == "" {
		signer.Algorithm = model.SignerAlgorithmSHA512Hex
	}
	if signer.PathStripPrefix == "" {
		signer.PathStripPrefix = "/api-apply"
	}
	if signer.BodyOrder == "" {
		signer.BodyOrder = "spec"
	}
	if signer.CanonicalTemplate == "" {
		signer.CanonicalTemplate = "method={method}&path={path}&appKey={access_key}&appSecret={secret_key}&timestamp={timestamp}&nonce={nonce}&jsonBody={json_body}"
	}
	return signer
}

func withSignerDefaults(signer model.Signer) model.Signer {
	if signer.AccessKeyHeader == "" {
		signer.AccessKeyHeader = "appKey"
	}
	if signer.SignatureHeader == "" {
		signer.SignatureHeader = "sign"
	}
	if signer.TimestampHeader == "" {
		signer.TimestampHeader = "timestamp"
	}
	if signer.NonceHeader == "" {
		signer.NonceHeader = "nonce"
	}
	if signer.BodyOrder == "" {
		signer.BodyOrder = "spec"
	}
	return signer
}
