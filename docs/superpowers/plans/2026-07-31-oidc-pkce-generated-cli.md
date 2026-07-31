# one-cli OIDC + PKCE User Authorization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 `one-cli` 能从 OpenAPI 生成面向终端桌面 Agent 的用户授权 CLI，支持标准 OIDC 和存量 OAuth2 Authorization Code + PKCE、loopback 回调和 OS Keychain，同时保持现有 `oauth2=client_credentials`、Token、API Key、AK/SK 行为兼容。

**Architecture:** 在生成器中新增 `oidc` 和 `oauth2-pkce` 两个用户授权类型，不改变现有 `oauth2` 的应用身份语义。OIDC 默认使用 Discovery；存量 OAuth2 PKCE 使用显式端点且不假定 ID Token。两种模式共享浏览器授权、PKCE、loopback、Keychain 和 Token Manager，业务系统仍是最终权限边界。

**Tech Stack:** Go 1.23、Cobra、kin-openapi、`golang.org/x/oauth2`、`github.com/coreos/go-oidc/v3/oidc`、`github.com/zalando/go-keyring`、Rust 2021、Clap、Reqwest、`openidconnect`、`keyring`、Go test、Cargo test。

---

## 1. 实施边界与关键决策

本计划的需求基线为：

- [设计规范](../specs/2026-07-29-ai-native-business-api-authorization-design.md)
- 仅支持终端电脑上的 Agent 代表当前用户。
- 不支持服务器 Agent 用户授权、Device Flow、Token Vault、应用权限或工作负载身份。
- 不支持 Resource Owner Password Grant，不接收 username/password。
- Public Client 不配置、不生成、不保存 `client_secret`。
- 每个业务系统使用自己的 provider identity、client_id、audience 和 Token。
- `oauth2` 保持为现有 Client Credentials；用户授权类型为 `oidc` 和 `oauth2-pkce`。
- 没有 Discovery 和 ID Token 的服务不得配置成 `oidc`。
- 受保护命令未登录时返回结构化 `login_required`，不自动打开浏览器。
- 只有显式执行 `auth login` 才启动 loopback listener 和系统浏览器。
- 第一交付版本完成 Go 生成目标；在 Rust 等价实现完成前，生成器必须明确拒绝 Rust 的两种用户授权模式，不能生成缺少安全能力的半成品。

### 1.1 不做的内容

- 不在 one-cli 中实现业务资源权限、租户权限或数据范围判断。
- 不把 `x-ai-access` 当作服务端授权策略。
- 不允许 `Authorization` 自定义 Header 覆盖 OIDC 用户身份。
- 不把 Token 写入 runtime YAML、普通文件、stdout、stderr、日志或 Agent 上下文。
- 不在本次实现高风险操作的一次性确认协议；只解析并暴露风险元数据，为后续治理保留契约。

### 1.2 交付里程碑

| 里程碑 | 范围 | 退出条件 |
| --- | --- | --- |
| M1 协议模型 | OpenAPI、模型、runtime config、生成参数 | `oidc`、`oauth2-pkce` 配置可验证，旧认证测试全通过 |
| M2 Go 最小闭环 | login/status/check/logout、Keychain、刷新、请求注入 | 本地模拟授权服务 E2E 通过 |
| M3 AI 契约 | `x-ai-access`、结构化错误、Skill/README | Agent 不接触 Token，scope 提示可解析 |
| M4 Rust 对齐 | Rust OIDC、Keychain、命令和 E2E | 与 Go 同一验收矩阵通过 |

## 2. 目标文件结构

### 2.1 生成器与共享模型

```text
internal/model/app.go
internal/openapi/document.go
internal/openapi/parser.go
internal/planner/plan.go
internal/runtimeconfig/runtimeconfig.go
internal/app/generate_command.go
internal/render/template_helpers.go
internal/render/render.go
internal/render/go_project.go
internal/render/rust_project.go
```

### 2.2 新增 Go 生成模板

```text
internal/render/templates/go/auth_pkce.go.tmpl
internal/render/templates/go/auth_oidc_verify.go.tmpl
internal/render/templates/go/auth_store.go.tmpl
internal/render/templates/go/auth_command.go.tmpl
internal/render/templates/go/identity_command.go.tmpl
```

生成结果：

```text
internal/auth/pkce.go
internal/auth/oidc_verify.go
internal/auth/store.go
internal/auth/command.go
internal/auth/identity.go
```

现有 `cmd/<app>/main.go` 只负责把 `auth.NewCommand()` 和
`auth.NewIdentityCommand()` 注册到 root，不承载协议逻辑。

### 2.3 新增 Rust 生成模板

```text
internal/render/templates/rust/pkce.rs.tmpl
internal/render/templates/rust/oidc_verify.rs.tmpl
internal/render/templates/rust/token_store.rs.tmpl
internal/render/templates/rust/auth_command.rs.tmpl
internal/render/templates/rust/identity_command.rs.tmpl
```

生成结果：

```text
src/pkce.rs
src/oidc_verify.rs
src/token_store.rs
src/auth_command.rs
src/identity_command.rs
```

### 2.4 测试与样例

```text
examples/oidc_user.yaml
examples/opencli-oidc.yaml
examples/opencli-oauth2-pkce.yaml
tests/unit/openapi_parser_test.go
internal/runtimeconfig/runtimeconfig_test.go
tests/command/opencli_generate_test.go
tests/unit/render_test.go
tests/integration/user_auth_go_cli_test.go
tests/integration/user_auth_rust_cli_test.go
tests/integration/testdata/oidc/
```

## 3. Task 1：扩展认证与 AI 访问领域模型

**Files:**

- Modify: `internal/model/app.go`
- Modify: `internal/openapi/document.go`
- Modify: `internal/planner/plan.go`
- Test: `tests/unit/planner_test.go`

- [ ] **Step 1: 先写失败测试，证明三种 OAuth 相关类型语义不同**

增加测试断言：

```go
func TestPlanPreservesDistinctOAuthAuthTypes(t *testing.T) {
    // oauth2 仍表示 client_credentials。
    // oidc 表示 authorization_code + PKCE 用户授权。
    // oauth2-pkce 表示无 OIDC 身份层的 authorization_code + PKCE。
}
```

运行：

```bash
go test ./tests/unit -run 'TestPlanPreservesDistinctOAuthAuthTypes' -v
```

预期：FAIL，用户授权类型尚不存在。

- [ ] **Step 2: 增加领域类型**

在 `internal/model/app.go` 增加：

```go
const (
    AuthTypeOIDC       = "oidc"
    AuthTypeOAuth2PKCE = "oauth2-pkce"
)

type AIAccess struct {
    SubjectModes       []string
    Audience           string
    Scopes             []string
    TenantBound        bool
    ResourceType       string
    DataClassification string
    Risk               string
    Confirmation       string
    IdempotencyRequired bool
}
```

给 `model.Operation` 增加：

```go
SecurityScopes map[string][]string
AIAccess       AIAccess
```

保留现有 `Security []string`，避免破坏已有模板和 MCP 生成路径。

- [ ] **Step 3: planner 原样传递新字段，不推断业务权限**

只进行确定性的字段复制与排序。`x-ai-access` 缺失时保持零值；不得根据 path、summary 或 HTTP method 猜测 scope、数据级别或风险。

- [ ] **Step 4: 运行单元测试**

```bash
go test ./tests/unit -run 'TestPlan.*(OIDC|AIAccess)' -v
```

预期：PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/model/app.go internal/openapi/document.go internal/planner/plan.go tests/unit/planner_test.go
git commit -m "add oidc and ai access domain model"
```

## 4. Task 2：解析 OpenAPI Authorization Code 与 `x-ai-access`

**Files:**

- Modify: `internal/openapi/document.go`
- Modify: `internal/openapi/parser.go`
- Modify: `tests/unit/openapi_parser_test.go`
- Add: `examples/oidc_user.yaml`

- [ ] **Step 1: 添加 Authorization Code 解析失败测试**

测试必须覆盖：

- `authorizationUrl`
- `tokenUrl`
- flow scopes
- operation security requirement 中“scheme → scopes”的映射
- OpenAPI 3.0 和 3.1
- OAuth Client Credentials 解析结果不变

目标结构：

```go
type SecurityScheme struct {
    Name                             string
    Type                             string
    ClientCredentialsTokenURL        string
    ClientCredentialsScopes          []string
    AuthorizationCodeAuthorizationURL string
    AuthorizationCodeTokenURL         string
    AuthorizationCodeScopes           []string
}
```

运行：

```bash
go test ./tests/unit -run 'TestParse.*AuthorizationCode' -v
```

预期：FAIL。

- [ ] **Step 2: 实现 flow 和 operation scope 解析**

修改 `convertSecuritySchemes`，同时解析
`Flows.ClientCredentials` 与 `Flows.AuthorizationCode`。

将当前只返回 scheme 名称的处理扩展为：

```go
func securityRequirements(
    requirements openapi3.SecurityRequirements,
) (names []string, scopes map[string][]string)
```

排序 scheme 和 scope，确保生成结果稳定。

- [ ] **Step 3: 添加 `x-ai-access` 严格解析测试**

覆盖：

- 合法 user subject mode。
- 必填字段缺失。
- `subject_modes` 包含非 `user`。
- `risk` 非 `read|write|high-risk-write`。
- `confirmation` 非 `never|always`。
- security scopes 与 `x-ai-access.scopes` 不一致。

解析器只负责读取；校验错误由独立验证函数返回，错误中包含
`operationId` 和字段名。

- [ ] **Step 4: 实现扩展解析**

从 kin-openapi 的 `op.Extensions["x-ai-access"]` 解码到显式内部结构。
禁止使用无类型 `map[string]any` 一路传到模板。

新增：

```go
func parseAIAccess(op *openapi3.Operation) (AIAccess, error)
func validateAIAccess(operation Operation) error
```

由于现有 `convertDocument` 不返回 error，需要把内部转换链调整为可返回 error，
并让 `Parse` 保留 `decode openapi` 风格的上下文错误。

- [ ] **Step 5: 增加样例**

`examples/oidc_user.yaml` 至少包含：

- 一个 `expense:read:self` 查询。
- 一个 `expense:submit:self` 写入。
- Authorization Code Security Scheme。
- 完整 `x-ai-access`。
- 不包含用户名、密码、client_secret 或 Token。

- [ ] **Step 6: 运行解析器测试和全量单测**

```bash
go test ./tests/unit -run 'TestParse' -v
go test ./tests/unit -v
```

预期：PASS。

- [ ] **Step 7: 提交**

```bash
git add internal/openapi/document.go internal/openapi/parser.go tests/unit/openapi_parser_test.go examples/oidc_user.yaml
git commit -m "parse oidc authorization code metadata"
```

## 5. Task 3：增加无 Secret 的用户授权 runtime config

**Files:**

- Modify: `internal/runtimeconfig/runtimeconfig.go`
- Modify: `internal/runtimeconfig/runtimeconfig_test.go`
- Add: `examples/opencli-oidc.yaml`

- [ ] **Step 1: 写 runtime config 失败测试**

合法输入：

```yaml
base_url: https://finance-api.example.com
auth:
  type: oidc
  issuer: https://finance-auth.example.com
  client_id: finance-cli
  audience: finance-api
  scopes: [openid, profile, expense:read:self]
  redirect:
    type: loopback
    path: /oauth/callback
  token_store: os_keychain
```

断言：

- 不读取任何 `OPENCLI_*SECRET` 环境变量。
- `Bundle.HasSecret == false`。
- `Bundle.YAML` 不包含 `encrypted_value`。
- issuer、client_id、audience、scopes 和 redirect 被保留。

再增加非法输入表格测试：

- username/password/client_secret/access_token/refresh_token 为未知字段并拒绝。
- issuer 不是 HTTPS。
- loopback path 为空或不是绝对 path。
- token_store 不是 `os_keychain`。
- `oidc` scopes 缺少 `openid`。
- 显式 endpoints 不是 HTTPS。
- `oidc` 缺少 issuer。
- `oidc` 禁用 Discovery 时缺少 authorization/token/JWKS endpoint。
- `oauth2_pkce` 缺少 provider_id、authorization endpoint、token endpoint 或 identity endpoint。
- `oauth2_pkce` 配置 issuer、JWKS 或要求 ID Token。

增加合法的存量 OAuth2 PKCE 输入：

```yaml
base_url: https://finance-api.example.com
auth:
  type: oauth2_pkce
  provider_id: finance-auth
  client_id: finance-cli
  audience: finance-api
  authorization_endpoint: https://finance.example.com/oauth/authorize
  token_endpoint: https://finance.example.com/oauth/token
  identity_endpoint: https://finance-api.example.com/api/v1/me
  scopes: [profile, expense:read:self]
  redirect:
    type: loopback
    path: /oauth/callback
  token_store: os_keychain
```

- [ ] **Step 2: 扩展配置结构**

新增 OIDC 与 OAuth2 PKCE 公共配置结构。OIDC 使用 issuer 作为 provider identity；OAuth2 PKCE 使用显式 `provider_id`：

```go
type UserAuthorizationDefaults struct {
    Scheme          string
    AuthorizationURL string
    TokenURL         string
    Scopes           []string
}

type sourceRedirect struct {
    Type string `yaml:"type"`
    Path string `yaml:"path"`
}
```

`sourceAuth` 和 `sealedAuth` 增加公开字段：

```go
Issuer, ProviderID, AuthorizationURL, RevocationURL, JWKSURL string
UserInfoURL, IdentityURL string
Audience, TokenStore string
Redirect sourceRedirect
```

OIDC 和 OAuth2 PKCE 分支直接 marshal 公开配置并返回，无加密、无 key share。
现有 bearer、api_key、oauth2 分支保持现状。

- [ ] **Step 3: 应用 OpenAPI 默认值**

当 OpenAPI 中恰好存在一个 Authorization Code scheme 时，可默认填充：

- scheme
- authorization endpoint
- token endpoint
- scopes

但 `client_id` 和 `audience` 必须由 runtime config 显式提供，生成器不得猜测。
`oidc` 有 issuer 时运行期优先 Discovery；显式 endpoint 仅作为 OIDC 不支持 Discovery 的 fallback。

`oauth2_pkce` 只使用显式 authorization/token endpoints，不执行 OIDC Discovery，不要求 `openid` scope，也不校验 ID Token。

- [ ] **Step 4: 运行测试**

```bash
go test ./internal/runtimeconfig -v
```

预期：PASS；现有密封配置 golden 不变化。

- [ ] **Step 5: 提交**

```bash
git add internal/runtimeconfig/runtimeconfig.go internal/runtimeconfig/runtimeconfig_test.go examples/opencli-oidc.yaml
git commit -m "add public oidc runtime configuration"
```

## 6. Task 4：扩展生成命令并建立兼容性护栏

**Files:**

- Modify: `internal/app/generate_command.go`
- Modify: `tests/command/opencli_generate_test.go`
- Modify: `tests/command/opencli_root_test.go`

- [ ] **Step 1: 写命令契约失败测试**

覆盖：

```text
--auth oidc --runtime-config <file> --target go      => success
--auth oidc without runtime config                   => clear error
--auth oidc --target rust                            => explicit not-yet-supported error
--auth oauth2-pkce --runtime-config <file>            => success
--auth oauth2-pkce without explicit endpoints         => clear error
--auth oauth2                                        => existing client_credentials behavior
--auth token|api_key|ak_sk|none                      => unchanged
```

- [ ] **Step 2: 更新 auth validation 和 help**

支持列表改为：

```text
token, api_key, ak_sk, oauth2, oidc, oauth2-pkce, or none
```

错误信息必须明确：

```text
oauth2 means client_credentials application auth
oidc means authorization_code + PKCE user auth
oauth2-pkce means authorization_code + PKCE without OIDC discovery or ID token
```

新增用户授权默认值解析，不得复用 Client Credentials 的 `oauth2Defaults`。OpenAPI 只能提供 authorize、token 和 scope 默认值，不能把普通 OAuth2 自动推断为 OIDC。

- [ ] **Step 3: 加 Rust 显式拒绝**

在 `ProjectWithOptions` 之前校验，返回：

```text
target rust does not support user authorization modes yet; use target go
```

M4 完成时删除此限制及其测试，替换为 Rust 成功测试。

- [ ] **Step 4: 运行命令测试**

```bash
go test ./tests/command -run 'TestOpenCLI.*(Auth|OIDC|OAuth)' -v
```

预期：PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/app/generate_command.go internal/render/project.go tests/command/opencli_generate_test.go tests/command/opencli_root_test.go
git commit -m "add user authorization generation modes"
```

## 7. Task 5：生成 Go 用户授权协议核心

**Files:**

- Add: `internal/render/templates/go/auth_pkce.go.tmpl`
- Modify: `internal/render/go_project.go`
- Modify: `internal/render/template_helpers.go`
- Modify: `internal/render/render.go`
- Modify: `internal/render/gomod.tmpl`
- Modify: `internal/render/gosum.tmpl`
- Modify: `tests/unit/render_test.go`

- [ ] **Step 1: 写生成文件失败测试**

分别生成 `--auth oidc` 和 `--auth oauth2-pkce` Go 项目并断言：

- 存在 `internal/auth/pkce.go`。
- OIDC 项目包含 OIDC、OAuth2 和 Keyring 依赖。
- OAuth2 PKCE 项目不引入 OIDC ID Token 校验依赖。
- 非用户授权项目不生成 PKCE 文件，依赖不变化。
- 生成项目通过 `go test ./...`。

- [ ] **Step 2: 添加条件模板函数和生成文件**

新增：

```go
func appUsesUserAuthorization(app model.App) bool
```

只在 auth type 为 `AuthTypeOIDC` 或 `AuthTypeOAuth2PKCE` 时生成共享 PKCE 文件；OIDC 校验模块只为 OIDC 生成。

- [ ] **Step 3: 定义可测试协议边界**

生成的 `internal/auth/pkce.go` 提供：

```go
type BrowserOpener interface {
    Open(string) error
}

type CallbackListener interface {
    RedirectURI() string
    Wait(context.Context, expectedState string) (code string, err error)
    Close() error
}

type TokenStore interface {
    Load(context.Context) (Session, error)
    Save(context.Context, Session) error
    Delete(context.Context) error
}

type Manager struct {
    config Config
    store TokenStore
    browser BrowserOpener
    newListener func(path string) (CallbackListener, error)
    httpClient *http.Client
    now func() time.Time
}
```

生产 listener 必须：

- `net.Listen("tcp", "127.0.0.1:0")`
- 只接受固定 callback path。
- state 不匹配时拒绝。
- 首次成功或超时后关闭。
- 返回浏览器可读的成功/失败短页面。
- 不记录 code、state 或 Token。

- [ ] **Step 4: 使用标准库/协议库实现 PKCE**

登录请求必须具备：

```text
response_type=code
client_id
redirect_uri
scope
state
code_challenge
code_challenge_method=S256
```

`nonce` 只在 OIDC authorize 请求中发送；OAuth2 PKCE 没有 ID Token，不发送 `nonce`。

`audience` 是 CLI 对业务 Access Token 目标系统的期望值，不是 OAuth 标准授权请求参数。
生成器不得擅自向 authorize endpoint 添加非标准 `audience` 参数；如果企业授权服务需要
RFC 8707 `resource` 或私有参数，应作为后续显式、可配置的协议扩展设计。

Token 交换必须具备：

```text
grant_type=authorization_code
client_id
code
redirect_uri
code_verifier
```

不得发送 `client_secret`。

- [ ] **Step 5: 实现 Provider 元数据安全校验**

- OIDC issuer URL 必须为 HTTPS。
- OIDC 默认 Discovery，文档 issuer 必须与配置 issuer 完全匹配。
- OIDC 显式 fallback 仍要求 issuer、authorization、token、JWKS 和 ID Token 校验。
- OAuth2 PKCE 不执行 OIDC Discovery，只读取显式 authorization/token endpoints。
- authorization、token 和所有已配置的 revocation、JWKS、Identity Endpoint 必须为 HTTPS。
- 两种模式都只接受 Authorization Code 和 PKCE S256。

- [ ] **Step 6: 写模板级生成测试并构建样例**

```bash
go test ./tests/unit -run 'TestRender.*OIDC' -v
go run ./cmd/opencli generate \
  --input ./examples/oidc_user.yaml \
  --runtime-config ./examples/opencli-oidc.yaml \
  --auth oidc \
  --output ./tmp/oidc-go \
  --module github.com/acme/finance-cli \
  --app finance
cd ./tmp/oidc-go && go test ./...
```

预期：全部 PASS。

- [ ] **Step 7: 提交**

```bash
git add internal/render/templates/go/auth_pkce.go.tmpl internal/render/go_project.go internal/render/template_helpers.go internal/render/render.go internal/render/gomod.tmpl internal/render/gosum.tmpl tests/unit/render_test.go
git commit -m "generate go user authorization client"
```

## 8. Task 6：实现 OS Keychain 会话存储

**Files:**

- Add: `internal/render/templates/go/auth_store.go.tmpl`
- Modify: `internal/render/go_project.go`
- Modify: `tests/unit/render_test.go`
- Add: `tests/integration/testdata/oidc/fake_store.go`

- [ ] **Step 1: 先写内存 store 合同测试**

合同覆盖：

- 不存在会话返回 `ErrLoginRequired`。
- Save 后可 Load。
- Delete 幂等。
- 多 provider identity/client_id/tenant/subject 不串用。
- 序列化错误不泄漏 Token。

- [ ] **Step 2: 实现 Keychain key 设计**

Token 隔离维度：

```text
provider identity × client_id × tenant-or-empty × subject
```

Keychain 使用：

```text
service = opencli/<app>/<sha256(providerIdentity|client_id)>
account = session/<sha256(tenant|subject)>
active  = active-session
```

`active-session` 也必须保存在 Keychain，而不是普通配置文件。CLI 本期只允许一个当前用户，
新登录覆盖 active pointer，但不允许 Agent 通过参数选择任意用户。

只有所有受保护 operation 都不是 tenant-bound 时才允许空 tenant。

- [ ] **Step 3: 定义 Session**

```go
type Session struct {
    AccessToken  string
    RefreshToken string
    TokenType    string
    Expiry       time.Time
    IDToken      string
    Subject      string
    TenantID     string
    Scopes       []string
}
```

禁止给 Session 实现会打印字段的 `String()`；所有错误只返回类别和恢复提示。

- [ ] **Step 4: 实现生产 Keychain store**

使用系统 Credential Store：

- macOS Keychain
- Windows Credential Manager
- Linux Secret Service

Keychain 不可用时返回结构化配置错误，不得自动降级到明文文件。

- [ ] **Step 5: 运行生成和测试**

```bash
go test ./tests/unit -run 'TestRender.*OIDCStore' -v
```

预期：PASS。

- [ ] **Step 6: 提交**

```bash
git add internal/render/templates/go/auth_store.go.tmpl internal/render/go_project.go tests/unit/render_test.go tests/integration/testdata/oidc/fake_store.go
git commit -m "store generated oidc sessions in keychain"
```

## 9. Task 7：生成统一认证与身份命令

**Files:**

- Add: `internal/render/templates/go/auth_command.go.tmpl`
- Add: `internal/render/templates/go/identity_command.go.tmpl`
- Modify: `internal/render/templates/go/root_main.go.tmpl`
- Modify: `internal/render/go_project.go`
- Modify: `tests/unit/render_test.go`
- Add: `tests/integration/user_auth_go_cli_test.go`

- [ ] **Step 1: 写命令树失败测试**

生成 CLI 必须包含：

```text
finance auth login
finance auth status
finance auth check --operation <operation>
finance auth logout
finance identity current
```

非用户授权 CLI 不增加这些命令，避免改变现有 UX。

- [ ] **Step 2: 实现 `auth login`**

行为：

1. OIDC 发现并校验元数据；OAuth2 PKCE 校验显式端点。
2. 两种模式生成 state 和 PKCE verifier/challenge；仅 OIDC 生成 nonce。
3. 启动 loopback 随机端口。
4. 打开系统外部浏览器。
5. 等待回调并校验 state。
6. 用 code + verifier 换 Token。
7. OIDC 从验签后的 ID Token 取得 `sub`，按需调用 UserInfo/Identity Endpoint 补充 tenant；OAuth2 PKCE 调用必选的 Identity Endpoint。
8. 保存 Session 到 Keychain。
9. stdout 输出不含 Token 的 JSON。

成功输出：

```json
{
  "authenticated": true,
  "subject": "user-10086",
  "tenant_id": "company-a",
  "scopes": ["expense:read:self"],
  "expires_at": "2026-07-31T15:15:00+08:00"
}
```

身份解析必须遵循：

- Identity Endpoint 返回 `sub`、可选 `tenant_id` 和展示用 `name`。
- OIDC 的端点 `sub` 必须等于 ID Token `sub`。
- tenant-bound operation 缺少可信 `tenant_id` 时登录失败。
- 非 tenant-bound CLI 的 `tenant_id` 可省略，Session 存储键使用空 tenant。

- [ ] **Step 3: 实现 `auth status`、`check`、`identity current`**

`auth check --operation expense.list` 使用生成的 operation → required scopes 映射，
仅检查当前 Session 的授权范围，不代替服务端资源授权。

scope 不足输出：

```json
{
  "error": {
    "type": "authorization",
    "subtype": "insufficient_scope",
    "required_scopes": ["expense:read:self"],
    "retryable": false,
    "hint": "run finance auth login to grant the required scopes"
  }
}
```

- [ ] **Step 4: 实现 `auth logout`**

已配置 revocation endpoint 时，优先撤销 Refresh Token，其次 Access Token；无论远端撤销是否返回
“Token 已失效”，都删除本地会话。网络失败时默认保留本地会话并返回 retryable 错误；
提供显式 `--local-only` 才允许只删除本地凭据。

未配置 revocation endpoint 时删除本地会话，并在结果中明确 `remote_revocation: unsupported`，不得声称远端 Token 已失效。

- [ ] **Step 5: 测试 stdout/stderr 红线**

所有命令测试扫描：

```text
access_token
refresh_token
Authorization:
code_verifier
client_secret
```

除测试夹具本身外，stdout、stderr 和 error message 均不得出现真实值。

- [ ] **Step 6: 运行测试**

```bash
go test ./tests/unit -run 'TestRender.*OIDCCommand' -v
go test ./tests/integration -run 'TestGeneratedGoOIDCAuthCommands' -v
```

预期：PASS。

- [ ] **Step 7: 提交**

```bash
git add internal/render/templates/go/auth_command.go.tmpl internal/render/templates/go/identity_command.go.tmpl internal/render/templates/go/root_main.go.tmpl internal/render/go_project.go tests/unit/render_test.go tests/integration/user_auth_go_cli_test.go
git commit -m "generate oidc auth and identity commands"
```

## 10. Task 8：刷新 Token 并注入受保护业务请求

**Files:**

- Modify: `internal/render/templates/go/auth_pkce.go.tmpl`
- Modify: `internal/render/templates/go/group_service_http.go.tmpl`
- Modify: `internal/render/templates/go/group_command.go.tmpl`
- Modify: `tests/integration/user_auth_go_cli_test.go`

- [ ] **Step 1: 写受保护请求失败测试**

覆盖：

- 未登录：不请求业务 API，返回 `login_required`。
- Access Token 有效：注入 `Authorization: Bearer ...`。
- 即将过期：先 refresh，再调用 API。
- refresh rotation：新 Access/Refresh Token 原子替换旧 Session。
- refresh invalid_grant：删除失效会话并返回 `login_required`。
- operation scope 不足：调用前返回 `insufficient_scope`。
- OpenAPI 无 security 的 operation：不要求登录。
- `--header Authorization: ...`：用户授权 operation 拒绝覆盖。

- [ ] **Step 2: 实现并发安全 Token Manager**

同一进程内多命令/请求同时触发刷新时只允许一个 refresh 请求；
其余调用复用刷新结果。刷新阈值固定为过期前 60 秒，并允许注入 clock 测试。

- [ ] **Step 3: 按 operation security 注入认证**

只对声明对应用户授权 scheme 的 operation 注入用户 Token。
不得根据全局 auth type 给公开 operation 强行加 Token。

保留现有 Client Credentials `applyOAuth2`，新增独立 `applyUserAccessToken`，禁止互相 fallback。

- [ ] **Step 4: 标准化 API 认证错误**

将业务 API 的标准 JSON 错误原样保留关键字段：

```text
type, subtype, message, required_scopes, request_id, retryable, hint
```

不得把 401 自动转换为 application credentials 或静态 Token 重试。

- [ ] **Step 5: 运行 E2E**

```bash
go test ./tests/integration -run 'TestGeneratedGoOIDC(Request|Refresh|Scope|Header)' -v
```

预期：PASS。

- [ ] **Step 6: 提交**

```bash
git add internal/render/templates/go/auth_pkce.go.tmpl internal/render/templates/go/group_service_http.go.tmpl internal/render/templates/go/group_command.go.tmpl tests/integration/user_auth_go_cli_test.go
git commit -m "apply user sessions to protected requests"
```

## 11. Task 9：把 `x-ai-access` 写入 CLI/Skill 契约

**Files:**

- Modify: `internal/render/templates/go/group_command.go.tmpl`
- Modify: `internal/render/templates/skill.md.tmpl`
- Modify: `internal/render/templates/skill_zh.md.tmpl`
- Modify: `internal/render/templates/go/readme.md.tmpl`
- Modify: `tests/unit/render_test.go`

- [ ] **Step 1: 写生成契约失败测试**

断言生成内容包括：

- operation 所需主体固定为 `user`。
- required scopes。
- audience。
- risk 与 confirmation 元数据。
- 不包含 issuer Token、client_secret 或真实用户信息。

- [ ] **Step 2: 生成显式但非安全边界的元数据**

命令 help 和 Skill 中说明：

```text
Authentication: user (OIDC + PKCE)
Required scopes: expense:read:self
Risk: read
Server-side authorization remains authoritative.
```

本期不根据 `risk=high-risk-write` 自动实现确认协议；生成时输出明确的
“高风险治理未启用”诊断并在验收中阻止此 operation 上线，避免形成假安全。

- [ ] **Step 3: 增加 OpenAPI lint 入口**

在 `opencli inspect` 或现有 inspect 输出中增加 `ai-access` 校验结果：

- 受保护 OIDC operation 必须有 `x-ai-access`。
- subject_modes 本期只能是 `[user]`。
- security scopes 与扩展 scopes 必须一致。
- audience 必须与 runtime config 一致。
- identity 参数只做告警：`userId`、`employeeId`、`tenantId`、`role`。

- [ ] **Step 4: 运行测试**

```bash
go test ./tests/unit -run 'TestRender.*AIAccess' -v
go test ./tests/command -run 'TestOpenCLIInspect.*AIAccess' -v
```

预期：PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/render/templates/go/group_command.go.tmpl internal/render/templates/skill.md.tmpl internal/render/templates/skill_zh.md.tmpl internal/render/templates/go/readme.md.tmpl tests/unit/render_test.go tests/command/opencli_inspect_test.go
git commit -m "publish ai access metadata in generated cli"
```

## 12. Task 10：建立可重复的本地用户授权安全 E2E

**Files:**

- Add: `tests/integration/testdata/oidc/server.go`
- Modify: `tests/integration/user_auth_go_cli_test.go`
- Modify: `scripts/smoke.sh`
- Modify: `scripts/SMOKE_TEST.md`

- [ ] **Step 1: 实现进程内假授权服务器**

只用于测试，提供：

```text
/.well-known/openid-configuration
/oauth/authorize
/oauth/token
/oauth/revoke
/oauth/jwks
/api/v1/me
/api/v1/me/expenses
```

服务器记录协议事件但不记录 Token 内容，支持故障注入：

- state mismatch
- code replay
- code expired
- redirect mismatch
- PKCE missing/plain/wrong verifier
- refresh rotation/reuse
- wrong issuer/audience/signature
- insufficient scope

同一测试服务增加无 Discovery、无 ID Token 的 OAuth2 PKCE 配置，用于验证显式端点和 Identity Endpoint；不得让该配置走 OIDC 校验分支。

- [ ] **Step 2: 自动化浏览器与 listener**

E2E 不启动真实浏览器，通过注入 `BrowserOpener` 访问授权 URL；
使用真实 `127.0.0.1:0` loopback 回调验证端口和 path 行为。
Keychain 使用合同一致的 fake store，平台 Keychain 另做手工 smoke。

- [ ] **Step 3: 完成安全矩阵**

至少覆盖设计规范第 21 节中属于 CLI 的用例：

- PKCE S256 成功。
- state 不匹配拒绝。
- code replay/过期拒绝。
- redirect mismatch 拒绝。
- 无 client_secret 成功。
- refresh rotation。
- revoke 后不可刷新。
- OIDC 的 iss/aud/exp/signature/scope 错误。
- OAuth2 PKCE 无 Discovery/ID Token 时仍能登录、识别当前用户并调用 API。
- Token 不出现在输出。
- 不自动切换身份。

- [ ] **Step 4: 运行全量 Go 验证**

```bash
make fmt
make test
./scripts/smoke.sh --target go --input ./examples/oidc_user.yaml
```

预期：全部 PASS；smoke 输出不包含 Token 或 Secret。

- [ ] **Step 5: 提交**

```bash
git add tests/integration/testdata/oidc tests/integration/user_auth_go_cli_test.go scripts/smoke.sh scripts/SMOKE_TEST.md
git commit -m "add user authorization security integration suite"
```

## 13. Task 11：Rust 功能对齐

**前置条件:** M1–M3 完成且 Go E2E 稳定。

**Files:**

- Add: `internal/render/templates/rust/pkce.rs.tmpl`
- Add: `internal/render/templates/rust/oidc_verify.rs.tmpl`
- Add: `internal/render/templates/rust/token_store.rs.tmpl`
- Add: `internal/render/templates/rust/auth_command.rs.tmpl`
- Add: `internal/render/templates/rust/identity_command.rs.tmpl`
- Modify: `internal/render/templates/rust/Cargo.toml.tmpl`
- Modify: `internal/render/templates/rust/main.rs.tmpl`
- Modify: `internal/render/templates/rust/cli.rs.tmpl`
- Modify: `internal/render/templates/rust/client.rs.tmpl`
- Modify: `internal/render/rust_project.go`
- Modify: `tests/integration/rust_generate_smoke_test.go`
- Add: `tests/integration/user_auth_rust_cli_test.go`

- [ ] **Step 1: 把“Rust 暂不支持”测试改为生成成功测试**

先修改测试期望，然后确认失败。

```bash
go test ./tests/command -run 'TestOpenCLIGenerate.*Rust(OIDC|OAuth2PKCE)' -v
```

预期：FAIL，因为护栏仍存在。

- [ ] **Step 2: 生成等价模块和依赖**

Rust 目标使用：

- `oauth2`：两种模式共享的 Authorization Code、PKCE 和 Token 刷新。
- `openidconnect`：仅 OIDC 使用的 Discovery 和 ID Token 校验。
- `keyring`：OS Credential Store。
- `open`：系统浏览器。
- loopback listener 采用小型 HTTP server 或 Tokio TCP，仍只绑定 `127.0.0.1:0`。

不得为了复用现有 `runtime_config.rs` 把用户 Token 放进 sealed config。

- [ ] **Step 3: 复用同一行为契约**

命令、JSON 输出、错误类型、刷新阈值、scope 检查、logout 行为必须与 Go 一致。
不要逐行翻译 Go 实现；共享的是测试向量和外部契约。

- [ ] **Step 4: 复用假授权服务器运行 Rust E2E**

```bash
go test ./tests/integration -run 'TestGeneratedRust(OIDC|OAuth2PKCE)' -v
cargo test --manifest-path ./tmp/oidc-rust/Cargo.toml
cargo test --manifest-path ./tmp/oauth2-pkce-rust/Cargo.toml
```

预期：PASS。

- [ ] **Step 5: 删除 Rust 拒绝护栏**

删除 Task 4 的暂时限制，确保 Rust 的 `oidc` 和 `oauth2-pkce` 都能正常生成。

- [ ] **Step 6: 提交**

```bash
git add internal/render/templates/rust internal/render/rust_project.go tests/command/opencli_generate_test.go tests/integration/rust_generate_smoke_test.go tests/integration/user_auth_rust_cli_test.go
git commit -m "add rust user authorization parity"
```

## 14. Task 12：文档、迁移与发布验收

**Files:**

- Modify: `README.md`
- Add: `docs/auth/oidc.md`
- Add: `docs/auth/business-system-checklist.md`
- Modify: `internal/render/templates/go/readme.md.tmpl`
- Modify: `internal/render/templates/rust/readme.md.tmpl`
- Modify: `Makefile`

- [ ] **Step 1: 写业务系统对接清单**

必须提供：

```text
authorization endpoint
token endpoint
public client_id
API audience
allowed scopes
loopback redirect path
```

OIDC 另外提供 issuer、Discovery 或显式 JWKS，以及 ID Token；Revocation 和 UserInfo 按需提供。

OAuth2 PKCE 另外提供 provider_id 和 Identity Endpoint。opaque Access Token 的 Introspection 由 Resource Server 对接。

明确服务端要求：

- 强制 PKCE S256。
- Public Client Token 请求不要求 client_secret。
- Code 一次性且短期。
- JWT Access Token 校验签名、iss/aud/exp/scope；opaque Token 通过 introspection 或等价机制校验。
- Refresh Token 轮换与复用检测。
- API 做资源/租户级授权。

- [ ] **Step 2: 写 CLI 使用和故障恢复**

只展示：

```bash
opencli generate ... --auth oidc --runtime-config ...
opencli generate ... --auth oauth2-pkce --runtime-config ...
finance auth login
finance auth status
finance auth check --operation expense.list
finance identity current
finance auth logout
```

不展示 username/password、client_secret 或手工粘贴 Token 的替代路径。

- [ ] **Step 3: 增加 release gate**

Makefile 增加聚合目标，顺序执行：

```bash
make fmt
make test
./scripts/smoke.sh --target go --input ./examples/oidc_user.yaml
./scripts/smoke.sh --target rust --input ./examples/oidc_user.yaml
```

Rust 尚未完成 M4 时，release gate 只允许发布标注为 “Go OIDC preview” 的版本，
且 CLI 对 Rust 保持显式拒绝。

- [ ] **Step 4: 人工平台验收**

在 macOS、Windows、Linux 各验证一次：

- Keychain 保存/读取/删除。
- 浏览器打开。
- loopback 防火墙提示和端口绑定。
- logout 撤销。
- 终端 history、process list、日志中无 Token。

- [ ] **Step 5: 最终全量验证**

```bash
make clean
make fmt
make test
make build
git status --short
```

预期：

- 全部命令成功。
- `git status` 只包含计划内文件。
- `dist/`、`tmp/` 不提交。
- 不包含真实 Secret。

- [ ] **Step 6: 最终提交**

```bash
git add README.md docs/auth Makefile internal/render/templates/go/readme.md.tmpl internal/render/templates/rust/readme.md.tmpl
git commit -m "document generated user authorization"
```

## 15. Definition of Done

只有同时满足以下条件才能宣布完成：

- [ ] `oauth2` Client Credentials 现有测试和生成结果保持兼容。
- [ ] `oidc` 只实现 Authorization Code + PKCE，不存在密码模式。
- [ ] `oauth2-pkce` 只使用显式端点，不执行 OIDC Discovery 或要求 ID Token。
- [ ] Public Client 不发送或保存 client_secret。
- [ ] listener 只绑定 loopback 随机端口，回调后立即关闭。
- [ ] 两种用户模式都验证 state 和 PKCE S256；OIDC 额外验证 issuer、ID Token audience、签名、nonce 和 expiry。
- [ ] JWT Access Token 由业务 API 验证签名、issuer、API audience、expiry 和 scope。
- [ ] opaque Access Token 由业务 API 通过 introspection 或等价机制验证有效性、主体、目标资源和 scope。
- [ ] Access/Refresh Token 只保存于 OS Keychain。
- [ ] 未登录、scope 不足和 Token 失效均输出结构化错误。
- [ ] 用户授权保护的 operation 不能覆盖 Authorization Header。
- [ ] 用户认证失败后不 fallback 到 oauth2、Token、AK/SK 或 API Key。
- [ ] 公开 operation 不强制登录。
- [ ] `auth login/status/check/logout` 与 `identity current` 可用。
- [ ] `x-ai-access` 被严格解析，但不冒充服务端授权。
- [ ] Go 和 Rust 都通过同一安全测试矩阵；若 Rust 未完成则生成器明确拒绝。
- [ ] stdout、stderr、日志、Skill、README 和生成配置中没有 Token 或 Secret。
- [ ] `make test`、`make build` 和两种用户授权 smoke test 全部通过。

## 16. 建议执行顺序与工作量

| 周期 | 工作内容 | 参考工作量 |
| --- | --- | --- |
| 第 1 周 | Task 1–4：模型、OpenAPI、runtime config、生成护栏 | 5 人日 |
| 第 2 周 | Task 5–7：Go PKCE、Keychain、认证命令 | 7 人日 |
| 第 3 周 | Task 8–10：请求注入、AI 契约、安全 E2E | 7 人日 |
| 第 4 周 | Task 11：Rust 对齐 | 6 人日 |
| 第 5 周 | Task 12：跨平台验收、文档、发布 | 4 人日 |

建议人员配置：

- 1 名 one-cli/生成器开发。
- 1 名 OAuth/OIDC 与安全开发，可与前者同人但需要独立评审。
- 1 名 QA 兼职搭建协议与跨平台矩阵。
- 业务系统团队提供可联调的授权服务和 `/me` API，不进入 one-cli 代码范围。

最大风险不是 PKCE 算法实现，而是“生成了能登录的 CLI，却没有强制业务 API 做 audience、scope、租户和资源级授权”。因此发布验收必须同时要求至少一个真实业务系统通过越权测试；仅 one-cli 测试通过不代表端到端授权方案上线。
