package integration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"one-cli/internal/app"
)

func TestGenerateRustOpenAPISmoke(t *testing.T) {
	dir := t.TempDir()
	if err := app.RunGenerate(app.GenerateOptions{
		Input:   filepath.Join("..", "..", "examples", "petstore.yaml"),
		Output:  dir,
		Module:  "petcli",
		AppName: "petcli",
		Target:  "rust",
	}); err != nil {
		t.Fatalf("run rust generate: %v", err)
	}

	for _, rel := range []string{
		"Cargo.toml",
		"README.md",
		"skills/README.md",
		"skills/pet/SKILL.md",
		"skills/pet/README.md",
		"skills/pet/assets/demo-request.json",
		"skills/pet/references/command-routing.md",
		"skills/pet/references/commands.md",
		"skills/pet/references/workflows.md",
		"skills/pet/references/production-checklist.md",
		"src/main.rs",
		"src/cli.rs",
		"src/client.rs",
		"src/output.rs",
		"src/types.rs",
		"src/commands/mod.rs",
		"src/commands/pet.rs",
	} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
	}

	cliContent, err := os.ReadFile(filepath.Join(dir, "src", "cli.rs"))
	if err != nil {
		t.Fatalf("read generated cli.rs: %v", err)
	}
	if !strings.Contains(string(cliContent), `#[arg(short = 'H', long = "header")]`) {
		t.Fatalf("generated Rust cli.rs should bind -H as shorthand for --header:\n%s", cliContent)
	}
	for _, want := range []string{
		`#[command(name = "skills")]`,
		`#[arg(long = "skills-dir", default_value = "skills")]`,
		"enum SkillsCommand",
		"fn run_skills(command: SkillsCommand, skills_dir: &str) -> AppResult<()>",
	} {
		if !strings.Contains(string(cliContent), want) {
			t.Fatalf("generated Rust cli.rs missing skills command fragment %q:\n%s", want, cliContent)
		}
	}

	skillsIndexContent, err := os.ReadFile(filepath.Join(dir, "skills", "README.md"))
	if err != nil {
		t.Fatalf("read generated skills index: %v", err)
	}
	for _, want := range []string{
		"petcli skills list",
		"petcli skills read <skill>",
		"| `pet` |",
	} {
		if !strings.Contains(string(skillsIndexContent), want) {
			t.Fatalf("generated Rust skills index missing %q:\n%s", want, skillsIndexContent)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "skills", "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("single-group Rust project should not generate skills router, got err: %v", err)
	}

	readmeContent, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatalf("read generated Rust README: %v", err)
	}
	readmeText := string(readmeContent)
	for _, want := range []string{
		`cargo run -- <group> <command> -H "ACCESS-STATUS=inner"`,
		`cargo run -- <group> <command> --header "ACCESS-STATUS=inner"`,
	} {
		if !strings.Contains(readmeText, want) {
			t.Fatalf("generated Rust README missing header example %q:\n%s", want, readmeText)
		}
	}
	for _, unwanted := range []string{
		`cargo run -- -H "ACCESS-STATUS=inner" <group> <command>`,
		`cargo run -- --header "ACCESS-STATUS=inner" <group> <command>`,
	} {
		if strings.Contains(readmeText, unwanted) {
			t.Fatalf("generated Rust README should recommend header flags after the leaf command, found %q:\n%s", unwanted, readmeText)
		}
	}

	cargoContent, err := os.ReadFile(filepath.Join(dir, "Cargo.toml"))
	if err != nil {
		t.Fatalf("read Cargo.toml: %v", err)
	}
	cargoText := string(cargoContent)
	for _, want := range []string{
		"[profile.release]",
		"strip = true",
		"lto = true",
		"codegen-units = 1",
		`panic = "abort"`,
	} {
		if !strings.Contains(cargoText, want) {
			t.Fatalf("generated Cargo.toml missing %q:\n%s", want, cargoText)
		}
	}

	tryCargoBuild(t, dir)
}

func TestGenerateRustHTTPTraceIncludesQueryAndHeaders(t *testing.T) {
	dir := t.TempDir()
	if err := app.RunGenerate(app.GenerateOptions{
		Input:   filepath.Join("..", "..", "examples", "petstore.yaml"),
		Output:  dir,
		Module:  "petcli",
		AppName: "petcli",
		Target:  "rust",
	}); err != nil {
		t.Fatalf("run rust generate: %v", err)
	}

	clientContent, err := os.ReadFile(filepath.Join(dir, "src", "client.rs"))
	if err != nil {
		t.Fatalf("read generated client: %v", err)
	}
	clientText := string(clientContent)

	for _, want := range []string{
		"let query_preview = preview_pairs(&query);",
		"let headers_preview = preview_headers(&headers);",
		"[opencli][http] request\\n  method: {method}\\n  url: {url}\\n  query: {query_preview}\\n  headers: {headers_preview}\\n  body: {}",
	} {
		if !strings.Contains(clientText, want) {
			t.Fatalf("generated Rust client missing trace fragment %q:\n%s", want, clientText)
		}
	}
}

func TestGenerateRustIncludesJSONOutputMode(t *testing.T) {
	dir := t.TempDir()
	if err := app.RunGenerate(app.GenerateOptions{
		Input:   filepath.Join("..", "..", "examples", "petstore.yaml"),
		Output:  dir,
		Module:  "petcli",
		AppName: "petcli",
		Target:  "rust",
	}); err != nil {
		t.Fatalf("run rust generate: %v", err)
	}

	for _, check := range []struct {
		path string
		want string
	}{
		{path: filepath.Join("src", "cli.rs"), want: "json: bool"},
		{path: filepath.Join("src", "cli.rs"), want: "crate::output::set_json_enabled(cli.json);"},
		{path: filepath.Join("src", "output.rs"), want: "SuccessEnvelope"},
		{path: filepath.Join("src", "output.rs"), want: "ErrorEnvelope"},
		{path: filepath.Join("src", "commands", "pet.rs"), want: `output::print_output("petcli pet list"`},
	} {
		content, err := os.ReadFile(filepath.Join(dir, check.path))
		if err != nil {
			t.Fatalf("read generated %s: %v", check.path, err)
		}
		if !strings.Contains(string(content), check.want) {
			t.Fatalf("generated %s missing %q:\n%s", check.path, check.want, content)
		}
	}
}

func TestGenerateRustHTTPClientUsesDefaultTimeout(t *testing.T) {
	dir := t.TempDir()
	if err := app.RunGenerate(app.GenerateOptions{
		Input:   filepath.Join("..", "..", "examples", "petstore.yaml"),
		Output:  dir,
		Module:  "petcli",
		AppName: "petcli",
		Target:  "rust",
	}); err != nil {
		t.Fatalf("run rust generate: %v", err)
	}

	clientContent, err := os.ReadFile(filepath.Join(dir, "src", "client.rs"))
	if err != nil {
		t.Fatalf("read generated client: %v", err)
	}
	clientText := string(clientContent)

	for _, want := range []string{
		"const DEFAULT_TIMEOUT_SECS: u64 = 30;",
		"fn new_http_client() -> AppResult<reqwest::Client>",
		".timeout(std::time::Duration::from_secs(DEFAULT_TIMEOUT_SECS))",
		"let client = new_http_client()?;",
	} {
		if !strings.Contains(clientText, want) {
			t.Fatalf("generated Rust client missing timeout fragment %q:\n%s", want, clientText)
		}
	}
}

func TestGenerateRustHTTPFormatsTimeoutErrors(t *testing.T) {
	dir := t.TempDir()
	if err := app.RunGenerate(app.GenerateOptions{
		Input:   filepath.Join("..", "..", "examples", "petstore.yaml"),
		Output:  dir,
		Module:  "petcli",
		AppName: "petcli",
		Target:  "rust",
	}); err != nil {
		t.Fatalf("run rust generate: %v", err)
	}

	clientContent, err := os.ReadFile(filepath.Join(dir, "src", "client.rs"))
	if err != nil {
		t.Fatalf("read generated client: %v", err)
	}
	clientText := string(clientContent)

	for _, want := range []string{
		"let response = match req.send().await",
		"fn format_request_error(err: reqwest::Error) -> String",
		"if err.is_timeout()",
		`format!("request timed out after {DEFAULT_TIMEOUT_SECS}s: {err}")`,
	} {
		if !strings.Contains(clientText, want) {
			t.Fatalf("generated Rust client missing timeout error fragment %q:\n%s", want, clientText)
		}
	}
}

func TestGenerateRustHTTPTraceLogsRequestFailures(t *testing.T) {
	dir := t.TempDir()
	if err := app.RunGenerate(app.GenerateOptions{
		Input:   filepath.Join("..", "..", "examples", "petstore.yaml"),
		Output:  dir,
		Module:  "petcli",
		AppName: "petcli",
		Target:  "rust",
	}); err != nil {
		t.Fatalf("run rust generate: %v", err)
	}

	clientContent, err := os.ReadFile(filepath.Join(dir, "src", "client.rs"))
	if err != nil {
		t.Fatalf("read generated client: %v", err)
	}
	clientText := string(clientContent)

	for _, want := range []string{
		"let method_label = method.as_str().to_string();",
		"let request_url = url.to_string();",
		`trace_log!("[opencli][http] request_failed\n  method: {method_label}\n  url: {request_url}\n  error: {message}")`,
	} {
		if !strings.Contains(clientText, want) {
			t.Fatalf("generated Rust client missing request failure trace fragment %q:\n%s", want, clientText)
		}
	}
}

func TestGenerateRustMCPSmoke(t *testing.T) {
	const sessionID = "session-rust-123"
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Skipf("skipping MCP smoke test in restricted environment: %v", recovered)
		}
	}()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Method string `json:"method"`
			ID     any    `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		switch request.Method {
		case "initialize":
			w.Header().Set("Mcp-Session-Id", sessionID)
			writeSSEJSONRPC(t, w, map[string]any{
				"jsonrpc": "2.0",
				"id":      request.ID,
				"result": map[string]any{
					"protocolVersion": "2025-03-26",
					"capabilities":    map[string]any{},
					"serverInfo":      map[string]any{"name": "fake-search", "version": "1.0.0"},
				},
			})
		case "tools/list":
			writeSSEJSONRPC(t, w, map[string]any{
				"jsonrpc": "2.0",
				"id":      request.ID,
				"result": map[string]any{
					"tools": []any{
						map[string]any{
							"name":        "search_tool",
							"description": "Search content",
							"inputSchema": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"query": map[string]any{"type": "string"},
								},
								"required": []any{"query"},
							},
						},
					},
				},
			})
		default:
			t.Fatalf("unexpected MCP method %q", request.Method)
		}
	}))
	defer server.Close()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "mcp.json")
	if err := os.WriteFile(configPath, []byte(fmt.Sprintf(`{
		"servers": {
			"search": {
				"transport": "streamable_http",
				"url": %q
			}
		}
	}`, server.URL)), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	outDir := filepath.Join(dir, "generated")
	if err := app.RunGenerate(app.GenerateOptions{
		MCPConfig: configPath,
		Output:    outDir,
		Module:    "quark",
		AppName:   "quark",
		Target:    "rust",
	}); err != nil {
		t.Fatalf("run rust MCP generate: %v", err)
	}

	for _, rel := range []string{
		"Cargo.toml",
		"README.md",
		"skills/search/SKILL.md",
		"src/main.rs",
		"src/client.rs",
		"src/commands/mod.rs",
		"src/commands/search.rs",
	} {
		if _, err := os.Stat(filepath.Join(outDir, rel)); err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
	}

	commandContent, err := os.ReadFile(filepath.Join(outDir, "src", "commands", "search.rs"))
	if err != nil {
		t.Fatalf("read generated MCP command: %v", err)
	}
	commandText := string(commandContent)
	for _, want := range []string{
		"call_mcp_tool",
		"search_tool",
	} {
		if !strings.Contains(commandText, want) {
			t.Fatalf("generated MCP command missing %q:\n%s", want, commandText)
		}
	}

	tryCargoBuild(t, outDir)
}

func tryCargoBuild(t *testing.T, dir string) {
	t.Helper()

	if _, err := exec.LookPath("cargo"); err != nil {
		t.Skip("cargo not installed")
	}

	cmd := exec.Command("cargo", "build")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		output := string(out)
		if strings.Contains(output, "Could not resolve host: index.crates.io") ||
			strings.Contains(output, "failed to download from `https://index.crates.io/") {
			t.Skipf("cargo build skipped due to network restrictions:\n%s", output)
		}
		t.Fatalf("cargo build failed: %v\n%s", err, string(out))
	}
}
