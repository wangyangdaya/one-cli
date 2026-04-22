package integration_test

// TestGenerateExamplesGoOpenAPI generates a Go CLI from examples/openapi.json into
// tmp/openapi-go, builds it with `go build`, and verifies the binary runs.
//
// TestGenerateExamplesGoMCP generates a Go CLI from examples/quark.json (live MCP
// server) into tmp/quark-go, builds it, and verifies the binary runs.
//
// TestGenerateExamplesRustOpenAPI generates a Rust CLI from examples/openapi.json
// into tmp/openapi-rust and verifies `cargo build` succeeds.
//
// TestGenerateExamplesRustMCP generates a Rust CLI from examples/quark.json into
// tmp/quark-rust and verifies `cargo build` succeeds.
//
// All tests write output under tmp/ (relative to the workspace root) so the
// artefacts survive the test run for manual inspection.  The directory is
// created if it does not exist and is NOT cleaned up automatically.

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"one-cli/internal/app"
)

// workspaceRoot returns the repository root (two levels above tests/integration).
func workspaceRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..")
}

// tmpDir returns <workspace>/tmp/<name>, creating it if necessary.
func tmpDir(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(workspaceRoot(t), "tmp", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create tmp dir %s: %v", dir, err)
	}
	return dir
}

// buildGoCLI runs `go build -o <binary> ./cmd/<app>` inside dir.
// Returns the path to the compiled binary.
func buildGoCLI(t *testing.T, dir, appName string) string {
	t.Helper()
	binaryName := appName
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(dir, binaryName)
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/"+appName)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GOCACHE="+filepath.Join(dir, ".gocache"),
		"GOTOOLCHAIN=local",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}
	return binaryPath
}

// runBinary executes the binary with the given args and returns combined output.
// It does NOT fail the test on non-zero exit — callers decide what to assert.
func runBinary(t *testing.T, binary string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(binary, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// ── Go + OpenAPI ──────────────────────────────────────────────────────────────

func TestGenerateExamplesGoOpenAPI(t *testing.T) {
	root := workspaceRoot(t)
	outDir := tmpDir(t, "openapi-go")

	inputPath := filepath.Join(root, "examples", "openapi.json")
	if err := app.RunGenerate(inputPath, "", outDir, "github.com/acme/openapi-cli", "openapi-cli", ""); err != nil {
		t.Fatalf("generate go from openapi.json: %v", err)
	}

	// Verify expected files exist.
	for _, rel := range []string{
		"cmd/openapi-cli/main.go",
		"internal/auth/command.go",
		"internal/auth/service.go",
		"internal/auth/types.go",
		"skills/auth/SKILL.md",
		"README.md",
		"go.mod",
		"go.sum",
	} {
		if _, err := os.Stat(filepath.Join(outDir, rel)); err != nil {
			t.Errorf("missing generated file %s: %v", rel, err)
		}
	}

	// Build the binary.
	binary := buildGoCLI(t, outDir, "openapi-cli")

	// --help must exit 0 and mention the app name.
	out, err := runBinary(t, binary, "--help")
	if err != nil {
		t.Fatalf("binary --help failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "openapi-cli") {
		t.Errorf("--help output missing app name:\n%s", out)
	}

	// `auth --help` must list the generated sub-commands.
	out, err = runBinary(t, binary, "auth", "--help")
	if err != nil {
		t.Fatalf("binary auth --help failed: %v\noutput: %s", err, out)
	}
	for _, want := range []string{"login", "register"} {
		if !strings.Contains(out, want) {
			t.Errorf("auth --help missing sub-command %q:\n%s", want, out)
		}
	}

	t.Logf("Go OpenAPI binary: %s", binary)
}

// ── Go + MCP (quark.json) ─────────────────────────────────────────────────────

func TestGenerateExamplesGoMCP(t *testing.T) {
	root := workspaceRoot(t)
	outDir := tmpDir(t, "quark-go")

	configPath := filepath.Join(root, "examples", "quark.json")
	if err := app.RunGenerate("", configPath, outDir, "github.com/acme/quark-cli", "quark-cli", ""); err != nil {
		// The quark.json points to a live external MCP server.  If discovery
		// fails (network unavailable, auth expired, etc.) we skip rather than
		// fail so CI stays green in offline environments.
		t.Skipf("generate go from quark.json skipped (MCP discovery failed): %v", err)
	}

	// Verify expected files exist.
	for _, rel := range []string{
		"cmd/quark-cli/main.go",
		"README.md",
		"go.mod",
		"go.sum",
	} {
		if _, err := os.Stat(filepath.Join(outDir, rel)); err != nil {
			t.Errorf("missing generated file %s: %v", rel, err)
		}
	}

	// Build the binary.
	binary := buildGoCLI(t, outDir, "quark-cli")

	// --help must exit 0.
	out, err := runBinary(t, binary, "--help")
	if err != nil {
		t.Fatalf("binary --help failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "quark-cli") {
		t.Errorf("--help output missing app name:\n%s", out)
	}

	t.Logf("Go MCP binary: %s", binary)
}

// ── Rust + OpenAPI ────────────────────────────────────────────────────────────

func TestGenerateExamplesRustOpenAPI(t *testing.T) {
	if _, err := exec.LookPath("cargo"); err != nil {
		t.Skip("cargo not installed")
	}

	root := workspaceRoot(t)
	outDir := tmpDir(t, "openapi-rust")

	inputPath := filepath.Join(root, "examples", "openapi.json")
	if err := app.RunGenerate(inputPath, "", outDir, "openapi-cli", "openapi-cli", "", "rust"); err != nil {
		t.Fatalf("generate rust from openapi.json: %v", err)
	}

	// Verify expected files exist.
	for _, rel := range []string{
		"Cargo.toml",
		"README.md",
		"src/main.rs",
		"src/cli.rs",
		"src/client.rs",
		"src/commands/mod.rs",
		"src/commands/auth.rs",
	} {
		if _, err := os.Stat(filepath.Join(outDir, rel)); err != nil {
			t.Errorf("missing generated file %s: %v", rel, err)
		}
	}

	commandContent, err := os.ReadFile(filepath.Join(outDir, "src", "commands", "auth.rs"))
	if err != nil {
		t.Fatalf("read generated auth command: %v", err)
	}
	commandText := string(commandContent)
	for _, unwanted := range []string{
		`let mut path = String::from("/nodus/api/v1/auth/login");`,
		`let mut query = Vec::new();`,
	} {
		if strings.Contains(commandText, unwanted) {
			t.Fatalf("generated auth command should avoid unnecessary mutability %q:\n%s", unwanted, commandText)
		}
	}

	clientContent, err := os.ReadFile(filepath.Join(outDir, "src", "client.rs"))
	if err != nil {
		t.Fatalf("read generated client: %v", err)
	}
	clientText := string(clientContent)
	for _, unwanted := range []string{
		"pub async fn call_mcp_tool(",
		"async fn send_mcp_request(",
		"fn parse_mcp_payload(",
	} {
		if strings.Contains(clientText, unwanted) {
			t.Fatalf("generated OpenAPI client should not include MCP helper %q:\n%s", unwanted, clientText)
		}
	}

	// Build with cargo.
	cmd := exec.Command("cargo", "build")
	cmd.Dir = outDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		output := string(out)
		if strings.Contains(output, "Could not resolve host") ||
			strings.Contains(output, "failed to download from") {
			t.Skipf("cargo build skipped (network restricted):\n%s", output)
		}
		t.Fatalf("cargo build failed: %v\n%s", err, output)
	}

	// Run the debug binary with --help.
	binaryName := "openapi-cli"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(outDir, "target", "debug", binaryName)
	helpOut, err := runBinary(t, binaryPath, "--help")
	if err != nil {
		t.Fatalf("rust binary --help failed: %v\noutput: %s", err, helpOut)
	}
	if !strings.Contains(helpOut, "openapi-cli") {
		t.Errorf("rust --help output missing app name:\n%s", helpOut)
	}

	t.Logf("Rust OpenAPI binary: %s", binaryPath)
}

func TestGenerateExamplesRustLeaveMakeupAvoidsEmptyBodyMutability(t *testing.T) {
	if _, err := exec.LookPath("cargo"); err != nil {
		t.Skip("cargo not installed")
	}

	root := workspaceRoot(t)
	outDir := tmpDir(t, "leave-makeup-rust")

	inputPath := filepath.Join(root, "examples", "leave_makeup.yaml")
	if err := app.RunGenerate(inputPath, "", outDir, "one-ai", "one-ai", "", "rust"); err != nil {
		t.Fatalf("generate rust from leave_makeup.yaml: %v", err)
	}

	for _, rel := range []string{
		"src/commands/attendance.rs",
		"src/commands/leave.rs",
	} {
		content, err := os.ReadFile(filepath.Join(outDir, rel))
		if err != nil {
			t.Fatalf("read generated file %s: %v", rel, err)
		}
		text := string(content)
		for _, unwanted := range []string{
			"let mut payload = Map::new();",
			"let mut has_flag_body = false;",
		} {
			if strings.Contains(text, unwanted) {
				t.Fatalf("generated file %s should avoid empty body mutability %q:\n%s", rel, unwanted, text)
			}
		}
	}
}

// ── Rust + MCP (quark.json) ───────────────────────────────────────────────────

func TestGenerateExamplesRustMCP(t *testing.T) {
	if _, err := exec.LookPath("cargo"); err != nil {
		t.Skip("cargo not installed")
	}

	root := workspaceRoot(t)
	outDir := tmpDir(t, "quark-rust")

	configPath := filepath.Join(root, "examples", "quark.json")
	if err := app.RunGenerate("", configPath, outDir, "quark-cli", "quark-cli", "", "rust"); err != nil {
		t.Skipf("generate rust from quark.json skipped (MCP discovery failed): %v", err)
	}

	// Verify expected files exist.
	for _, rel := range []string{
		"Cargo.toml",
		"README.md",
		"src/main.rs",
		"src/client.rs",
		"src/commands/mod.rs",
	} {
		if _, err := os.Stat(filepath.Join(outDir, rel)); err != nil {
			t.Errorf("missing generated file %s: %v", rel, err)
		}
	}

	// Build with cargo.
	cmd := exec.Command("cargo", "build")
	cmd.Dir = outDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		output := string(out)
		if strings.Contains(output, "Could not resolve host") ||
			strings.Contains(output, "failed to download from") {
			t.Skipf("cargo build skipped (network restricted):\n%s", output)
		}
		t.Fatalf("cargo build failed: %v\n%s", err, output)
	}

	// Run the debug binary with --help.
	binaryName := "quark-cli"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(outDir, "target", "debug", binaryName)
	helpOut, err := runBinary(t, binaryPath, "--help")
	if err != nil {
		t.Fatalf("rust binary --help failed: %v\noutput: %s", err, helpOut)
	}
	if !strings.Contains(helpOut, "quark-cli") {
		t.Errorf("rust --help output missing app name:\n%s", helpOut)
	}

	t.Logf("Rust MCP binary: %s", binaryPath)
}
