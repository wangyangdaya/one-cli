# one-cli OIDC 用户认证对接与本地验证指南

## 1. 文档目的

本文面向业务系统、身份平台和 one-cli 开发团队，说明如何把个人数据 API 接入终端 Agent。

目标协议是 OIDC + OAuth 2.0 Authorization Code + PKCE。CLI 属于 Public Client，不持有 `client_secret`，也不接收用户账号密码。

本文可以独立阅读。总体安全架构参见《企业业务系统 AI Native 认证授权架构与开发规范》。

## 2. 适用范围

适用于终端电脑上的 Agent 代表当前用户访问个人数据，例如查询本人费用、考勤、审批和提交本人申请。

不适用于服务器 Agent、Device Flow、应用权限、工作负载身份、Client Credentials、AK/SK 或 API Key。

业务 API 必须根据 Token 主体完成租户、资源、数据范围和操作权限校验。CLI 不是最终安全边界。

## 3. `oidc` 与 `oauth2` 的区别

one-cli 中两个名称代表不同主体，不得互相降级：

| Generate 参数 | 主体 | 协议 | 用途 |
| --- | --- | --- | --- |
| `--auth oidc` | 当前用户 | Authorization Code + PKCE | 个人数据和用户委托 |
| `--auth oauth2` | 应用 | Client Credentials | 非个人应用调用 |

`--auth oauth2` 是已有能力，仍表示 Client Credentials。

`--auth oidc` 是用户认证目标契约。用户认证失败后，不得自动改用 `oauth2`、静态 Token、AK/SK 或 API Key。

## 4. 当前实施状态

截至 2026-07-31：

| 能力 | 状态 |
| --- | --- |
| 根目录 FastAPI OIDC/OAuth2 验证服务 | 已实现 |
| Discovery、JWKS、UserInfo、PKCE、Refresh、Revoke | 已实现 |
| 个人费用 API 和用户数据隔离 | 已实现 |
| `oauth2/openapi.yaml` 与 runtime 示例 | 已实现 |
| one-cli `--auth oidc` 参数和生成模板 | 待实施 |
| 生成 CLI 的 Keychain 和 `auth` 命令 | 待实施 |

因此，本文中的 `--auth oidc` 命令是已确认的目标接口。当前分支需完成 OIDC CLI Provider 后才能执行完整端到端流程。

## 5. CLI Generate 参数

目标命令：

```bash
opencli generate \
  --input ./business.openapi.yaml \
  --output ./business-cli \
  --app business \
  --module example.com/business-cli \
  --auth oidc \
  --runtime-config ./business.runtime.yaml
```

参数职责：

- `--auth oidc`：选择用户级 OIDC + PKCE Provider。
- `--runtime-config`：提供业务 API 和 Public Client 的公开配置。
- `--input`：提供 operation、Security Scheme、scope 和 `x-ai-access`。
- `--target`：选择 Go 或 Rust 生成目标；未实现的目标必须明确拒绝，不能生成不完整认证代码。

CLI 不通过 Generate 参数接收授权端点、用户名、密码、Token 或 `client_secret`。

## 6. Runtime 配置

优先只配置 issuer：

```yaml
base_url: https://business-api.example.com

auth:
  type: oidc
  issuer: https://business-auth.example.com
  client_id: business-cli
  audience: business-api
  scopes:
    - openid
    - profile
    - expense:read:self

  redirect:
    type: loopback
    path: /oauth/callback

  token_store: os_keychain
```

字段含义：

| 字段 | 含义 |
| --- | --- |
| `base_url` | 业务 Resource Server 地址 |
| `issuer` | OIDC Authorization Server 标识和 Discovery 基地址 |
| `client_id` | 已注册的 Public CLI Client |
| `audience` | Access Token 允许访问的业务 API |
| `scopes` | CLI 申请的用户权限 |
| `redirect.path` | loopback 固定回调路径 |
| `token_store` | OS Keychain/Credential Store |

CLI 根据 issuer 请求：

```text
<issuer>/.well-known/openid-configuration
```

Discovery 返回 authorize、token、revoke、JWKS 和 UserInfo 地址。

业务系统暂时不支持 Discovery 时，才显式配置：

```yaml
auth:
  type: oidc
  client_id: business-cli
  audience: business-api
  authorization_endpoint: https://business.example.com/oauth/authorize
  token_endpoint: https://business.example.com/oauth/token
  revocation_endpoint: https://business.example.com/oauth/revoke
  jwks_uri: https://business.example.com/oauth/jwks
```

生产配置必须使用 HTTPS。本仓库的 `127.0.0.1` HTTP 服务仅用于本地验证。

## 7. CLI 登录流程

目标命令：

```bash
business auth login
```

处理顺序：

1. CLI 读取 issuer 并校验 Discovery。
2. CLI 生成 `state`、`nonce` 和 PKCE verifier/challenge。
3. CLI 监听 `127.0.0.1` 随机端口，回调路径固定。
4. CLI 使用系统浏览器打开 authorize endpoint。
5. 用户在业务授权页面完成登录和授权确认。
6. 授权服务把一次性 code 返回 loopback callback。
7. CLI 校验 state，并使用 code + verifier 调用 token endpoint。
8. CLI 校验 ID Token，把 Access/Refresh Token 保存到 OS Keychain。
9. 业务命令使用 Access Token 调用 API，过期前通过 Refresh Token 轮换。

受保护业务命令未登录时返回结构化 `login_required`，不应在后台自动打开浏览器。

账号密码只进入业务授权页面。CLI、Agent、runtime YAML 和命令行参数均不得接触密码。

## 8. 生成 CLI 契约

目标命令：

```bash
business auth login
business auth status
business auth check --operation <operation>
business auth logout
business identity current
```

行为要求：

- `auth status` 不输出 Token。
- `auth check` 只检查登录状态和 scope，不代替服务端资源授权。
- `auth logout` 调用 revoke 后删除 Keychain 会话。
- 受保护 operation 不允许调用方覆盖 `Authorization` Header。
- 用户认证失败后不得切换为应用身份。

Token 存储隔离维度：

```text
issuer × client_id × tenant × user
```

## 9. 业务授权服务接口

业务系统至少提供：

| 接口 | 必选 | 说明 |
| --- | --- | --- |
| `GET /.well-known/openid-configuration` | 是 | OIDC/OAuth 元数据 |
| `GET /oauth/authorize` | 是 | 浏览器登录与授权 |
| `POST /oauth/token` | 是 | Code/Refresh Token 交换 |
| `POST /oauth/revoke` | 是 | 登出与撤销 |
| `GET /oauth/jwks` | JWT 时必选 | 公钥和轮换 |
| `GET /oauth/userinfo` | 建议 | OIDC 用户信息 |
| `GET /api/v1/me` | 建议 | 业务主体映射 |

Public Client 注册要求：

```yaml
client_id: business-cli
client_type: public
grant_types: [authorization_code, refresh_token]
redirect:
  type: loopback
  path: /oauth/callback
pkce:
  required: true
  methods: [S256]
```

Token Endpoint 不得要求 Public Client 提供 `client_secret`。

## 10. OpenAPI 规范

Security Scheme：

```yaml
components:
  securitySchemes:
    userOIDC:
      type: oauth2
      flows:
        authorizationCode:
          authorizationUrl: https://business.example.com/oauth/authorize
          tokenUrl: https://business.example.com/oauth/token
          scopes:
            expense:read:self: 查询本人费用
```

Operation：

```yaml
paths:
  /api/v1/me/expenses:
    get:
      operationId: listMyExpenses
      security:
        - userOIDC:
            - expense:read:self
      x-ai-access:
        subject_modes: [user]
        audience: business-api
        scopes: [expense:read:self]
        tenant_bound: true
        resource_type: expense
        data_classification: personal
        risk: read
        confirmation: never
```

OpenAPI 只声明所需权限。API 必须从 Token 的 `sub`、tenant 和 scope 派生真实访问范围。

## 11. 本地验证服务

仓库根目录的 `oauth2/` 是 FastAPI 验证服务，与 `sql-mcp-server/` 同级。

启动：

```bash
cd oauth2
uv sync
uv run oauth2-server
```

固定地址：

```text
issuer/API: http://127.0.0.1:18080
client_id:  one-cli-demo
audience:   demo-api
```

测试用户：

| 用户 | 密码 | Token subject |
| --- | --- | --- |
| Alice | `alice123` | `user-alice` |
| Bob | `bob123` | `user-bob` |

服务包含：

- Authorization Code + PKCE S256；
- RS256 Access Token 和 ID Token；
- Discovery、JWKS、UserInfo；
- Refresh Token 轮换、重用检测和撤销；
- Alice/Bob 个人费用隔离；
- `oauth2/openapi.yaml`；
- `oauth2/opencli.runtime.yaml`。

服务重启后，内存 code、session、Token 状态和 RSA 密钥全部失效。

## 12. 测试与验收

验证服务：

```bash
cd oauth2
uv run pytest -q
uv run ruff check .
uv run ruff format --check .
```

业务系统验收至少覆盖：

- 缺少 PKCE、`plain`、错误 verifier 被拒绝；
- code 过期和重放被拒绝；
- redirect URI 不匹配被拒绝；
- Refresh Token 轮换和旧 Token 重用；
- revoke 后不能继续刷新；
- 错误签名、issuer、audience、expiry 和 scope；
- 普通用户不能读取他人数据；
- Token、密码和 Authorization Header 不进入日志。

## 13. 安全红线

- 不使用 Resource Owner Password Grant。
- 不使用 Implicit Grant。
- Public Client 不依赖 `client_secret`。
- 不把 Token 写入 YAML、命令参数、stdout、stderr 或 Skill。
- 不允许 Agent 指定可信 user、tenant、role 或 Authorization Header。
- 不用 CLI 参数隐藏代替服务端授权。
- 不接受其他业务系统 audience 的 Token。
- 不在认证失败后自动降级为高权限应用身份。

## 14. 后续工作

one-cli 侧下一步按以下顺序实施：

1. 增加独立 `AuthTypeOIDC`，保留 `oauth2=client_credentials`。
2. 解析 Authorization Code Security Scheme 和 operation scopes。
3. 支持公开 OIDC runtime 配置，禁止 Secret 字段。
4. 生成 PKCE、loopback、Keychain 和 `auth`/`identity` 命令。
5. 把本地验证服务接入生成 CLI 端到端测试。
6. Go 目标稳定后再完成 Rust 对齐。

详细任务参见 `docs/superpowers/plans/2026-07-31-oidc-pkce-generated-cli.md`。
