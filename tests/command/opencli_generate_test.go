package command_test

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"one-cli/internal/app"
)

func TestGenerateCommand(t *testing.T) {
	dir := t.TempDir()
	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{
		"generate",
		"--input", filepath.Join("..", "..", "examples", "petstore.yaml"),
		"--output", dir,
		"--module", "github.com/acme/generated",
		"--app", "petcli",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute generate: %v", err)
	}
}

func TestGenerateCommandJSONOutput(t *testing.T) {
	dir := t.TempDir()
	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{
		"--json",
		"generate",
		"--input", filepath.Join("..", "..", "examples", "petstore.yaml"),
		"--output", dir,
		"--module", "github.com/acme/generated",
		"--app", "petcli",
	})

	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute generate: %v output=%s", err, out.String())
	}

	var envelope struct {
		OK      bool   `json:"ok"`
		Command string `json:"command"`
		Message string `json:"message"`
		Data    struct {
			Output string `json:"output"`
			Module string `json:"module"`
			App    string `json:"app"`
			Target string `json:"target"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(out.String()), &envelope); err != nil {
		t.Fatalf("expected JSON output, got error %v output=%s", err, out.String())
	}
	if !envelope.OK || envelope.Command != "opencli generate" || envelope.Data.Output != dir || envelope.Data.Target != "go" {
		t.Fatalf("unexpected JSON envelope: %+v", envelope)
	}
}

func TestGenerateCommandWithSimpleJSONBodySpec(t *testing.T) {
	dir := t.TempDir()
	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{
		"generate",
		"--input", filepath.Join("..", "..", "examples", "openapi.json"),
		"--output", dir,
		"--module", "github.com/acme/generated",
		"--app", "openapi-cli",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute generate: %v", err)
	}
}

func TestGenerateCommandAcceptsChineseSkillLanguage(t *testing.T) {
	dir := t.TempDir()
	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{
		"generate",
		"--input", filepath.Join("..", "..", "examples", "petstore.yaml"),
		"--output", dir,
		"--module", "github.com/acme/generated",
		"--app", "petcli",
		"--skill-lang", "zh",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute generate: %v", err)
	}

	skillContent, err := os.ReadFile(filepath.Join(dir, "skills", "pet", "SKILL.md"))
	if err != nil {
		t.Fatalf("read generated skill: %v", err)
	}
	if !strings.Contains(string(skillContent), "## 包文件") {
		t.Fatalf("generated skill is not Chinese:\n%s", skillContent)
	}
}

func TestGenerateCommandRequiresExactlyOneSource(t *testing.T) {
	dir := t.TempDir()
	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{
		"generate",
		"--output", dir,
		"--module", "github.com/acme/generated",
		"--app", "generated",
	})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "exactly one of --input or --mcp-config is required") {
		t.Fatalf("expected source selection error, got %v", err)
	}
}

func TestGenerateCommandJSONErrorOutput(t *testing.T) {
	cmd := newGoRunCommand(t, "./cmd/opencli", "--json", "generate", "--output", t.TempDir(), "--module", "github.com/acme/generated", "--app", "generated")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected generate to fail, got output=%s", out)
	}

	var envelope struct {
		OK      bool   `json:"ok"`
		Command string `json:"command"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	line := bytes.SplitN(bytes.TrimSpace(out), []byte("\n"), 2)[0]
	if err := json.Unmarshal(line, &envelope); err != nil {
		t.Fatalf("expected JSON error output, got error %v output=%s", err, out)
	}
	if envelope.OK || envelope.Error.Code != "command_error" || !strings.Contains(envelope.Error.Message, "exactly one of --input or --mcp-config is required") {
		t.Fatalf("unexpected JSON error envelope: %+v", envelope)
	}
}

func TestGenerateCommandRejectsMixedSources(t *testing.T) {
	dir := t.TempDir()
	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{
		"generate",
		"--input", filepath.Join("..", "..", "examples", "petstore.yaml"),
		"--mcp-config", filepath.Join("testdata", "mcp.json"),
		"--output", dir,
		"--module", "github.com/acme/generated",
		"--app", "generated",
	})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "exactly one of --input or --mcp-config is required") {
		t.Fatalf("expected mixed source error, got %v", err)
	}
}

func TestGenerateCommandAcceptsRustTargetWithOpenAPI(t *testing.T) {
	dir := t.TempDir()
	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{
		"generate",
		"--target", "rust",
		"--input", filepath.Join("..", "..", "examples", "petstore.yaml"),
		"--output", dir,
		"--module", "petcli",
		"--app", "petcli",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute rust generate: %v", err)
	}
}

func TestGenerateCommandAcceptsAKSKAuthForGo(t *testing.T) {
	dir := t.TempDir()
	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{
		"generate",
		"--auth", "ak_sk",
		"--signer", "supplier_edi",
		"--input", filepath.Join("..", "..", "examples", "petstore.yaml"),
		"--output", dir,
		"--module", "github.com/acme/generated",
		"--app", "petcli",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute generate: %v", err)
	}

	authContent, err := os.ReadFile(filepath.Join(dir, "internal", "auth", "aksk.go"))
	if err != nil {
		t.Fatalf("read generated aksk auth: %v", err)
	}
	content := string(authContent)
	for _, want := range []string{"OPENCLI_AK", "OPENCLI_SK", "supplier_edi", "Signer interface", "appKey", "sign", "timestamp", "nonce"} {
		if !strings.Contains(content, want) {
			t.Fatalf("generated AK/SK auth missing %q:\n%s", want, content)
		}
	}

	skillContent, err := os.ReadFile(filepath.Join(dir, "skills", "pet", "SKILL.md"))
	if err != nil {
		t.Fatalf("read generated skill: %v", err)
	}
	skillText := string(skillContent)
	for _, want := range []string{"OPENCLI_AK", "OPENCLI_SK", "AK/SK", "appKey", "sign", "timestamp", "nonce"} {
		if !strings.Contains(skillText, want) {
			t.Fatalf("generated AK/SK skill missing %q:\n%s", want, skillText)
		}
	}
}

func TestGenerateCommandAcceptsAKSKAuthForRust(t *testing.T) {
	dir := t.TempDir()
	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{
		"generate",
		"--target", "rust",
		"--auth", "ak_sk",
		"--signer", "supplier_edi",
		"--input", filepath.Join("..", "..", "examples", "petstore.yaml"),
		"--output", dir,
		"--module", "petcli",
		"--app", "petcli",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute rust generate: %v", err)
	}

	authContent, err := os.ReadFile(filepath.Join(dir, "src", "auth.rs"))
	if err != nil {
		t.Fatalf("read generated aksk auth: %v", err)
	}
	content := string(authContent)
	for _, want := range []string{"OPENCLI_AK", "OPENCLI_SK", "SIGNER_PROFILE", "supplier_edi", "appKey", "sign", "timestamp", "nonce"} {
		if !strings.Contains(content, want) {
			t.Fatalf("generated AK/SK auth missing %q:\n%s", want, content)
		}
	}

	skillContent, err := os.ReadFile(filepath.Join(dir, "skills", "pet", "SKILL.md"))
	if err != nil {
		t.Fatalf("read generated skill: %v", err)
	}
	skillText := string(skillContent)
	for _, want := range []string{"OPENCLI_AK", "OPENCLI_SK", "AK/SK", "appKey", "sign", "timestamp", "nonce"} {
		if !strings.Contains(skillText, want) {
			t.Fatalf("generated AK/SK skill missing %q:\n%s", want, skillText)
		}
	}
}

func TestGenerateCommandAcceptsNoAuthForGo(t *testing.T) {
	dir := t.TempDir()
	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{
		"generate",
		"--auth", "none",
		"--input", filepath.Join("..", "..", "examples", "petstore.yaml"),
		"--output", dir,
		"--module", "github.com/acme/generated",
		"--app", "petcli",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute generate: %v", err)
	}

	serviceContent, err := os.ReadFile(filepath.Join(dir, "internal", "pet", "service.go"))
	if err != nil {
		t.Fatalf("read generated service: %v", err)
	}
	serviceText := string(serviceContent)
	for _, unwanted := range []string{"applyAuthToken", "OPENCLI_AUTH_TOKEN", "Authorization\", \"Bearer "} {
		if strings.Contains(serviceText, unwanted) {
			t.Fatalf("generated no-auth service should not include %q:\n%s", unwanted, serviceText)
		}
	}

	skillContent, err := os.ReadFile(filepath.Join(dir, "skills", "pet", "SKILL.md"))
	if err != nil {
		t.Fatalf("read generated skill: %v", err)
	}
	skillText := string(skillContent)
	if !strings.Contains(skillText, "No authentication is configured") {
		t.Fatalf("generated no-auth skill should document no authentication:\n%s", skillText)
	}
	if strings.Contains(skillText, "OPENCLI_AUTH_TOKEN") {
		t.Fatalf("generated no-auth skill should not require OPENCLI_AUTH_TOKEN:\n%s", skillText)
	}

	readmeContent, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatalf("read generated readme: %v", err)
	}
	readmeText := string(readmeContent)
	for _, want := range []string{
		"## Auth",
		"Mode: `none`",
		"This CLI does not inject authentication headers.",
		`./bin/petcli auth me --header "ACCESS-STATUS=inner"`,
	} {
		if !strings.Contains(readmeText, want) {
			t.Fatalf("generated no-auth README missing %q:\n%s", want, readmeText)
		}
	}
	if strings.Contains(readmeText, `./bin/petcli --header "ACCESS-STATUS=inner" auth me`) {
		t.Fatalf("generated no-auth README should recommend header flags after the leaf command:\n%s", readmeText)
	}
	if strings.Contains(readmeText, "OPENCLI_AUTH_TOKEN") {
		t.Fatalf("generated no-auth README should not require OPENCLI_AUTH_TOKEN:\n%s", readmeText)
	}
}

func TestGenerateCommandAcceptsNoAuthForRust(t *testing.T) {
	dir := t.TempDir()
	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{
		"generate",
		"--target", "rust",
		"--auth", "none",
		"--input", filepath.Join("..", "..", "examples", "petstore.yaml"),
		"--output", dir,
		"--module", "petcli",
		"--app", "petcli",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute rust generate: %v", err)
	}

	clientContent, err := os.ReadFile(filepath.Join(dir, "src", "client.rs"))
	if err != nil {
		t.Fatalf("read generated client: %v", err)
	}
	clientText := string(clientContent)
	for _, unwanted := range []string{"OPENCLI_AUTH_TOKEN", "Authorization\", format!(\"Bearer"} {
		if strings.Contains(clientText, unwanted) {
			t.Fatalf("generated no-auth client should not include %q:\n%s", unwanted, clientText)
		}
	}

	skillContent, err := os.ReadFile(filepath.Join(dir, "skills", "pet", "SKILL.md"))
	if err != nil {
		t.Fatalf("read generated skill: %v", err)
	}
	skillText := string(skillContent)
	if !strings.Contains(skillText, "No authentication is configured") {
		t.Fatalf("generated no-auth skill should document no authentication:\n%s", skillText)
	}
	if strings.Contains(skillText, "OPENCLI_AUTH_TOKEN") {
		t.Fatalf("generated no-auth skill should not require OPENCLI_AUTH_TOKEN:\n%s", skillText)
	}

	readmeContent, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatalf("read generated readme: %v", err)
	}
	readmeText := string(readmeContent)
	for _, want := range []string{
		"## Auth",
		"Mode: `none`",
		"This CLI does not inject authentication headers.",
		`cargo run -- <group> <command> --header "ACCESS-STATUS=inner"`,
	} {
		if !strings.Contains(readmeText, want) {
			t.Fatalf("generated no-auth README missing %q:\n%s", want, readmeText)
		}
	}
	if strings.Contains(readmeText, `cargo run -- --header "ACCESS-STATUS=inner" <group> <command>`) {
		t.Fatalf("generated no-auth README should recommend header flags after the leaf command:\n%s", readmeText)
	}
	if strings.Contains(readmeText, "OPENCLI_AUTH_TOKEN") {
		t.Fatalf("generated no-auth README should not require OPENCLI_AUTH_TOKEN:\n%s", readmeText)
	}
}

func TestGenerateCommandSupplierAKSKSkillDocumentsBodyExampleFields(t *testing.T) {
	dir := t.TempDir()
	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{
		"generate",
		"--input", filepath.Join("..", "..", "examples", "supplier.json"),
		"--config", filepath.Join("..", "..", "examples", "supplier.opencli.yaml"),
		"--output", dir,
		"--module", "github.com/acme/supplier-cli",
		"--app", "supplier-cli",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute supplier generate: %v", err)
	}

	skillContent, err := os.ReadFile(filepath.Join(dir, "skills", "supplier", "SKILL.md"))
	if err != nil {
		t.Fatalf("read generated skill: %v", err)
	}
	skillText := string(skillContent)
	for _, want := range []string{"--date", "--pageSize", "--pageNum", "--isForce"} {
		if !strings.Contains(skillText, want) {
			t.Fatalf("generated supplier skill missing body flag %q:\n%s", want, skillText)
		}
	}
	for _, unwanted := range []string{`--header "appKey: <value>"`, `--header "sign: <value>"`, `--header "timestamp: <value>"`, `--header "nonce: <value>"`} {
		if strings.Contains(skillText, unwanted) {
			t.Fatalf("generated supplier AK/SK skill should not ask for auth header %q:\n%s", unwanted, skillText)
		}
	}
}

func TestGenerateCommandHTTPGoOutputIsGofmtClean(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		config string
		module string
		app    string
	}{
		{
			name:   "token",
			input:  filepath.Join("..", "..", "examples", "petstore.yaml"),
			module: "github.com/acme/petcli",
			app:    "petcli",
		},
		{
			name:   "aksk",
			input:  filepath.Join("..", "..", "examples", "supplier.json"),
			config: filepath.Join("..", "..", "examples", "supplier.opencli.yaml"),
			module: "github.com/acme/supplier-cli",
			app:    "supplier-cli",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			args := []string{
				"generate",
				"--input", tc.input,
				"--output", dir,
				"--module", tc.module,
				"--app", tc.app,
			}
			if tc.config != "" {
				args = append(args, "--config", tc.config)
			}
			cmd := app.NewRootCommand()
			cmd.SetArgs(args)

			if err := cmd.Execute(); err != nil {
				t.Fatalf("execute generate: %v", err)
			}
			assertGoFilesGofmtClean(t, dir)
		})
	}
}

func TestGenerateCommandSupplierAKSKMatchesGatewaySigningExample(t *testing.T) {
	dir := t.TempDir()
	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{
		"generate",
		"--input", filepath.Join("..", "..", "examples", "supplier.json"),
		"--config", filepath.Join("..", "..", "examples", "supplier.opencli.yaml"),
		"--output", dir,
		"--module", "github.com/acme/supplier-cli",
		"--app", "supplier-cli",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute supplier generate: %v", err)
	}

	authTest := `package auth

import "testing"

func TestSupplierSignMatchesGatewayExample(t *testing.T) {
	body := []byte(` + "`" + `{"date":"2026-04-21","pageSize":25,"pageNum":25,"isForce":true}` + "`" + `)
	got, err := sign("0AA14a7757576434d7", "cc637052aebe4687ac1b5c5d4a509485", "POST", "/api-apply/v2/get/supplierDelState", body, "1783046918568", "18bea7b9b71c1778")
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	want := "fc748c543a3645b5c07c104f6ba92b83945466bf74548f52416c2afa48495993e5fef62e798f5646442539251f51c8c98d83d4fec4743b77efa94d7192f738e8"
	if got != want {
		t.Fatalf("sign mismatch: got %s want %s", got, want)
	}
}

func TestSupplierSignMatchesOfficialDocumentExample(t *testing.T) {
	body := []byte(` + "`" + `{"batchNo":"fasdfsd202506090084","total":1,"pageSize":1,"pageNum":1,"list":[{"supplierCode":"TS","supplierName":"测试","plantId":"测试1","plantName":"测试2","vendorProductNo":"201034AA","vendorProductName":"电动转向管","cheryProductNo":"20104AA","manufactureNum":"2000.00","manufactureInputNum":1764,"actualBeginTime":"2025-06-06 08:27:05","actualEndTime":""}]}` + "`" + `)
	got, err := sign("8e79ac36fcce490", "e1231387b0bd684ac7a27dde792e836785", "POST", "/api-apply/v2/push/supplierProMaterialStock", body, "1747386804000", "1747382356599Hd")
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	want := "594b2a0b2583befc29b34017b2371c5dd1b1721b5d87040fcdb3216a4b2319ad8b71d1e27d797b4fb10646abed8b43643b0b2c6a2e836f08d2ef2a6e13757657"
	if got != want {
		t.Fatalf("sign mismatch: got %s want %s", got, want)
	}
}
`
	if err := os.WriteFile(filepath.Join(dir, "internal", "auth", "aksk_test.go"), []byte(authTest), 0o644); err != nil {
		t.Fatalf("write auth test: %v", err)
	}

	supplierTest := `package supplier

import "testing"

func TestKanbanDeliveryBodyMatchesGatewayExampleOrder(t *testing.T) {
	body, err := buildKanbanDeliveryBody(KanbanDeliveryInput{
		Date: "2026-04-21",
		Pagesize: 25,
		Pagenum: 25,
		Isforce: true,
	})
	if err != nil {
		t.Fatalf("build body: %v", err)
	}
	want := ` + "`" + `{"date":"2026-04-21","pageSize":25,"pageNum":25,"isForce":true}` + "`" + `
	if string(body) != want {
		t.Fatalf("body mismatch: got %s want %s", body, want)
	}
}
`
	if err := os.WriteFile(filepath.Join(dir, "internal", "supplier", "supplier_test.go"), []byte(supplierTest), 0o644); err != nil {
		t.Fatalf("write supplier test: %v", err)
	}

	testCmd := exec.Command("go", "test", "./internal/auth", "./internal/supplier")
	testCmd.Dir = dir
	testCmd.Env = append(os.Environ(), "GOCACHE="+filepath.Join(t.TempDir(), "gocache"))
	if output, err := testCmd.CombinedOutput(); err != nil {
		t.Fatalf("generated supplier tests failed: %v\n%s", err, output)
	}
}

func TestGenerateCommandSupplierRustAKSKPreservesBodyOrder(t *testing.T) {
	dir := t.TempDir()
	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{
		"generate",
		"--target", "rust",
		"--input", filepath.Join("..", "..", "examples", "supplier.json"),
		"--config", filepath.Join("..", "..", "examples", "supplier.opencli.yaml"),
		"--output", dir,
		"--module", "supplier-cli",
		"--app", "supplier-cli",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute supplier rust generate: %v", err)
	}

	commandContent, err := os.ReadFile(filepath.Join(dir, "src", "commands", "supplier.rs"))
	if err != nil {
		t.Fatalf("read generated supplier command: %v", err)
	}
	commandText := string(commandContent)
	for _, want := range []string{
		`#[arg(short = 'H', long = "header")]`,
		`let body = build_kanban_delivery_http_body(&args)?;`,
		`client::request_json_text("POST", &path, query, headers, body).await?;`,
		`parts.push(format!("\"date\":{value}"));`,
		`parts.push(format!("\"pageSize\":{value}"));`,
		`parts.push(format!("\"pageNum\":{value}"));`,
		`parts.push(format!("\"isForce\":{value}"));`,
	} {
		if !strings.Contains(commandText, want) {
			t.Fatalf("generated supplier rust command missing %q:\n%s", want, commandText)
		}
	}
	dateIdx := strings.Index(commandText, `parts.push(format!("\"date\":{value}"));`)
	pageSizeIdx := strings.Index(commandText, `parts.push(format!("\"pageSize\":{value}"));`)
	pageNumIdx := strings.Index(commandText, `parts.push(format!("\"pageNum\":{value}"));`)
	isForceIdx := strings.Index(commandText, `parts.push(format!("\"isForce\":{value}"));`)
	if !(dateIdx < pageSizeIdx && pageSizeIdx < pageNumIdx && pageNumIdx < isForceIdx) {
		t.Fatalf("generated supplier rust body fields are not in spec order:\n%s", commandText)
	}
	if strings.Contains(commandText, `return Ok(Some(Value::Object(payload)));`) {
		t.Fatalf("generated supplier rust HTTP command should not rebuild JSON through Value::Object:\n%s", commandText)
	}

	clientContent, err := os.ReadFile(filepath.Join(dir, "src", "client.rs"))
	if err != nil {
		t.Fatalf("read generated client: %v", err)
	}
	clientText := string(clientContent)
	for _, want := range []string{
		`pub async fn request_json_text`,
		`crate::auth::apply_aksk(&mut headers, method, path, body.as_deref())?;`,
		`.body(body.clone())`,
	} {
		if !strings.Contains(clientText, want) {
			t.Fatalf("generated rust client missing %q:\n%s", want, clientText)
		}
	}
}

func TestGenerateCommandRejectsIncompleteCustomSigner(t *testing.T) {
	dir := t.TempDir()
	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{
		"generate",
		"--auth", "ak_sk",
		"--signer", "unknown",
		"--input", filepath.Join("..", "..", "examples", "petstore.yaml"),
		"--output", dir,
		"--module", "github.com/acme/generated",
		"--app", "petcli",
	})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), `custom signer "unknown" requires auth.signer.algorithm`) {
		t.Fatalf("expected incomplete custom signer error, got %v", err)
	}
}

func TestGenerateCommandRejectsUnsupportedSignerAlgorithm(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "opencli.yaml")
	if err := os.WriteFile(configPath, []byte(`
auth:
  type: ak_sk
  signer:
    profile: custom_supplier
    algorithm: hmac_sha256
    headers:
      access_key: X-App-Key
      signature: X-Sign
      timestamp: X-Timestamp
      nonce: X-Nonce
    canonical:
      template: "method={method}&path={path}&appKey={access_key}&appSecret={secret_key}&timestamp={timestamp}&nonce={nonce}&jsonBody={json_body}"
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	outDir := filepath.Join(dir, "out")
	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{
		"generate",
		"--input", filepath.Join("..", "..", "examples", "petstore.yaml"),
		"--config", configPath,
		"--output", outDir,
		"--module", "github.com/acme/generated",
		"--app", "petcli",
	})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), `unsupported algorithm "hmac_sha256"`) {
		t.Fatalf("expected unsupported algorithm error, got %v", err)
	}
}

func TestGenerateCommandAcceptsConfiguredSignerProfile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "opencli.yaml")
	if err := os.WriteFile(configPath, []byte(`
auth:
  type: ak_sk
  signer:
    profile: custom_supplier
    algorithm: sha512_hex
    headers:
      access_key: X-App-Key
      signature: X-Sign
      timestamp: X-Timestamp
      nonce: X-Nonce
    path:
      strip_prefix: /api-apply
    canonical:
      template: "method={method}&path={path}&appKey={access_key}&appSecret={secret_key}&timestamp={timestamp}&nonce={nonce}&jsonBody={json_body}"
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	outDir := filepath.Join(dir, "out")
	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{
		"generate",
		"--input", filepath.Join("..", "..", "examples", "petstore.yaml"),
		"--config", configPath,
		"--output", outDir,
		"--module", "github.com/acme/generated",
		"--app", "petcli",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute generate: %v", err)
	}

	authContent, err := os.ReadFile(filepath.Join(outDir, "internal", "auth", "aksk.go"))
	if err != nil {
		t.Fatalf("read generated auth: %v", err)
	}
	content := string(authContent)
	for _, want := range []string{`const signerProfile = "custom_supplier"`, `const accessKeyHeader = "X-App-Key"`, `const signatureHeader = "X-Sign"`} {
		if !strings.Contains(content, want) {
			t.Fatalf("generated auth missing %q:\n%s", want, content)
		}
	}
}

func TestGenerateCommandAcceptsAppVersionFlag(t *testing.T) {
	dir := t.TempDir()
	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{
		"generate",
		"--target", "rust",
		"--input", filepath.Join("..", "..", "examples", "petstore.yaml"),
		"--output", dir,
		"--module", "petcli",
		"--app", "petcli",
		"--app-version", "0.0.1",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute rust generate: %v", err)
	}

	cargo, err := os.ReadFile(filepath.Join(dir, "Cargo.toml"))
	if err != nil {
		t.Fatalf("read Cargo.toml: %v", err)
	}
	if !strings.Contains(string(cargo), `version = "0.0.1"`) {
		t.Fatalf("generated Cargo.toml missing app version:\n%s", cargo)
	}
}

func TestGenerateCommandAppVersionFlagOverridesConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "opencli.yaml")
	if err := os.WriteFile(configPath, []byte(`app:
  version: 0.0.1
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	outDir := filepath.Join(dir, "out")
	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{
		"generate",
		"--target", "rust",
		"--input", filepath.Join("..", "..", "examples", "petstore.yaml"),
		"--output", outDir,
		"--module", "petcli",
		"--app", "petcli",
		"--config", configPath,
		"--app-version", "0.0.2",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute rust generate: %v", err)
	}

	cargo, err := os.ReadFile(filepath.Join(outDir, "Cargo.toml"))
	if err != nil {
		t.Fatalf("read Cargo.toml: %v", err)
	}
	if !strings.Contains(string(cargo), `version = "0.0.2"`) {
		t.Fatalf("generated Cargo.toml did not prefer flag version:\n%s", cargo)
	}
}

func TestGenerateCommandRejectsUnknownTarget(t *testing.T) {
	dir := t.TempDir()
	cmd := app.NewRootCommand()
	cmd.SetArgs([]string{
		"generate",
		"--target", "python",
		"--input", filepath.Join("..", "..", "examples", "petstore.yaml"),
		"--output", dir,
		"--module", "petcli",
		"--app", "petcli",
	})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "unsupported target") {
		t.Fatalf("expected unsupported target error, got %v", err)
	}
}

func assertGoFilesGofmtClean(t *testing.T, root string) {
	t.Helper()

	var files []string
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		t.Fatalf("walk generated files: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("expected generated Go files under %s", root)
	}

	cmd := exec.Command("gofmt", append([]string{"-l"}, files...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run gofmt: %v\n%s", err, output)
	}
	if strings.TrimSpace(string(output)) != "" {
		t.Fatalf("generated Go files are not gofmt-clean:\n%s", output)
	}
}
