# OIDC OAuth2 Local Service Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在根目录实现 Python FastAPI OIDC/OAuth2 Authorization Code + PKCE 服务和个人数据 API，完整验证 one-cli 用户认证流程。

**Architecture:** FastAPI 负责 HTTP 和页面，协议逻辑分为内存 grant store、RS256 token service、authorization router 和 resource router。状态和密钥仅在进程内，测试注入 clock 并通过 TestClient 验证协议，不依赖真实浏览器。

**Tech Stack:** Python 3.12、FastAPI、Uvicorn、PyJWT/Cryptography、Jinja2、pytest、HTTPX、uv。

---

### Task 1: 项目骨架与内存 Grant Store

**Files:**
- Create: `oauth2/pyproject.toml`
- Create: `oauth2/src/oauth2_server/config.py`
- Create: `oauth2/src/oauth2_server/models.py`
- Create: `oauth2/src/oauth2_server/store.py`
- Test: `oauth2/tests/test_store.py`

- [ ] 先写 code 一次性消费、过期、refresh rotation/reuse 的失败测试。
- [ ] 运行 `cd oauth2 && uv run pytest tests/test_store.py -q`，确认因实现缺失失败。
- [ ] 实现带 `threading.RLock` 的内存 store；refresh token 只保存 SHA-256 hash。
- [ ] 重跑测试并确认通过。

### Task 2: RS256 Token 与 JWKS

**Files:**
- Create: `oauth2/src/oauth2_server/tokens.py`
- Test: `oauth2/tests/test_tokens.py`

- [ ] 先写 Access Token、ID Token、错误 audience、过期和篡改测试。
- [ ] 运行单文件测试确认红灯。
- [ ] 实现进程内 RSA key、RS256 签发/验证和 JWKS。
- [ ] 重跑测试确认绿灯。

### Task 3: Authorization Code + PKCE、Refresh 与 Revoke

**Files:**
- Create: `oauth2/src/oauth2_server/authorization.py`
- Create: `oauth2/templates/authorize.html`
- Test: `oauth2/tests/test_authorization.py`

- [ ] 先写 authorize 参数、loopback redirect、登录和授权 redirect 测试。
- [ ] 实现 GET/POST `/oauth/authorize`。
- [ ] 先写 code exchange、verifier 错误、重放和过期测试。
- [ ] 实现 `/oauth/token` authorization_code 分支。
- [ ] 先写 refresh rotation/reuse 与 revoke 测试。
- [ ] 实现 refresh_token 和 `/oauth/revoke`。
- [ ] 运行 `uv run pytest tests/test_authorization.py -q`。

### Task 4: Discovery、JWKS、UserInfo 与个人数据 API

**Files:**
- Create: `oauth2/src/oauth2_server/resources.py`
- Create: `oauth2/src/oauth2_server/app.py`
- Test: `oauth2/tests/test_app.py`
- Test: `oauth2/tests/test_resources.py`

- [ ] 先写 Discovery/JWKS/health 失败测试并实现应用组装。
- [ ] 先写无 Token、缺 scope、Alice/Bob 数据隔离和 POST owner 派生测试。
- [ ] 实现 Bearer 验证、UserInfo、`/me` 和费用 API。
- [ ] 运行相关测试确认通过。

### Task 5: 可运行入口与 one-cli 资产

**Files:**
- Create: `oauth2/src/oauth2_server/__main__.py`
- Create: `oauth2/openapi.yaml`
- Create: `oauth2/opencli.runtime.yaml`
- Create: `oauth2/README.md`

- [ ] 增加 `uv run oauth2-server` 启动入口，只绑定 `127.0.0.1:18080`。
- [ ] 添加 Authorization Code security scheme 和 `x-ai-access` OpenAPI。
- [ ] 添加 `--auth oidc --runtime-config` 生成与登录说明。
- [ ] 验证配置中没有 client_secret、密码或 Token。

### Task 6: 安全与回归验证

**Files:**
- Modify: `oauth2/tests/test_authorization.py`
- Modify: `oauth2/tests/test_resources.py`
- Modify: `oauth2/README.md`

- [ ] 增加敏感值不泄漏测试。
- [ ] 运行 `cd oauth2 && uv run pytest -q`。
- [ ] 运行 `cd oauth2 && uv run ruff check . && uv run ruff format --check .`。
- [ ] 运行主仓库 `go test ./...`，确认独立 Python 服务不影响 Go 模块。

## Definition of Done

- [ ] 根目录 `oauth2/` 是 FastAPI 服务，不含 go.mod。
- [ ] Authorization Code + PKCE S256、Discovery、JWKS、UserInfo 完整。
- [ ] Public Client 无 client_secret。
- [ ] refresh rotation/reuse/revoke 可验证。
- [ ] 个人数据服务端按 subject、scope 和 audience 隔离。
- [ ] `openapi.yaml` 和 `opencli.runtime.yaml` 可供 one-cli 使用。
- [ ] pytest、ruff 和主仓库 Go 测试通过。
