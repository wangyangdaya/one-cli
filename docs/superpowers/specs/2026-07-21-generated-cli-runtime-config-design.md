# Generated CLI Runtime Configuration Design

**Date:** 2026-07-21
**Status:** Approved
**Scope:** One generated CLI and its generated Skills; Go and Rust targets

## 1. Context

`opencli` currently generates Go and Rust CLIs that obtain their API base URL and token from environment variables such as `OPENCLI_BASE_URL` and `OPENCLI_AUTH_TOKEN`. A generated Skill invokes the CLI, so the Skill inherits that runtime behavior.

For many internal systems, however, the CLI has a stable service endpoint and an application-level credential. The person or Agent invoking the Skill should not need to provide those values for every run. Some services use a bearer token; others use a fixed API key header.

The first delivery must solve this for one generated CLI while preserving a contract that later supports several independently configured CLIs. It must behave identically for Go and Rust generation targets.

## 2. Goals

- Allow a generated CLI to read `base_url` and authentication material from one YAML file.
- Preserve environment variables as higher-priority runtime overrides.
- Support bearer tokens and fixed API key headers.
- Give Go and Rust generated CLIs the same schema, lookup rules, errors, and request behavior.
- Keep existing generated projects compatible when no runtime configuration file is supplied.
- Keep credentials out of generated `SKILL.md`, command arguments, trace output, and error messages.

## 3. Non-goals

- Composing multiple generated CLIs or routing among multiple Skills.
- User login, OAuth refresh, or per-user credential storage.
- AK/SK signing changes; the existing AK/SK mechanism remains separate.
- Claiming that an application-level credential distributed with a CLI is secret from a user who can read or execute that CLI.
- Providing cryptographic confidentiality from a person who controls the client. A client that decrypts without external identity or key provisioning necessarily contains recoverable decryption material.

## 4. Selected approach

Add an optional generation input:

```bash
./dist/opencli generate \
  --input ./openapi.yaml \
  --output ./generated-cli \
  --module example/generated-cli \
  --app domain-cli \
  --target go \
  --runtime-config ./runtime.yaml
```

The same flag and source file work with `--target rust`. `opencli` validates the source and copies it to:

```text
generated-cli/
├── config/
│   └── runtime.yaml
├── skills/
├── bin/
└── ... generated Go or Rust sources
```

Passing secrets directly as generator command-line flags is intentionally not supported because shell history and process listings can expose them. The source `runtime.yaml` contains only non-secret metadata. At generation time, `opencli` obtains the credential from `OPENCLI_AUTH_TOKEN` or `OPENCLI_API_KEY`, encrypts it, and writes only ciphertext to the generated runtime file. A per-CLI random decryption key is split and obfuscated in generated runtime code and compiled into the CLI. Generated output may be overwritten by later generation.

If `--runtime-config` is omitted, no credential file is generated and current environment-based behavior remains valid.

## 5. Shared configuration contract

The generator input contains no credential value:

```yaml
version: v1
base_url: https://api.example.com
auth:
  type: bearer
```

For `api_key`, the input additionally declares `auth.header`. The generator reads the corresponding credential environment variable and produces the sealed form below.

### 5.1 Bearer token

```yaml
version: v1
base_url: https://api.example.com
auth:
  type: bearer
  encrypted_value: ENC[v1:BASE64_NONCE_AND_CIPHERTEXT]
```

The resulting request header is:

```text
Authorization: Bearer <decrypted-application-token>
```

### 5.2 Fixed API key

```yaml
version: v1
base_url: https://api.example.com
auth:
  type: api_key
  header: X-API-Key
  encrypted_value: ENC[v1:BASE64_NONCE_AND_CIPHERTEXT]
```

The resulting request header is:

```text
X-API-Key: <decrypted-application-api-key>
```

### 5.3 Schema rules

- `version` must be `v1`.
- `base_url` is optional only when another source supplies it.
- `auth` is optional for a CLI generated with no authentication.
- `bearer` requires non-empty `encrypted_value` and rejects `header`.
- `api_key` requires non-empty `header` and `encrypted_value`.
- Plaintext fields such as `token` and `value` are rejected.
- Unknown fields are rejected so a misspelled credential field does not silently disable authentication.
- `http` URLs remain allowed for local development and tests; production policy may enforce HTTPS outside this feature.
- Whitespace-only values are treated as absent.

The runtime file describes concrete request behavior. Generator auth settings still control which authentication implementation is emitted. A bearer file is valid for generated token authentication; an API key file is valid for generated API-key authentication. This delivery therefore adds `api_key` as a supported generated auth mode alongside `token`, `ak_sk`, and `none`.

## 6. Lookup and precedence

Resolution is performed per field, not by selecting one source for the entire configuration.

### 6.1 Configuration file path

The generated CLI searches in this order:

1. `OPENCLI_CONFIG`, when non-empty.
2. `config/runtime.yaml` rooted at the generated project, set by the generated launcher.
3. `${XDG_CONFIG_HOME}/<app>/config.yaml`, or `~/.config/<app>/config.yaml` when `XDG_CONFIG_HOME` is unset.
4. No file.

An explicitly selected file through `OPENCLI_CONFIG` must exist and parse successfully; otherwise the command fails. Missing implicit files are skipped. A discovered but malformed implicit file fails rather than silently falling through.

Generated shell and Windows launchers set `OPENCLI_CONFIG` only when the caller has not already set it. Go and Rust receive equivalent launchers so Skill invocation does not depend on the caller's current directory.

### 6.2 Base URL

```text
OPENCLI_BASE_URL
  > runtime file base_url
  > endpoint discovered from the source OpenAPI/MCP definition
  > actionable missing-base-URL error
```

### 6.3 Bearer token

```text
explicit Authorization request header
  > OPENCLI_AUTH_TOKEN
  > decrypted runtime file auth.encrypted_value
  > actionable missing-token error when bearer auth is required
```

The environment variable contains the raw token; the file contains its sealed representation. After selecting and, when necessary, decrypting the value, the CLI adds the `Bearer ` prefix exactly once.

`OPENCLI_AUTH_TOKEN` is a plaintext runtime override. When it is present, the CLI uses it directly and does not decrypt `auth.encrypted_value` from the file.

### 6.4 API key

```text
explicit request header matching auth.header
  > OPENCLI_API_KEY
  > decrypted runtime file auth.encrypted_value
  > actionable missing-API-key error when API-key auth is required
```

`auth.header` comes from the validated runtime file or the generated auth configuration. Header-name comparison is case-insensitive.

`OPENCLI_API_KEY` is also a plaintext runtime override. When it is present, the CLI uses it directly and does not decrypt the file credential.

An explicit `--header` value always wins, allowing a one-request override without modifying environment or disk configuration.

## 7. Runtime architecture

Both targets expose the same conceptual runtime module:

```text
load selected YAML file
        |
        v
validate v1 schema and generated auth compatibility
        |
        v
apply plaintext environment overrides per field
        |
        v
decrypt encrypted_value only for an unresolved credential
        |
        v
resolve endpoint and request authentication
        |
        v
construct HTTP or MCP streamable-HTTP request
```

### 7.1 Go target

- Extend the copied generated runtime under `internal/config` with typed YAML loading, path discovery, validation, and merge helpers.
- Route HTTP and MCP streamable-HTTP service templates through that runtime instead of reading environment variables independently.
- Add the YAML dependency to the generated module.

### 7.2 Rust target

- Generate a dedicated `src/runtime_config.rs` with matching data types, path discovery, validation, and merge behavior.
- Route HTTP and MCP streamable-HTTP clients through that module.
- Add matching YAML and configuration-directory dependencies to `Cargo.toml`.

Language-specific error wording may differ grammatically, but errors must identify the same field and condition and must never contain credential values.

## 8. Skill behavior

`SKILL.md` remains declarative and invokes the generated CLI. It does not parse configuration itself.

Generated Skill documentation will state:

- the CLI normally reads `config/runtime.yaml`;
- `OPENCLI_CONFIG` selects another file;
- `OPENCLI_BASE_URL`, `OPENCLI_AUTH_TOKEN`, and `OPENCLI_API_KEY` are higher-priority overrides when applicable;
- credential values must never be placed in the Skill file.

This keeps one credential-loading implementation per CLI and prevents Go/Rust Skill instructions from drifting.

## 9. Security model

- A bundled fixed token/API key is an application credential, not a user secret.
- Credentials are encrypted with AES-256-GCM using a fresh random 256-bit key and nonce. Go and Rust use the same `ENC[v1:...]` envelope format.
- The envelope is `ENC[v1:<base64url(nonce || ciphertext || authentication-tag)>]`. Auth type and API-key header name are authenticated as additional data so they cannot be altered independently of the ciphertext.
- The generated `config/runtime.yaml` contains ciphertext rather than plaintext credentials and uses normal generated-config file permissions; the design does not depend on POSIX `0600` support.
- The source runtime file contains the Base URL and authentication metadata but no credential value. The generator reads the credential from its process environment, which should be populated by a CI/CD secret facility rather than an inline shell assignment.
- The per-CLI key is represented as generated fragments and reconstructed only when decrypting. This prevents the key from appearing as one plain configuration value, but a person who can inspect or debug the generated source/binary can recover it.
- This feature is explicitly lightweight sealing against direct configuration inspection and accidental secret scanning, not a substitute for an OS keychain, TPM, user/device authentication, token broker, or external secret manager.
- Trace and error output may show the selected source path and authentication type, but never token/API-key values or complete authentication headers.
- Documentation and generated messages call this mechanism `sealed` or `obfuscated`; they do not claim that a client-distributed application credential is secret from the client owner.
- A future extension may integrate OS keychains or an external secret manager without changing this precedence contract.

## 10. Generator changes

- Add `RuntimeConfigPath` to `GenerateOptions` and `--runtime-config` to `opencli generate`.
- Parse and validate the file before rendering either target.
- For bearer auth, obtain the generation-time secret from `OPENCLI_AUTH_TOKEN`; for API-key auth, obtain it from `OPENCLI_API_KEY`.
- Generate a random AES-256-GCM key and nonce, encrypt the secret, and discard the plaintext after producing the output buffers.
- Carry only non-secret metadata and obfuscated key fragments needed by templates; never embed credential plaintext into generated source.
- Write `config/runtime.yaml` only after validation and never emit a companion `runtime.key` file.
- Extend supported auth validation with `api_key`.
- Keep `--config` for generator behavior (`opencli.yaml`) distinct from `--runtime-config` for generated CLI runtime values.

## 11. Compatibility

- Existing commands using only `OPENCLI_BASE_URL` and `OPENCLI_AUTH_TOKEN` continue to work.
- Environment credential overrides remain plaintext inputs; encryption/decryption applies only to credential values stored in `runtime.yaml`.
- Existing generated projects are unchanged when `--runtime-config` is omitted, except that regenerated runtime code understands optional files.
- `token` continues to mean bearer authentication.
- `none` never reads or emits authentication credentials.
- `ak_sk` keeps its current environment-based signing behavior in this first delivery.
- Go and Rust accept the same runtime file; target selection cannot alter schema or precedence.

## 12. Verification matrix

The implementation is complete only when equivalent Go and Rust tests cover:

| Case | Expected result |
|---|---|
| File supplies base URL and bearer token | Request reaches file URL with one `Bearer` prefix |
| Environment supplies both fields | Environment wins over file |
| Environment supplies only token | File URL and environment token are combined |
| Explicit auth header is supplied | Explicit header wins |
| File supplies API key | Configured header contains raw API-key value |
| `OPENCLI_API_KEY` is set | Environment API key wins over file value |
| `OPENCLI_CONFIG` points to missing file | Command fails with path, without secret data |
| Implicit file is absent | Generated endpoint/environment fallback works |
| File contains unknown/mutually exclusive fields | Generation or runtime load fails clearly |
| Auth mode is `none` | No auth variable or file credential is used |
| Trace/error output is enabled | Credential values are absent |
| Runtime file is generated | Ciphertext is present and no companion `runtime.key` exists |
| Generated files are inspected | Neither file set nor generated source contains credential plaintext |
| Ciphertext or authentication tag is modified | Command fails without revealing plaintext |
| Generated Go/Rust key fragments are reconstructed | Both targets decrypt the same envelope contract without a companion key file |

At least one integration test per target must execute a generated CLI against a local HTTP server and assert the observed URL and headers. Unit/command tests verify generator validation, emitted files, and documentation.

## 13. Follow-up path

After this single-CLI contract is stable, a multi-Skills installation can place one runtime file beside each CLI and let an orchestrator invoke the appropriate binary. No shared global token registry is required, and independent Base URLs/tokens remain isolated by CLI.
