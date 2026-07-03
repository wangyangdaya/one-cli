package app

import (
	"fmt"
	"strings"

	"one-cli/internal/configgen"
	"one-cli/internal/loaders"
	"one-cli/internal/mcp"
	"one-cli/internal/model"
	"one-cli/internal/openapi"
	outjson "one-cli/internal/output"
	"one-cli/internal/planner"
	"one-cli/internal/render"

	"github.com/spf13/cobra"
)

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

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate a Go CLI project from Swagger/OpenAPI or MCP",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := RunGenerateWithVersionSkillLangAuthAndSigner(input, mcpConfig, output, module, appName, appVersion, configPath, skillLang, auth, signer, target); err != nil {
				return err
			}
			if JSONEnabled() {
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
	cmd.Flags().StringVar(&auth, "auth", "", "Generated auth mode: token or ak_sk")
	cmd.Flags().StringVar(&signer, "signer", "", "AK/SK signer profile, for example supplier_edi")
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

func RunGenerate(input, mcpConfig, output, module, appName, configPath string, targets ...string) error {
	return RunGenerateWithVersion(input, mcpConfig, output, module, appName, "", configPath, targets...)
}

func RunGenerateWithVersion(input, mcpConfig, output, module, appName, appVersion, configPath string, targets ...string) error {
	return RunGenerateWithVersionAndSkillLang(input, mcpConfig, output, module, appName, appVersion, configPath, "en", targets...)
}

func RunGenerateWithVersionAndSkillLang(input, mcpConfig, output, module, appName, appVersion, configPath, skillLang string, targets ...string) error {
	return RunGenerateWithVersionSkillLangAndAuth(input, mcpConfig, output, module, appName, appVersion, configPath, skillLang, "", targets...)
}

func RunGenerateWithVersionSkillLangAndAuth(input, mcpConfig, output, module, appName, appVersion, configPath, skillLang, auth string, targets ...string) error {
	return RunGenerateWithVersionSkillLangAuthAndSigner(input, mcpConfig, output, module, appName, appVersion, configPath, skillLang, auth, "", targets...)
}

func RunGenerateWithVersionSkillLangAuthAndSigner(input, mcpConfig, output, module, appName, appVersion, configPath, skillLang, auth, signer string, targets ...string) error {
	if err := validateGenerateSources(input, mcpConfig); err != nil {
		return err
	}

	cfg, err := configgen.Load(strings.TrimSpace(configPath))
	if err != nil {
		return err
	}
	if trimmed := strings.TrimSpace(appVersion); trimmed != "" {
		cfg.App.Version = trimmed
	}
	auth = resolveAuth(auth, cfg)
	signer = resolveSigner(signer, cfg)
	signerConfig, err := resolveSignerConfig(auth, signer, cfg)
	if err != nil {
		return err
	}

	var doc openapi.Document
	if strings.TrimSpace(mcpConfig) != "" {
		doc, err = mcp.DiscoverDocument(strings.TrimSpace(mcpConfig))
		if err != nil {
			return err
		}
	} else {
		raw, err := loaders.Load(strings.TrimSpace(input))
		if err != nil {
			return err
		}

		doc, err = openapi.Parse(raw)
		if err != nil {
			return err
		}
	}

	plan := planner.Build(doc, cfg)
	plan.Name = strings.TrimSpace(appName)
	plan.Auth.Type = auth
	plan.Auth.SignerProfile = signerConfig.Profile
	plan.Auth.Signer = signerConfig
	target := "go"
	if len(targets) > 0 {
		target = strings.TrimSpace(targets[0])
	}
	return render.Project(strings.TrimSpace(output), strings.TrimSpace(module), plan, target, strings.TrimSpace(skillLang))
}

func resolveAuth(flag string, cfg configgen.Config) string {
	if trimmed := strings.TrimSpace(flag); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(cfg.Auth.Type)
}

func resolveSigner(flag string, cfg configgen.Config) string {
	if trimmed := strings.TrimSpace(flag); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(cfg.Auth.Signer.Profile)
}

func validateAuthAndSigner(auth, signer string) error {
	switch auth {
	case "", "token", "ak_sk":
	default:
		return fmt.Errorf("unsupported auth %q: expected token or ak_sk", auth)
	}
	if strings.TrimSpace(signer) != "" && auth != "ak_sk" {
		return fmt.Errorf("--signer requires --auth ak_sk")
	}
	if auth != "ak_sk" {
		return nil
	}
	return nil
}

func resolveSignerConfig(auth, signer string, cfg configgen.Config) (model.Signer, error) {
	if err := validateAuthAndSigner(auth, signer); err != nil {
		return model.Signer{}, err
	}
	if auth != "ak_sk" {
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
		return withSupplierEDIDefaults(resolved), nil
	}
	if resolved.Algorithm == "" || resolved.CanonicalTemplate == "" || resolved.SignatureHeader == "" {
		return model.Signer{}, fmt.Errorf("custom signer %q requires auth.signer.algorithm, auth.signer.canonical.template, and auth.signer.headers.signature", resolved.Profile)
	}
	return withSignerDefaults(resolved), nil
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
		signer.Algorithm = "sha512_hex"
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
