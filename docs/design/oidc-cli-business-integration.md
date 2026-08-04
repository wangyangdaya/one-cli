# one-cli 用户授权对接与本地验证指南

## 1. 当前实现结论

one-cli 当前通过统一的 `--auth oauth2` 生成 OAuth2 运行时，具体流程由 `runtime.yaml` 的 `auth.grant_type` 决定：

| Runtime 配置 | 主体 | 当前支持 |
| --- | --- | --- |
| `grant_type: client_credentials` | 应用 | Client Credentials，支持 Basic、Body、Query 三种 Client Secret 位置 |
| `grant_type: authorization_code` | 当前用户 | 浏览器授权、loopback 回调、Token 本地保存与自动刷新 |
| Authorization Code + `pkce.enabled: true` | 当前用户 | PKCE S256 |
| Authorization Code + `oidc.enabled: true` | 当前用户 | nonce、Discovery/JWKS、RS256 ID Token 校验 |

当前没有独立的 `--auth oidc`、`--auth oauth2-pkce` 或 `--auth iam-code`。OIDC 和 PKCE 是 `oauth2 + authorization_code` 的显式可选能力。

生成的用户登录命令位于顶层：

```bash
business login
business login --no-browser
business status
business logout
```

当前不生成 `business auth login`、`auth check` 或 `identity current`。

## 2. 生成命令

```bash
opencli generate \
  --input ./business.openapi.yaml \
  --output ./business-cli \
  --module example.com/business-cli \
  --app business \
  --auth oauth2 \
  --runtime-config ./business.runtime.yaml
```

Go 和 Rust 目标均支持 Authorization Code、PKCE 和 OIDC：

```bash
# Go，默认目标
opencli generate ... --target go

# Rust
opencli generate ... --target rust
```

CLI 属于 Public Client。Authorization Code 配置不得包含 `client_secret`，用户密码、MFA 和 SSO 只应出现在授权服务器的浏览器页面。

## 3. 基础 Authorization Code

```yaml
base_url: https://business-api.example.com

auth:
  type: oauth2
  grant_type: authorization_code
  client_id: business-cli
  authorization_url: https://identity.example.com/authorize
  token_url: https://identity.example.com/token
  redirect_uri: http://127.0.0.1:18081/oauth/callback
  scopes:
    - profile
```

`scopes` 可省略。省略时授权 URL 不发送空的 `scope` 参数。

`redirect_uri` 也可省略：CLI 会监听 `127.0.0.1:0`，由操作系统分配空闲端口，并生成 `http://127.0.0.1:<port>/oauth/callback`。只有授权服务器允许原生应用 loopback 动态端口时才能使用这种模式；飞书等要求精确登记完整回调地址的系统必须配置固定地址。

同一次登录中，以下三处必须使用完全相同的实际 redirect URI：

1. Authorization Endpoint 请求。
2. CLI 本地回调监听。
3. Authorization Code 换 Token 请求。

## 4. 开启 PKCE

```yaml
auth:
  type: oauth2
  grant_type: authorization_code
  client_id: business-cli
  authorization_url: https://identity.example.com/authorize
  token_url: https://identity.example.com/token
  pkce:
    enabled: true
    method: S256
```

实现规则：

- 只支持 `S256`，`method` 省略时默认为 `S256`，不允许降级为 `plain`。
- 每次登录生成新的 32 字节随机 verifier，Base64URL 无填充编码后为 43 个字符。
- 授权请求发送 `code_challenge` 和 `code_challenge_method=S256`。
- Code 换 Token 发送原始 `code_verifier`。
- verifier 只存在于当前登录进程内，不写入授权 URL、日志或 token 文件。
- Refresh Token 请求不发送 PKCE 参数。

## 5. 开启 OIDC

```yaml
base_url: https://business-api.example.com

auth:
  type: oauth2
  grant_type: authorization_code
  client_id: business-cli
  authorization_url: https://identity.example.com/authorize
  token_url: https://identity.example.com/token
  scopes:
    - openid
    - profile
  pkce:
    enabled: true
  oidc:
    enabled: true
    issuer: https://identity.example.com
```

OIDC 必须显式开启，不会根据 `openid` scope 自动推断。开启时必须满足：

- scopes 包含 `openid`。
- `issuer` 是无 userinfo、query、fragment 的绝对 URL；生产使用 HTTPS，本地验证允许 HTTP loopback。
- Token 响应包含 `id_token`。
- `<issuer>/.well-known/openid-configuration` 返回与配置完全一致的 `issuer` 和有效 `jwks_uri`。
- `jwks_uri` 使用 HTTPS；本地验证允许 `http://127.0.0.1` 或 `http://localhost`。

当前 ID Token 校验范围：

- RS256 签名和匹配的 JWKS key；
- `iss` 等于配置的 issuer；
- `aud` 包含当前 `client_id`；
- 多 audience 或存在 `azp` 时，`azp` 等于当前 `client_id`；
- `exp` 未过期，允许 60 秒时钟偏差；
- `iat` 存在且不能超过当前时间 60 秒；
- `nonce` 与本次授权请求完全一致。

任何 OIDC 校验失败都会终止登录，并且不会保存新返回的 Access/Refresh Token。ID Token 校验后不会持久化，也不会作为业务 API Bearer Token。

当前只支持 RS256，不支持 ES256、UserInfo、`identity current` 或多账号身份展示。

## 6. Token Endpoint 标准契约

默认 Code 交换请求：

```http
POST /token
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code
&client_id=business-cli
&code=<authorization-code>
&redirect_uri=<本次实际回调地址>
&code_verifier=<PKCE开启时发送>
```

标准响应：

```json
{
  "access_token": "access-token",
  "refresh_token": "refresh-token",
  "token_type": "Bearer",
  "scope": "openid profile",
  "expires_in": 3600,
  "refresh_token_expires_in": 604800,
  "id_token": "OIDC开启时返回"
}
```

`access_token` 必填；非空 `token_type` 必须为 `Bearer`。OIDC 开启时 `id_token` 必填。

业务服务需要使用 `client_secret` 对接上游身份平台时，Secret 必须只保留在业务服务或 broker 中，CLI 的 Token Endpoint 面向 Public Client 提供上述无 Secret 契约。

## 7. 自定义业务 Token 接口

已有业务接口参数名、位置或响应 envelope 不标准时，可配置 `token_exchange`：

```yaml
auth:
  type: oauth2
  grant_type: authorization_code
  client_id: business-cli
  authorization_url: https://identity.example.com/authorize
  token_url: https://business.example.com/cli-auth/exchange
  scopes: [openid]
  pkce: {enabled: true}
  oidc: {enabled: true, issuer: https://identity.example.com}

  token_exchange:
    method: POST
    body_format: json
    parameters:
      - {source: code, name: authorizationCode, in: body, required: true}
      - {source: code_verifier, name: verifier, in: body, required: true}
      - {source: state, name: loginState, in: header, required: true}
    response:
      access_token: {in: body, path: data.accessToken}
      refresh_token: {in: body, path: data.refreshToken}
      token_type: {in: body, path: data.tokenType}
      expires_in: {in: body, path: data.expiresIn}
      refresh_token_expires_in: {in: body, path: data.refreshExpiresIn}
      id_token: {in: body, path: data.idToken}
```

请求 source 支持：`code`、`code_verifier`、`state`、`client_id`、`redirect_uri`、`scope`、`grant_type`、`literal`。位置支持 `body`、`query`、`header`、`cookie`；Body 支持 `form` 和 `json`。

响应支持从 body 点路径或 header 映射。PKCE 与自定义交换同时开启时必须显式映射 `code_verifier`。

注意：当前自动刷新固定调用同一个 `token_url`，使用标准 `application/x-www-form-urlencoded` Refresh Grant 和标准顶层响应，不复用自定义 Code 交换映射。需要自动刷新的业务服务应让该地址同时兼容标准 Refresh Grant。

## 8. 登录、Agent 与超时

```bash
# 默认打印链接并尝试打开浏览器
business login

# 只打印链接，不自动打开浏览器；仍会等待回调
business login --no-browser
```

`--no-browser` 适合 Agent 场景：Agent 在可持续轮询的终端会话中启动命令，把 stdout 第一行的授权链接交给用户点击，并继续保持同一个 CLI 进程。用户授权后，本地回调会唤醒该进程，CLI 完成 Token 交换并输出 `login successful`。

CLI 等待回调 120 秒。超过时间仍未收到有效回调时退出并返回 `OAuth login timed out`。

Agent 如果无法维持同一个后台进程或终端会话，就无法接收回调结果；这属于 Agent 执行环境限制，不是 OAuth 协议要求用户必须手动运行 CLI。

## 9. 会话文件、status 和刷新

默认 token 文件：

```text
$HOME/.opencli/oauth2/<配置哈希>/oauth-token.json
```

配置哈希由 `client_id`、`authorization_url` 和 `token_url` 计算。macOS、Linux 和 Windows 都使用当前用户目录；可通过以下变量覆盖完整文件路径：

```bash
export OPENCLI_OAUTH_TOKEN_FILE=/path/to/oauth-token.json
```

Unix 下目录权限为 `0700`，token 文件为 `0600`。当前使用用户目录文件存储，不是 OS Keychain/Credential Store。

`status` 只读取本地文件，不访问网络，也不主动刷新：

| 输出 | 含义 |
| --- | --- |
| `valid` | Access Token 剩余时间超过 5 分钟 |
| `needs_refresh` | Access Token 即将过期或已过期，但 Refresh Token 仍有效 |
| `expired` | 当前会话不能继续刷新，需要重新登录 |
| `not_logged_in` | token 文件不存在 |

受保护业务命令发现 `needs_refresh` 时自动刷新并原子替换 Access/Refresh Token。当前不会因为业务 API 返回 401 自动重放请求。

`logout` 只删除本地会话文件，当前不调用远程 revocation endpoint。

## 10. 业务系统安全责任

- Authorization Code 必须短时、一次性，并绑定 `client_id` 和实际 redirect URI。
- 开启 PKCE 时，服务端必须把 Code 绑定到 challenge，并在 Token Endpoint 校验 verifier。
- 业务 API 必须验证 Access Token 的签名或 introspection 结果、issuer、audience、有效期和 scope。
- Resource Server 必须从可信 Token 主体派生 user、tenant 和数据范围，不能信任 Agent 传入的 userId、employeeId、tenant 或 role。
- 不使用 Resource Owner Password Grant 或 Implicit Grant。
- Token、Code、Cookie、密码和 Secret 不得进入日志、stdout、stderr 或 Skill。
- 用户授权失败后不得自动降级为 Client Credentials、AK/SK、API Key 或共享 Token。

## 11. 当前边界

已实现：

- Go/Rust Authorization Code；
- 固定或动态 loopback；
- state；
- PKCE S256；
- OIDC Discovery/JWKS 和 RS256 ID Token 校验；
- Access/Refresh Token 文件存储；
- `login`、`status`、`logout`；
- 到期前自动 Refresh；
- 自定义 Code 交换请求/响应映射。

未实现：

- 独立 `--auth oidc` / `--auth oauth2-pkce`；
- OS Keychain/Credential Store；
- UserInfo、`identity current`、多账号；
- `auth check`、operation scope 本地检查；
- 远程 revoke；
- Device Flow；
- PKCE `plain` 或 RS256 之外的 ID Token 算法；
- 自定义 Refresh Grant 映射和业务 API 401 自动重试。

标准接口细节见 [`../../oauth2/OAUTH2_AUTHORIZATION_CODE_CONTRACT.md`](../../oauth2/OAUTH2_AUTHORIZATION_CODE_CONTRACT.md)。飞书验证流程见 [`../../oauth2/README.md`](../../oauth2/README.md)。
