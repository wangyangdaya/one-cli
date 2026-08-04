# OAuth 2.0 Authorization Code 接口契约

本文定义 one-cli 与授权服务器之间的标准 OAuth 2.0 Authorization Code 接口。业务授权服务优先通过适配层提供标准契约；仅参数名、位置或 JSON envelope 不同时，也可使用 one-cli 的 `token_exchange` 显式映射，CLI 不会自动猜测或探测业务格式。

规范依据：[RFC 6749](https://www.rfc-editor.org/rfc/rfc6749)、[RFC 7636](https://www.rfc-editor.org/rfc/rfc7636) 和 [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html)。PKCE 与 OIDC 均为可选能力；未配置时保持基础 Authorization Code 流程。

## 1. 接口

| 接口 | 配置项 | 用途 |
| --- | --- | --- |
| Authorization Endpoint | `authorization_url` | 浏览器登录、用户授权并签发 Authorization Code |
| Token Endpoint | `token_url` | Authorization Code 换 Token，以及 Refresh Token 续期 |
| Redirect Endpoint | `redirect_uri` | 授权服务器将浏览器重定向回 CLI 的本地回调地址；配置可固定，也可由 CLI 动态生成 |

### 回调地址模式

| 模式 | 配置 | CLI 行为 | 适用场景 |
| --- | --- | --- | --- |
| 固定端口 | 配置完整 `redirect_uri` | 监听配置中的主机、端口和路径 | 授权服务器要求预先登记完整回调地址，例如飞书 |
| 随机端口 | 省略 `redirect_uri` | 绑定 `127.0.0.1:0`，由操作系统分配当前未占用端口，路径使用 `/oauth/callback` | 授权服务器允许 loopback 回调使用动态端口 |

随机端口示例：

```text
http://127.0.0.1:51004/oauth/callback
```

端口分配完成后，CLI 才能构造授权链接。该次登录生成的实际 `redirect_uri` 必须同时用于：

1. Authorization Endpoint 的授权请求。
2. 本地 HTTP 回调监听。
3. Authorization Code 换 Token 请求。

三处地址必须完全一致。随机端口只解决本机端口占用问题；授权服务器本身必须允许 loopback URI 使用动态端口。该行为符合 [RFC 8252 第 7.3 节](https://datatracker.ietf.org/doc/html/rfc8252#section-7.3)。

## 2. 发起授权

客户端使用浏览器访问 `authorization_url`：

```http
GET /authorize?response_type=code&client_id=cli_example&redirect_uri=http%3A%2F%2F127.0.0.1%3A18081%2Foauth%2Fcallback&state=random-state HTTP/1.1
Host: authorization.example.com
```

### Query 参数

| 参数 | 必填 | 定义 |
| --- | --- | --- |
| `response_type` | 是 | 固定为 `code` |
| `client_id` | 是 | 授权服务器分配的客户端标识 |
| `redirect_uri` | 是 | 本次登录实际使用的 CLI 本地回调地址；必须与本地监听地址及换 Token 时的值完全一致 |
| `scope` | 否 | 空格分隔的权限集合；未配置时必须完全省略，不能发送空值 |
| `state` | 是 | CLI 生成的不可预测随机值，用于关联请求并防止回调伪造 |
| `code_challenge` | PKCE 开启时 | `BASE64URL(SHA256(code_verifier))` |
| `code_challenge_method` | PKCE 开启时 | 固定为 `S256`，不支持 `plain` |
| `nonce` | OIDC 开启时 | CLI 为本次登录生成的随机值，必须出现在返回的 ID Token 中 |

## 3. 授权回调

### 成功回调

授权服务器重定向到 `redirect_uri`：

```http
HTTP/1.1 302 Found
Location: http://127.0.0.1:18081/oauth/callback?code=authorization-code&state=random-state
```

| 参数 | 必填 | 定义 |
| --- | --- | --- |
| `code` | 是 | 一次性 Authorization Code，应短时有效且只能使用一次 |
| `state` | 是 | 必须与发起授权时的 `state` 完全一致 |

CLI 必须先校验 `state`，校验成功后才能使用 `code`。缺少或不匹配时必须终止登录。

### 失败回调

```http
HTTP/1.1 302 Found
Location: http://127.0.0.1:18081/oauth/callback?error=access_denied&error_description=user+denied&state=random-state
```

| 参数 | 必填 | 定义 |
| --- | --- | --- |
| `error` | 是 | OAuth 错误码 |
| `error_description` | 否 | 面向开发者的错误说明 |
| `error_uri` | 否 | 错误说明文档地址 |
| `state` | 是 | 原授权请求携带了 `state` 时必须原样返回 |

常见授权错误：`invalid_request`、`unauthorized_client`、`access_denied`、`unsupported_response_type`、`invalid_scope`、`server_error`、`temporarily_unavailable`。

## 4. Authorization Code 换 Token

客户端向 `token_url` 发送表单请求：

```http
POST /oauth/token HTTP/1.1
Host: token.example.com
Content-Type: application/x-www-form-urlencoded
Accept: application/json

grant_type=authorization_code&code=authorization-code&client_id=cli_example&redirect_uri=http%3A%2F%2F127.0.0.1%3A18081%2Foauth%2Fcallback
```

### 表单参数

| 参数 | 必填 | 定义 |
| --- | --- | --- |
| `grant_type` | 是 | 固定为 `authorization_code` |
| `code` | 是 | 回调收到的 Authorization Code |
| `client_id` | 是 | 与发起授权时相同的客户端标识 |
| `redirect_uri` | 是 | 与发起授权时完全相同的回调地址 |
| `code_verifier` | PKCE 开启时 | 本次登录保存在内存中的原始 verifier；不得写入 URL、日志或 token 文件 |

CLI 属于不能安全保存服务端密钥的客户端，不在本地配置或发送 `client_secret`。业务方需要 `client_secret` 时，由服务端授权适配层持有并完成上游请求。

### 成功响应

```http
HTTP/1.1 200 OK
Content-Type: application/json
Cache-Control: no-store
Pragma: no-cache

{
  "access_token": "access-token",
  "token_type": "Bearer",
  "expires_in": 7200,
  "refresh_token": "refresh-token"
}
```

| 字段 | 必填 | 定义 |
| --- | --- | --- |
| `access_token` | 是 | 访问受保护业务接口的 Token |
| `token_type` | 是 | 通常为 `Bearer`，大小写不敏感 |
| `expires_in` | 建议 | Access Token 从响应时刻开始计算的有效秒数 |
| `refresh_token` | 否 | 用于续期 Access Token；请求离线访问能力时通常返回 |
| `scope` | 条件必填 | 实际授权范围与请求范围不一致时必须返回 |
| `id_token` | OIDC 开启时 | 授权服务器签发的 ID Token；CLI 校验后仅用于确认身份结果，不作为业务 API Bearer Token |

Token 响应不得被缓存。服务端应返回 `Cache-Control: no-store` 和 `Pragma: no-cache`。

## 5. Refresh Token 续期

```http
POST /oauth/token HTTP/1.1
Host: token.example.com
Content-Type: application/x-www-form-urlencoded
Accept: application/json

grant_type=refresh_token&refresh_token=refresh-token&client_id=cli_example
```

### 表单参数

| 参数 | 必填 | 定义 |
| --- | --- | --- |
| `grant_type` | 是 | 固定为 `refresh_token` |
| `refresh_token` | 是 | 登录或上一次续期返回的 Refresh Token |
| `client_id` | 是 | 当前 CLI 的客户端标识 |
| `scope` | 否 | 不得包含原授权范围之外的权限；省略表示沿用原范围 |

成功响应与 Code 换 Token 相同。OAuth 标准允许服务端不轮换 Refresh Token；但 one-cli 当前运行时要求刷新响应返回新的非空 `access_token` 和 `refresh_token`，并原子替换旧值。需要兼容当前 CLI 的授权服务必须在每次刷新时轮换并返回 Refresh Token。

当前 CLI 还需要登录或刷新响应提供正数 `refresh_token_expires_in`，才能把会话识别为可刷新。缺少该字段时，Access Token 到期后会要求重新登录。

刷新请求不发送 `code_verifier`、`code_challenge` 或 `nonce`。这些值只属于单次浏览器授权登录。

## 6. Token Endpoint 错误响应

```http
HTTP/1.1 400 Bad Request
Content-Type: application/json
Cache-Control: no-store
Pragma: no-cache

{
  "error": "invalid_grant",
  "error_description": "authorization code expired"
}
```

| 字段 | 必填 | 定义 |
| --- | --- | --- |
| `error` | 是 | OAuth 错误码 |
| `error_description` | 否 | 面向开发者的错误说明，不得包含 Token 或密钥 |
| `error_uri` | 否 | 错误说明文档地址 |

标准错误码：

| 错误码 | 含义 |
| --- | --- |
| `invalid_request` | 缺少参数、参数重复或请求格式错误 |
| `invalid_client` | 客户端身份无效；必要时可返回 HTTP 401 |
| `invalid_grant` | Code 或 Refresh Token 无效、过期、撤销、已使用，或 `redirect_uri` 不匹配 |
| `unauthorized_client` | 当前客户端不允许使用该授权类型 |
| `unsupported_grant_type` | 不支持请求中的 `grant_type` |
| `invalid_scope` | 请求的权限范围无效或超出原授权范围 |

CLI 处理规则：

- `invalid_grant`：当前凭证不可继续使用，返回 `login_required`。
- 网络错误、超时或 HTTP 5xx：保留已有 Token，返回 `refresh_failed`，不得自动清除登录态。
- 错误响应中的 Token、Authorization Code 和服务端密钥不得写入日志。

## 7. one-cli 配置示例

```yaml
base_url: https://api.example.com

auth:
  type: oauth2
  grant_type: authorization_code
  client_id: cli_example
  authorization_url: https://authorization.example.com/authorize
  token_url: https://token.example.com/oauth/token
  redirect_uri: http://127.0.0.1:18081/oauth/callback
  pkce:
    enabled: true
    method: S256
```

其中 `token_url` 优先使用符合本文定义的标准化 Token Endpoint。业务方原始接口字段不同时，可由 Endpoint 背后的适配层转换，也可使用下文的显式 `token_exchange` 映射。

允许动态 loopback 端口的业务系统可以省略 `redirect_uri`：

```yaml
auth:
  type: oauth2
  grant_type: authorization_code
  client_id: cli_example
  authorization_url: https://authorization.example.com/authorize
  token_url: https://token.example.com/oauth/token
```

省略后不是不发送 `redirect_uri`，而是由 CLI 获取可用端口、生成实际地址，再把该地址发送给授权服务器。

只有授权服务器明确要求权限参数时才配置：

```yaml
auth:
  scopes:
    - offline_access
```

未配置 `scopes` 或配置为空列表时，CLI 必须从授权 URL 和 Token 请求中省略 `scope` 参数。

开启 OIDC 时必须配置 `issuer`，并在 scopes 中包含 `openid`：

```yaml
auth:
  type: oauth2
  grant_type: authorization_code
  client_id: cli_example
  authorization_url: https://identity.example.com/authorize
  token_url: https://identity.example.com/token
  scopes: [openid, profile]
  pkce:
    enabled: true
  oidc:
    enabled: true
    issuer: https://identity.example.com
```

`pkce.method` 省略时默认为 `S256`。OIDC 登录会从 `<issuer>/.well-known/openid-configuration` 获取 `jwks_uri`，目前最小实现只接受 `RS256`，并校验签名、`iss`、`aud`、`azp`、`exp`、`iat` 和 `nonce`。

业务 Token 接口使用自定义请求或响应结构时，`token_exchange.parameters[].source` 额外支持 `code_verifier`，`token_exchange.response` 额外支持 `id_token`。PKCE 与自定义交换同时开启时必须显式映射 `code_verifier`。

## 8. 实现边界

- 使用 Authorization Code，不使用 Implicit 或 Password Grant。
- PKCE 可选，开启后只支持 `S256`，不允许降级为 `plain`。
- OIDC 可选，开启后必须包含 `openid` scope，并要求 Token 响应包含可验证的 `id_token`。
- OIDC 当前支持规范要求客户端必须支持的 `RS256`；其他 ID Token 签名算法暂不接受。
- Authorization Code 必须一次性、短时有效。
- 固定模式必须匹配已注册的完整 `redirect_uri`；随机模式必须由授权服务器允许 loopback 动态端口。
- 每次登录在授权请求、本地监听和 Code 换 Token 三处使用的实际 `redirect_uri` 必须完全一致。
- CLI 必须校验 `state`。
- 生产环境的远程授权和 Token Endpoint 必须使用 HTTPS；`http://127.0.0.1` 仅用于本机回调或本地验证服务。
