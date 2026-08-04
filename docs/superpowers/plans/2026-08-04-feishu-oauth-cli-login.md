# Feishu OAuth CLI Login MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the local mock authorization server with a Feishu token broker and make generated authorization-code CLIs support `login`, `status`, `logout`, local refresh-token storage, and automatic refresh.

**Architecture:** Feishu owns browser authorization. The Python demo broker holds the Feishu App Secret and proxies authorization-code and refresh-token grants. Generated CLIs hold the resulting token set locally and refresh it through the broker before authenticated requests.

**Tech Stack:** Python 3.12, FastAPI, HTTPX, pytest, Go templates, Rust templates, Cobra, clap, Go integration tests.

---

### Task 1: Replace the local OAuth server with a Feishu token broker

**Files:**
- Modify: `oauth2/src/oauth2_server/config.py`
- Modify: `oauth2/src/oauth2_server/app.py`
- Create: `oauth2/src/oauth2_server/broker.py`
- Replace: `oauth2/tests/conftest.py`
- Replace: `oauth2/tests/test_authorization.py`
- Delete: obsolete local issuer tests and implementation files after their callers are removed

- [ ] **Step 1: Write failing broker tests**

Cover a JSON or form authorization-code grant, a refresh grant, rejection of a mismatched client ID/redirect URI, safe upstream error forwarding, and missing environment configuration. Inject `httpx.MockTransport` so no test reaches Feishu.

```python
response = client.post("/oauth/token", data={
    "grant_type": "authorization_code",
    "client_id": "cli_test",
    "code": "one-time-code",
    "redirect_uri": "http://127.0.0.1:18081/oauth/callback",
})
assert response.status_code == 200
assert response.json()["refresh_token"] == "refresh-1"
assert upstream_request.json()["client_secret"] == "server-secret"
```

- [ ] **Step 2: Run tests and verify RED**

Run: `cd oauth2 && uv run pytest tests/test_authorization.py -q`

Expected: failures because the current app is a local authorization server and does not proxy Feishu.

- [ ] **Step 3: Implement the minimal broker**

Use a validated settings object with `FEISHU_APP_ID`, `FEISHU_APP_SECRET`, `FEISHU_REDIRECT_URI`, and a default upstream token URL of `https://open.feishu.cn/open-apis/authen/v2/oauth/token`. Expose `POST /oauth/token`; accept only `authorization_code` and `refresh_token`; add the secret only in the upstream JSON body; return `Cache-Control: no-store` and never log request bodies.

- [ ] **Step 4: Remove obsolete local issuer behavior**

Remove authorization pages, test users, JWT/JWKS, in-memory grants, expenses, and templates once the broker app has no imports of them. Keep `/healthz` and the fixed local server entrypoint.

- [ ] **Step 5: Verify GREEN and lint**

Run:

```bash
cd oauth2
uv run pytest -q
uv run ruff check .
uv run ruff format --check .
```

Expected: all pass.

### Task 2: Extend authorization-code runtime configuration

**Files:**
- Modify: `internal/runtimeconfig/runtimeconfig.go`
- Modify: `internal/runtimeconfig/runtimeconfig_test.go`
- Modify: `internal/render/templates/go/runtime_config.go.tmpl`
- Modify: `internal/render/templates/rust/runtime_config.rs.tmpl`
- Modify: `tests/unit/runtime_config_contract_test.go`

- [ ] **Step 1: Write failing config tests**

Add `redirect_uri` to authorization-code runtime YAML and assert it survives sealing/rendering. Reject a non-loopback redirect URI for this generated CLI flow.

```yaml
auth:
  type: oauth2
  grant_type: authorization_code
  redirect_uri: http://127.0.0.1:18081/oauth/callback
```

- [ ] **Step 2: Run tests and verify RED**

Run: `go test ./internal/runtimeconfig ./tests/unit -run 'OAuth2|RuntimeConfig' -v`

- [ ] **Step 3: Implement the field and validation**

Carry `RedirectURI` through source config, sealed bundle, and generated runtime structs. Require `http`, host `127.0.0.1`, an explicit port, and a non-root callback path when configured. Preserve the existing dynamic loopback fallback for non-Feishu configurations.

- [ ] **Step 4: Run tests and verify GREEN**

Run: `go test ./internal/runtimeconfig ./tests/unit -run 'OAuth2|RuntimeConfig' -v`

### Task 3: Implement the generated Go CLI token lifecycle

**Files:**
- Modify: `internal/render/templates/go/auth_oauth2.go.tmpl`
- Modify: `internal/render/templates/go/runtime_config.go.tmpl`
- Modify: `internal/render/templates/go/root_main.go.tmpl`
- Modify: `tests/integration/runtime_config_cli_test.go`

- [ ] **Step 1: Write failing generated-CLI tests**

Extend the existing authorization-code integration fixture so the fake token endpoint returns:

```json
{
  "access_token": "access-1",
  "refresh_token": "refresh-1",
  "token_type": "Bearer",
  "scope": "offline_access contact:user.base:readonly",
  "expires_in": 3600,
  "refresh_token_expires_in": 604800
}
```

Assert fixed callback use, complete token persistence, `status` results, automatic refresh before a generated API request, refresh-token rotation, and logout cleanup.

- [ ] **Step 2: Run the integration test and verify RED**

Run: `go test ./tests/integration -run TestGeneratedGoOAuth2AuthorizationCode -v`

- [ ] **Step 3: Implement token storage and status**

Define one stored token set with access/refresh tokens, token type, scope, and absolute expirations. Save through create-`0600` temporary file plus rename. Add `NewOAuth2StatusCommand` and register it at the root beside `login` and `logout`.

- [ ] **Step 4: Implement refresh resolution**

In `ResolveOAuthToken`, classify the stored set with a five-minute refresh window. For `needs_refresh`, submit a standard `refresh_token` grant to the configured token endpoint, require the rotated refresh token, atomically save the new set, and return the new access token. For missing/expired refresh credentials, return `login_required: run <app> login`.

- [ ] **Step 5: Harden callback behavior**

Bind the configured loopback redirect URI when present, retain the dynamic fallback otherwise, and handle `error=access_denied` before checking for a missing code. Keep state validation and the two-minute timeout. Do not add PKCE.

- [ ] **Step 6: Run the integration test and verify GREEN**

Run: `go test ./tests/integration -run TestGeneratedGoOAuth2AuthorizationCode -v`

### Task 4: Preserve Rust target parity

**Files:**
- Modify: `internal/render/templates/rust/oauth_auth.rs.tmpl`
- Modify: `internal/render/templates/rust/runtime_config.rs.tmpl`
- Modify: `internal/render/templates/rust/cli.rs.tmpl`
- Modify: `tests/integration/runtime_config_cli_test.go`

- [ ] **Step 1: Write failing Rust generation/runtime assertions**

Require the generated Rust CLI to expose `login`, `status`, and `logout`, persist the complete token set, and refresh a near-expiry token through the same standard broker contract.

- [ ] **Step 2: Run and verify RED**

Run: `go test ./tests/integration -run 'OAuth2AuthorizationCodeGeneratesRustTarget|GeneratedRustOAuth2AuthorizationCode' -v`

- [ ] **Step 3: Implement parity using existing Rust helpers**

Extend the existing serde token record and reqwest exchange code; do not introduce a second protocol abstraction. Use the configured fixed loopback URI where present and preserve the existing dynamic fallback.

- [ ] **Step 4: Run and verify GREEN**

Run: `go test ./tests/integration -run 'OAuth2AuthorizationCodeGeneratesRustTarget|GeneratedRustOAuth2AuthorizationCode' -v`

### Task 5: Wire the Feishu demo and verify the repository

**Files:**
- Modify: `oauth2/openapi.yaml`
- Modify: `oauth2/opencli.runtime.yaml`
- Modify: `oauth2/README.md`
- Delete: `oauth2/templates/authorize.html`

- [ ] **Step 1: Update the example contract**

Point authorization to `https://accounts.feishu.cn/open-apis/authen/v1/authorize`, token exchange to `http://127.0.0.1:18080/oauth/token`, use an explicit `cli_replace_with_feishu_app_id` placeholder in the checked-in YAML, request `offline_access`, and document the exact fixed callback registration.

- [ ] **Step 2: Format generated-source inputs**

Run: `make fmt` and `cd oauth2 && uv run ruff format .`

- [ ] **Step 3: Run full verification**

Run:

```bash
make test
cd oauth2 && uv run pytest -q && uv run ruff check . && uv run ruff format --check .
```

Expected: all repository and broker checks pass.
