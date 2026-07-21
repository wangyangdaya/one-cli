# Generated CLI Runtime Configuration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Generate Go and Rust Skills CLIs that resolve Base URL and credentials from environment overrides or a sealed `config/runtime.yaml`, with identical behavior in both targets.

**Architecture:** A generator-side `runtimeconfig` package validates a secret-free YAML source, reads the build-time credential from the environment, seals it with AES-256-GCM, and returns ciphertext plus two XOR key shares. Rendering writes only the sealed YAML and embeds the shares in target-specific runtime modules. Generated runtimes apply explicit headers and plaintext environment overrides first, then lazily decrypt the file credential only when unresolved.

**Tech Stack:** Go 1.26, Cobra, `gopkg.in/yaml.v3`, Go `crypto/aes`; generated Rust 2021, `serde_yaml`, `aes-gcm`, `base64`, existing `reqwest`/`clap`.

---

## File structure

- Create `internal/runtimeconfig/runtimeconfig.go`: source schema, strict validation, environment credential selection, AES-GCM sealing, key-share generation.
- Create `internal/runtimeconfig/runtimeconfig_test.go`: deterministic unit tests for validation, envelope format, API-key metadata, and absence of plaintext.
- Modify `internal/app/generate_command.go`: add `--runtime-config`, load/seal before rendering, add `api_key` auth mode.
- Modify `internal/model/app.go`: add `AuthTypeAPIKey`.
- Modify `internal/render/project.go`: introduce typed render options while preserving the existing `Project` test API.
- Modify `internal/render/files.go`, `go_project.go`, and `rust_project.go`: carry sealing metadata, write `config/runtime.yaml` as `0600`, emit target runtime modules and launchers.
- Create `internal/render/templates/go/runtime_config.go.tmpl`: Go file lookup, strict parsing, lazy decrypt, and per-field resolution.
- Create `internal/render/templates/rust/runtime_config.rs.tmpl`: Rust equivalent of the Go contract.
- Modify generated HTTP/MCP templates, launchers, module manifests, README, and Skill templates to use and document the shared contract.
- Modify `tests/command/opencli_generate_test.go`: generator flag, sealed output, permissions, auth validation, and generated-file assertions.
- Create `tests/integration/runtime_config_cli_test.go`: execute generated Go and Rust CLIs against a local server and verify file/env/header precedence.

### Task 1: Generator-side runtime configuration sealing

**Files:**
- Create: `internal/runtimeconfig/runtimeconfig.go`
- Create: `internal/runtimeconfig/runtimeconfig_test.go`

- [ ] **Step 1: Write failing strict-schema and sealing tests**

Define tests around this API:

```go
type SealOptions struct {
	AuthMode  string
	Getenv    func(string) string
	Random    io.Reader
}

type Bundle struct {
	YAML      []byte
	KeyShareA [32]byte
	KeyShareB [32]byte
	HasSecret bool
}

func LoadAndSeal(path string, opts SealOptions) (Bundle, error)
```

Cover bearer input, API-key input, unknown fields, plaintext `token`/`value` rejection, missing generation-time credential, auth-mode mismatch, deterministic random input, and assertions that neither the credential nor a complete AES key appears in `Bundle.YAML`.

- [ ] **Step 2: Verify RED**

Run: `env GOCACHE=/tmp/one-cli-gocache go test ./internal/runtimeconfig -run Test -v`

Expected: FAIL because `internal/runtimeconfig` and `LoadAndSeal` do not exist.

- [ ] **Step 3: Implement the minimal sealing package**

Use strict `yaml.Decoder.KnownFields(true)`, validate `version: v1`, map `token` to `OPENCLI_AUTH_TOKEN` and `api_key` to `OPENCLI_API_KEY`, generate a 32-byte AES key plus GCM nonce from `Random`, and emit:

```yaml
version: v1
base_url: https://api.example.com
auth:
  type: bearer
  encrypted_value: ENC[v1:...]
```

Compute AAD as `opencli:v1:<auth-type>:<lowercase-header>`. Generate `KeyShareA` randomly and set `KeyShareB[i] = key[i] ^ KeyShareA[i]`.

- [ ] **Step 4: Verify GREEN**

Run: `env GOCACHE=/tmp/one-cli-gocache go test ./internal/runtimeconfig -v`

Expected: PASS.

### Task 2: Generator flag and render bundle

**Files:**
- Modify: `internal/app/generate_command.go`
- Modify: `internal/model/app.go`
- Modify: `internal/render/project.go`
- Modify: `internal/render/files.go`
- Modify: `internal/render/go_project.go`
- Modify: `internal/render/rust_project.go`
- Test: `tests/command/opencli_generate_test.go`

- [ ] **Step 1: Write failing command tests**

Add tests that invoke `RunGenerate` with `RuntimeConfigPath`, set the relevant credential through `t.Setenv`, and assert:

```go
runtimePath := filepath.Join(dir, "config", "runtime.yaml")
content, err := os.ReadFile(runtimePath)
// content contains encrypted_value and not the plaintext credential
// mode.Perm() == 0o600 on POSIX
```

Also test `--auth api_key`, missing credential, incompatible auth type, and that omission of `--runtime-config` preserves generation.

- [ ] **Step 2: Verify RED**

Run: `env GOCACHE=/tmp/one-cli-gocache go test ./tests/command -run 'TestGenerateCommand.*RuntimeConfig|TestGenerateCommand.*APIKey' -v`

Expected: FAIL because the option and auth type are unsupported.

- [ ] **Step 3: Add the option and typed render path**

Add:

```go
type GenerateOptions struct {
	// existing fields
	RuntimeConfigPath string
}

type ProjectOptions struct {
	Target        string
	SkillLang     string
	RuntimeBundle *runtimeconfig.Bundle
}
```

Keep `render.Project(...)` as a compatibility wrapper for existing tests and route `RunGenerate` through `render.ProjectWithOptions(...)`. Write the bundle only after successful validation. Extend accepted auth values with `model.AuthTypeAPIKey`.

- [ ] **Step 4: Verify GREEN**

Run the targeted command-test command from Step 2.

Expected: PASS.

### Task 3: Generated Go runtime resolution

**Files:**
- Create: `internal/render/templates/go/runtime_config.go.tmpl`
- Modify: `internal/render/templates/go/group_service_http.go.tmpl`
- Modify: `internal/render/templates/go/group_service_mcp_http.go.tmpl`
- Modify: `internal/render/templates/go/bin_launcher.sh.tmpl`
- Modify: `internal/render/templates/go/bin_launcher.cmd.tmpl`
- Test: `tests/integration/runtime_config_cli_test.go`

- [ ] **Step 1: Write a failing generated-Go integration test**

Generate a token-auth Go CLI with a sealed file, start `httptest.Server`, put the server URL in the source runtime YAML, run the generated CLI with credential environment variables removed, and assert the server receives `Authorization: Bearer file-token`.

Add table cases for `OPENCLI_AUTH_TOKEN` overriding the file and explicit `--header "Authorization: Bearer explicit"` overriding both.

- [ ] **Step 2: Verify RED**

Run: `env GOCACHE=/tmp/one-cli-gocache go test ./tests/integration -run TestGeneratedGoRuntimeConfig -v`

Expected: FAIL with missing `OPENCLI_BASE_URL` or absent authorization.

- [ ] **Step 3: Implement the Go runtime template**

Generate an `internal/config/runtime_config.go` API shaped as:

```go
type Resolved struct {
	BaseURL string
	Header  string
	Secret  string
}

func Resolve(appName, fallbackURL, authMode, authHeader string) (Resolved, error)
```

Resolve `OPENCLI_CONFIG`, launcher-rooted `config/runtime.yaml`, and XDG/home fallback. Parse strictly, apply `OPENCLI_BASE_URL` and credential environment values first, and call AES-GCM `Open` only if the credential is still unresolved. Update HTTP and MCP streamable-HTTP templates to use `Resolved`; apply auth only when the matching explicit request header is absent.

- [ ] **Step 4: Verify GREEN**

Run the generated-Go integration test from Step 2.

Expected: PASS.

### Task 4: Generated API-key behavior

**Files:**
- Modify: `internal/render/template_helpers.go`
- Modify: `internal/render/templates/go/group_service_http.go.tmpl`
- Modify: `internal/render/templates/go/group_service_mcp_http.go.tmpl`
- Modify: `internal/render/templates/rust/client.rs.tmpl`
- Test: `tests/integration/runtime_config_cli_test.go`

- [ ] **Step 1: Add failing API-key integration cases**

Generate with `--auth api_key` and `auth.header: X-API-Key`. Assert file fallback, `OPENCLI_API_KEY` override, and explicit `--header "X-API-Key: explicit"` precedence.

- [ ] **Step 2: Verify RED**

Run: `env GOCACHE=/tmp/one-cli-gocache go test ./tests/integration -run TestGeneratedGoAPIKeyRuntimeConfig -v`

Expected: FAIL because API-key request application is not rendered.

- [ ] **Step 3: Render API-key authentication**

Add `appUsesAPIKey` template helper and emit raw API-key values under the configured header. Do not add a `Bearer` prefix. Keep `none` and `ak_sk` paths unchanged.

- [ ] **Step 4: Verify GREEN**

Run the API-key test from Step 2.

Expected: PASS.

### Task 5: Generated Rust runtime parity

**Files:**
- Create: `internal/render/templates/rust/runtime_config.rs.tmpl`
- Modify: `internal/render/templates/rust/main.rs.tmpl`
- Modify: `internal/render/templates/rust/client.rs.tmpl`
- Modify: `internal/render/templates/rust/Cargo.toml.tmpl`
- Create: `internal/render/templates/rust/bin_launcher.sh.tmpl`
- Create: `internal/render/templates/rust/bin_launcher.cmd.tmpl`
- Modify: `internal/render/rust_project.go`
- Test: `tests/integration/runtime_config_cli_test.go`

- [ ] **Step 1: Write failing Rust parity tests**

Reuse the local HTTP observer cases for Rust: bearer file fallback, plaintext environment override, explicit header override, API-key fallback, and API-key environment override. Skip only when `cargo` is unavailable using the repository's existing Rust-test convention.

- [ ] **Step 2: Verify RED**

Run: `env GOCACHE=/tmp/one-cli-gocache go test ./tests/integration -run 'TestGeneratedRust.*RuntimeConfig' -v`

Expected: FAIL because Rust still reads only environment variables.

- [ ] **Step 3: Implement matching Rust resolution**

Add `serde_yaml`, `aes-gcm`, and `base64` dependencies. Generate `runtime_config.rs` with the same schema, path order, AAD, XOR share reconstruction, environment-first lazy decrypt, and error redaction as Go. Add Rust launchers that set `OPENCLI_CONFIG` to the project-root config only when the caller has not supplied it.

- [ ] **Step 4: Verify GREEN**

Run the Rust parity test from Step 2.

Expected: PASS.

### Task 6: Skill and README contract

**Files:**
- Modify: `internal/render/templates/skill.md.tmpl`
- Modify: `internal/render/templates/skill_zh.md.tmpl`
- Modify: `internal/render/templates/go/readme.md.tmpl`
- Modify: `internal/render/templates/rust/readme.md.tmpl`
- Test: `tests/unit/render_test.go`
- Test: `tests/command/readme_smoke_test.go`

- [ ] **Step 1: Write failing documentation assertions**

Assert generated documents mention `config/runtime.yaml`, `OPENCLI_CONFIG`, environment-first behavior, and applicable credential variables, while never containing actual fixture credentials or claiming the value is unrecoverably encrypted.

- [ ] **Step 2: Verify RED**

Run: `env GOCACHE=/tmp/one-cli-gocache go test ./tests/unit ./tests/command -run 'RuntimeConfig|Readme' -v`

Expected: FAIL because generated documentation still describes environment-only setup.

- [ ] **Step 3: Update shared documentation templates**

Document file fallback and plaintext environment override behavior in English and Chinese. Keep values and key material out of all Skill assets.

- [ ] **Step 4: Verify GREEN**

Run the documentation test command from Step 2.

Expected: PASS.

### Task 7: Full verification and compatibility

**Files:**
- Modify as required by failures only.

- [ ] **Step 1: Format**

Run: `make fmt`

Expected: exit 0.

- [ ] **Step 2: Run all Go-hosted tests**

Run: `env GOCACHE=/tmp/one-cli-gocache go test ./...`

Expected: all packages PASS.

- [ ] **Step 3: Run generated-target smoke tests**

Run:

```bash
./scripts/smoke.sh --target go --input ./examples/petstore.yaml
./scripts/smoke.sh --target rust --input ./examples/petstore.yaml
```

Expected: both targets generate and build successfully.

- [ ] **Step 4: Inspect for plaintext leakage and patch hygiene**

Run:

```bash
rg -n 'file-token|fixed-application-token|fixed-application-api-key' tmp tests internal/render || true
git diff --check
git status --short
```

Expected: fixture secrets occur only in test source, `git diff --check` exits 0, and status lists only planned files.

- [ ] **Step 5: Review requirements**

Re-read `docs/superpowers/specs/2026-07-21-generated-cli-runtime-config-design.md` and verify every requirement against tests or generated output before claiming completion.
