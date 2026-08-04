# OAuth 2.0 Client Credentials 生成设计

> 本文只描述应用身份的 Client Credentials。当前用户授权使用同一个 `--auth oauth2` 入口，但必须在 runtime 中明确配置 `grant_type: authorization_code`；参见 [one-cli 用户授权对接与本地验证指南](./oidc-cli-business-integration.md)。

## 目标

为下载并安装给 Agent 使用的 Go/Rust CLI 增加 `--auth oauth2`。首期只实现 OAuth 2.0 Client Credentials，并兼容三方文档中 Client Secret 位于 Basic、表单 Body 或 Query 的接口。

现有 `token`、`api_key`、`ak_sk` 和 `none` 的行为保持不变。

## CLI 契约

```text
--auth token|api_key|ak_sk|oauth2|none
```

- 未传 `--auth` 时先使用 `opencli.yaml` 的 `auth.type`，两者都为空时默认 `token`。
- `token`：调用方已经持有 Bearer Token。
- `oauth2`：具体流程由 runtime 的 `grant_type` 决定；本文只覆盖 `client_credentials`。
- `api_key`：直接注入 API Key。
- `ak_sk`：每次请求计算签名。
- `none`：不注入认证信息。

`--auth oauth2` 必须同时提供 `--runtime-config`。

## OpenAPI 映射

生成器保留：

- `components.securitySchemes`
- 文档级 `security`
- 操作级 `security`
- `oauth2.flows.clientCredentials.tokenUrl`
- Client Credentials scopes

操作级 `security` 使用三态语义：

- 未定义：继承文档级 `security`
- `security: []`：明确无认证
- 非空：使用声明的 Security Scheme

文档只有一个 OAuth 2.0 Client Credentials Scheme 时，生成器自动补齐 scheme、`grant_type: client_credentials`、token URL 和 scopes。Token 操作的 `client_id`、`client_secret` 都声明为 Query 参数时，自动选择 `placement: query`；否则默认使用标准的 `basic`。

多个 Client Credentials Scheme 并存时不猜测，运行配置必须显式给出完整 OAuth 元数据。

## Runtime Config

源配置：

```yaml
base_url: https://api.example.com
auth:
  type: oauth2
  client_id: public-client-id
```

生成配置：

```yaml
base_url: https://api.example.com
auth:
  type: oauth2
  grant_type: client_credentials
  scheme: vendorOAuth
  token_url: https://identity.example.com/oauth/token
  client_id: public-client-id
  client_auth:
    method: client_secret
    placement: basic
  encrypted_value: ENC[v1:...]
```

`client_id` 是公开标识，不加密。生成器从 `OPENCLI_OAUTH_CLIENT_SECRET` 读取 Client Secret，并复用现有 AES-256-GCM 密封机制写入 `encrypted_value`。环境变量是运行时的明文最高优先级覆盖。

OAuth 密封的 AAD 绑定 grant type、scheme、token URL、Client ID、Client Authentication method 和 placement，配置元数据被替换后密文无法通过认证。

该机制用于减少明文凭证误泄漏，不构成对生成包持有者的硬安全边界，因为解密材料随 CLI 一同交付。

## Token 请求

支持：

- `basic`：HTTP Basic Client Authentication，`grant_type` 和 scope 位于表单 Body。
- `body`：Client ID、Client Secret、`grant_type` 和 scope 位于表单 Body。
- `query`：全部参数位于 Query，用于兼容三方遗留接口。

Token 响应必须包含非空 `access_token`。非空 `token_type` 必须为 Bearer。Go 运行时在进程内缓存 Token，并在距离过期不足 60 秒时重新获取；Rust 首期每个 CLI 调用获取一次 Token。

Token 请求错误不得包含 Client Secret 或完整 Query URL。

## 请求注入

只有有效 Security Requirement 非空的操作才执行 OAuth：

```http
Authorization: Bearer <access_token>
```

显式 `Authorization` Header 优先，CLI 不覆盖它。启用 `--auth oauth2` 时，生成器移除对应的 Token Endpoint operation，不生成可能暴露 `--client-secret` 的公开命令；Token 请求只由内部 Token Provider 发起。

## 兼容性

- Runtime Config 当前只有一个配置格式，因此不设置无实际用途的顶层 schema version。
- Bearer、API Key 和 OAuth Client Secret 统一复用 `encrypted_value`。
- 不指定 `--auth oauth2` 时，不根据 OpenAPI 自动改变现有认证行为。
- Authorization Code、PKCE 和 OIDC 已由 `grant_type: authorization_code` 的运行时支持，但不属于本文范围。当前仍不支持 Device Code、OAuth Token Exchange Grant、mTLS 或 `private_key_jwt`。

## 验收

- 包含唯一 `clientCredentials` Security Scheme 的 OpenAPI 可自动识别 Scheme、Token URL 和 Client Authentication placement。
- 生成产物不包含明文 Client Secret。
- Client ID 保持明文。
- 受保护操作自动取得并注入 Bearer Token。
- Token Endpoint operation 不生成公开命令。
- Go/Rust 生成项目能够编译。
- 现有认证模式测试全部通过。
