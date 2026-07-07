package app

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"

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
	var singleSkill bool
	var buildTargets string

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate a Go CLI project from Swagger/OpenAPI or MCP",
		RunE: func(cmd *cobra.Command, args []string) error {
			autoBuild := cmd.Flags().Changed("build") || singleSkill
			buildPlatform := strings.TrimSpace(buildTargets)
			if autoBuild && len(args) > 0 {
				buildPlatform = strings.Join(args, ",")
			}
			if !autoBuild && len(args) > 0 {
				return fmt.Errorf("unexpected arguments %q: build targets must follow --build", strings.Join(args, " "))
			}
			if autoBuild && buildPlatform == "" {
				buildPlatform = "current"
			}
			if err := RunGenerateWithVersionSkillLangAuthSignerAndBuild(input, mcpConfig, output, module, appName, appVersion, configPath, skillLang, auth, signer, singleSkill, autoBuild, buildPlatform, target); err != nil {
				return err
			}
			if JSONEnabled() {
				selectedTarget := strings.TrimSpace(target)
				if selectedTarget == "" {
					selectedTarget = "go"
				}
				rendered, err := outjson.JSONSuccess(cmd.CommandPath(), "generated project", map[string]string{
					"output":   strings.TrimSpace(output),
					"module":   strings.TrimSpace(module),
					"app":      strings.TrimSpace(appName),
					"target":   selectedTarget,
					"build":    fmt.Sprintf("%t", autoBuild),
					"platform": strings.TrimSpace(buildPlatform),
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
	cmd.Flags().BoolVar(&singleSkill, "single-skill", false, "Generate a single unified skill instead of one per group; builds the current-platform CLI into the skill scripts directory by default")
	cmd.Flags().StringVar(&buildTargets, "build", "", "Build after rendering; optional targets: current, windows, mac-silicon, mac-intel, linux, comma-separated values, or all")
	cmd.Flags().Lookup("build").NoOptDefVal = "current"
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
	return RunGenerateWithVersionSkillLangAuthAndSigner(input, mcpConfig, output, module, appName, appVersion, configPath, skillLang, auth, "", false, targets...)
}

func RunGenerateWithVersionSkillLangAuthAndSigner(input, mcpConfig, output, module, appName, appVersion, configPath, skillLang, auth, signer string, singleSkill bool, targets ...string) error {
	return RunGenerateWithVersionSkillLangAuthSignerAndBuild(input, mcpConfig, output, module, appName, appVersion, configPath, skillLang, auth, signer, singleSkill, false, "current", targets...)
}

func RunGenerateWithVersionSkillLangAuthSignerAndBuild(input, mcpConfig, output, module, appName, appVersion, configPath, skillLang, auth, signer string, singleSkill, autoBuild bool, buildPlatform string, targets ...string) error {
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
	plan.SingleSkill = singleSkill
	plan.Auth.Type = auth
	plan.Auth.SignerProfile = signerConfig.Profile
	plan.Auth.Signer = signerConfig
	target := "go"
	if len(targets) > 0 {
		target = strings.TrimSpace(targets[0])
	}
	outputDir := strings.TrimSpace(output)
	if err := render.Project(outputDir, strings.TrimSpace(module), plan, target, strings.TrimSpace(skillLang)); err != nil {
		return err
	}
	if autoBuild {
		return buildGeneratedProject(outputDir, strings.TrimSpace(module), plan.Name, target, singleSkill, buildPlatform)
	}
	return nil
}

func buildGeneratedProject(outputDir, module, appName, target string, singleSkill bool, platform string) error {
	specs, err := resolveBuildPlatforms(platform)
	if err != nil {
		return err
	}
	multiPlatform := len(specs) > 1
	switch strings.ToLower(strings.TrimSpace(target)) {
	case "", "go":
		for _, spec := range specs {
			if err := buildGeneratedGoProject(outputDir, appName, singleSkill, spec, multiPlatform); err != nil {
				return err
			}
		}
		if multiPlatform {
			return writeMultiPlatformLaunchers(outputDir, appName, singleSkill)
		}
		return nil
	case "rust":
		for _, spec := range specs {
			if err := buildGeneratedRustProject(outputDir, module, appName, singleSkill, spec, multiPlatform); err != nil {
				return err
			}
		}
		if multiPlatform {
			return writeMultiPlatformLaunchers(outputDir, appName, singleSkill)
		}
		return nil
	default:
		return fmt.Errorf("unsupported target %q: expected go or rust", target)
	}
}

type buildPlatformSpec struct {
	Name       string
	GOOS       string
	GOARCH     string
	RustTarget string
	Windows    bool
}

func resolveBuildPlatforms(platform string) ([]buildPlatformSpec, error) {
	parts := splitBuildPlatforms(platform)
	if len(parts) == 0 {
		parts = []string{"current"}
	}
	if len(parts) == 1 && parts[0] == "all" {
		return []buildPlatformSpec{
			{Name: "windows-amd64", GOOS: "windows", GOARCH: "amd64", RustTarget: "x86_64-pc-windows-msvc", Windows: true},
			{Name: "darwin-arm64", GOOS: "darwin", GOARCH: "arm64", RustTarget: "aarch64-apple-darwin"},
			{Name: "darwin-amd64", GOOS: "darwin", GOARCH: "amd64", RustTarget: "x86_64-apple-darwin"},
			{Name: "linux-amd64", GOOS: "linux", GOARCH: "amd64", RustTarget: "x86_64-unknown-linux-gnu"},
		}, nil
	}
	specs := make([]buildPlatformSpec, 0, len(parts))
	seen := make(map[string]bool, len(parts))
	for _, part := range parts {
		if part == "all" {
			return nil, fmt.Errorf("unsupported build platform list %q: all must be used alone", platform)
		}
		spec, err := resolveBuildPlatform(part)
		if err != nil {
			return nil, err
		}
		if seen[spec.Name] {
			continue
		}
		seen[spec.Name] = true
		specs = append(specs, spec)
	}
	return specs, nil
}

func splitBuildPlatforms(platform string) []string {
	raw := strings.FieldsFunc(platform, func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	})
	parts := make([]string, 0, len(raw))
	for _, part := range raw {
		trimmed := strings.ToLower(strings.TrimSpace(part))
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

func resolveBuildPlatform(platform string) (buildPlatformSpec, error) {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "", "current":
		return buildPlatformSpec{Name: runtime.GOOS + "-" + runtime.GOARCH, GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, Windows: runtime.GOOS == "windows"}, nil
	case "windows", "win":
		return buildPlatformSpec{Name: "windows-amd64", GOOS: "windows", GOARCH: "amd64", RustTarget: "x86_64-pc-windows-msvc", Windows: true}, nil
	case "mac-silicon", "darwin-arm64", "mac-arm64":
		return buildPlatformSpec{Name: "darwin-arm64", GOOS: "darwin", GOARCH: "arm64", RustTarget: "aarch64-apple-darwin"}, nil
	case "mac-intel", "darwin-amd64", "mac-amd64":
		return buildPlatformSpec{Name: "darwin-amd64", GOOS: "darwin", GOARCH: "amd64", RustTarget: "x86_64-apple-darwin"}, nil
	case "linux", "linux-amd64":
		return buildPlatformSpec{Name: "linux-amd64", GOOS: "linux", GOARCH: "amd64", RustTarget: "x86_64-unknown-linux-gnu"}, nil
	default:
		return buildPlatformSpec{}, fmt.Errorf("unsupported build platform %q: expected current, windows, mac-silicon, mac-intel, linux, or all", platform)
	}
}

func buildGeneratedGoProject(outputDir, appName string, singleSkill bool, spec buildPlatformSpec, multiPlatform bool) error {
	finalBinary := generatedBinaryPath(outputDir, appName, singleSkill, spec, multiPlatform)
	tempDir := filepath.Join(outputDir, ".opencli-build")
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return fmt.Errorf("prepare generated Go build directory: %w", err)
	}
	absTempDir, err := filepath.Abs(tempDir)
	if err != nil {
		return fmt.Errorf("resolve generated Go build directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	tempBinary := filepath.Join(absTempDir, filepath.Base(finalBinary))
	cmd := exec.Command(goExecutable(), "build", "-mod=mod", "-o", tempBinary, "./cmd/"+strings.TrimSpace(appName))
	cmd.Dir = outputDir
	cmd.Env = generatedGoBuildEnv(absTempDir, spec)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	absFinalBinary, err := filepath.Abs(finalBinary)
	if err != nil {
		return fmt.Errorf("resolve generated Go binary path: %w", err)
	}
	printBuildStart("Go", spec.Name, outputDir, cmd.Args)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build generated Go project: %w", err)
	}
	if spec.Windows {
		_ = os.Remove(finalBinary)
	}
	if err := os.MkdirAll(filepath.Dir(finalBinary), 0o755); err != nil {
		return fmt.Errorf("prepare generated Go script directory: %w", err)
	}
	if err := os.Rename(tempBinary, finalBinary); err != nil {
		return fmt.Errorf("install generated Go binary: %w", err)
	}
	printBuildComplete(absFinalBinary)
	return nil
}

func generatedGoBuildEnv(tempDir string, spec buildPlatformSpec) []string {
	env := os.Environ()
	env = append(env, "GOOS="+spec.GOOS, "GOARCH="+spec.GOARCH, "CGO_ENABLED=0")
	if strings.TrimSpace(os.Getenv("GOCACHE")) == "" {
		env = append(env, "GOCACHE="+filepath.Join(tempDir, "gocache"))
	}
	return env
}

func buildGeneratedRustProject(outputDir, module, appName string, singleSkill bool, spec buildPlatformSpec, multiPlatform bool) error {
	args := []string{"build"}
	if spec.RustTarget != "" {
		args = append(args, "--target", spec.RustTarget)
	}
	cmd := exec.Command("cargo", args...)
	cmd.Dir = outputDir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	sourceDir := filepath.Join(outputDir, "target", "debug")
	if spec.RustTarget != "" {
		sourceDir = filepath.Join(outputDir, "target", spec.RustTarget, "debug")
	}
	binaryPath := filepath.Join(sourceDir, cargoBinaryName(module))
	if spec.Windows {
		binaryPath += ".exe"
	}
	finalBinary := generatedBinaryPath(outputDir, appName, singleSkill, spec, multiPlatform)
	absFinalBinary, err := filepath.Abs(finalBinary)
	if err != nil {
		return fmt.Errorf("resolve generated Rust binary path: %w", err)
	}
	printBuildStart("Rust", spec.Name, outputDir, cmd.Args)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build generated Rust project: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(finalBinary), 0o755); err != nil {
		return fmt.Errorf("prepare generated Rust script directory: %w", err)
	}
	if err := copyFile(binaryPath, finalBinary, 0o755); err != nil {
		return fmt.Errorf("install generated Rust binary: %w", err)
	}
	printBuildComplete(absFinalBinary)
	return nil
}

func generatedBinaryPath(outputDir, appName string, singleSkill bool, spec buildPlatformSpec, multiPlatform bool) string {
	name := strings.TrimSpace(appName)
	if name == "" {
		name = "app"
	}
	binaryName := name
	if multiPlatform {
		binaryName += "-" + spec.Name
	}
	if spec.Windows {
		binaryName += ".exe"
	}
	if !singleSkill {
		return filepath.Join(outputDir, "bin", binaryName)
	}
	return filepath.Join(outputDir, "skills", skillPackageName(name), "scripts", binaryName)
}

func generatedBinaryDir(outputDir, appName string, singleSkill bool) string {
	if !singleSkill {
		return filepath.Join(outputDir, "bin")
	}
	return filepath.Join(outputDir, "skills", skillPackageName(appName), "scripts")
}

func writeMultiPlatformLaunchers(outputDir, appName string, singleSkill bool) error {
	name := strings.TrimSpace(appName)
	if name == "" {
		name = "app"
	}
	dir := generatedBinaryDir(outputDir, name, singleSkill)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("prepare generated launcher directory: %w", err)
	}
	shellLauncher := fmt.Sprintf(`#!/bin/sh
set -eu

DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
OS=$(uname -s)
ARCH=$(uname -m)

case "$OS:$ARCH" in
  Darwin:arm64) BIN="$DIR/%[1]s-darwin-arm64" ;;
  Darwin:x86_64) BIN="$DIR/%[1]s-darwin-amd64" ;;
  Linux:x86_64|Linux:amd64) BIN="$DIR/%[1]s-linux-amd64" ;;
  MINGW*:x86_64|MSYS*:x86_64|CYGWIN*:x86_64) BIN="$DIR/%[1]s-windows-amd64.exe" ;;
  *) echo "unsupported platform: $OS/$ARCH" >&2; exit 1 ;;
esac

exec "$BIN" "$@"
`, name)
	if err := os.WriteFile(filepath.Join(dir, name), []byte(shellLauncher), 0o755); err != nil {
		return fmt.Errorf("write generated shell launcher: %w", err)
	}

	cmdLauncher := fmt.Sprintf(`@echo off
set "DIR=%%~dp0"
set "BIN=%%DIR%%%[1]s-windows-amd64.exe"
if not exist "%%BIN%%" (
  echo unsupported platform or missing binary: %%BIN%% 1>&2
  exit /b 1
)
"%%BIN%%" %%*
exit /b %%ERRORLEVEL%%
`, name)
	if err := os.WriteFile(filepath.Join(dir, name+".cmd"), []byte(cmdLauncher), 0o644); err != nil {
		return fmt.Errorf("write generated Windows launcher: %w", err)
	}
	absShellLauncher, err := filepath.Abs(filepath.Join(dir, name))
	if err != nil {
		absShellLauncher = filepath.Join(dir, name)
	}
	absCmdLauncher, err := filepath.Abs(filepath.Join(dir, name+".cmd"))
	if err != nil {
		absCmdLauncher = filepath.Join(dir, name+".cmd")
	}
	fmt.Fprintf(os.Stderr, "[opencli] Build complete: %s\n", absShellLauncher)
	fmt.Fprintf(os.Stderr, "[opencli] Build complete: %s\n", absCmdLauncher)
	return nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func skillPackageName(name string) string {
	value := strings.TrimSpace(name)
	if value == "" {
		return "skill"
	}
	var builder strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(value) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				builder.WriteRune('-')
				lastDash = true
			}
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "skill"
	}
	return result
}

func printBuildStart(target, platform, outputDir string, args []string) {
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		absOutputDir = outputDir
	}
	fmt.Fprintf(os.Stderr, "[opencli] Building generated %s project for %s\n", target, platform)
	fmt.Fprintf(os.Stderr, "[opencli] Build directory: %s\n", absOutputDir)
	fmt.Fprintf(os.Stderr, "[opencli] Command: %s\n", shellCommand(args))
}

func printBuildComplete(binaryPath string) {
	fmt.Fprintf(os.Stderr, "[opencli] Build complete: %s\n", binaryPath)
}

func shellCommand(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, shellQuote(arg))
	}
	return strings.Join(quoted, " ")
}

func shellQuote(arg string) string {
	if arg == "" {
		return "''"
	}
	if strings.IndexFunc(arg, func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("-_./:=+", r))
	}) < 0 {
		return arg
	}
	return "'" + strings.ReplaceAll(arg, "'", "'\\''") + "'"
}

func cargoBinaryName(module string) string {
	module = strings.TrimSpace(module)
	if module == "" {
		return "generated-cli"
	}
	if idx := strings.LastIndex(module, "/"); idx >= 0 && idx+1 < len(module) {
		module = module[idx+1:]
	}

	var builder strings.Builder
	lastDash := false
	for _, r := range module {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(unicode.ToLower(r))
			lastDash = false
		default:
			if !lastDash {
				builder.WriteRune('-')
				lastDash = true
			}
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "generated-cli"
	}
	return result
}

func goExecutable() string {
	if goroot := runtime.GOROOT(); strings.TrimSpace(goroot) != "" {
		candidate := filepath.Join(goroot, "bin", "go")
		if runtime.GOOS == "windows" {
			candidate += ".exe"
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return "go"
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
