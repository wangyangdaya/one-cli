# 本地 OIDC/OAuth2 联调服务设计

## 目标

在仓库根目录新增 `oauth2/` Python FastAPI 服务，用于验证 one-cli 生成客户端的
OIDC + OAuth 2.0 Authorization Code + PKCE 用户认证闭环。它与
`sql-mcp-server/` 同级，是独立的验证服务，不属于 one-cli Go module。

## 目录

```text
oauth2/
├── pyproject.toml
├── src/oauth2_server/
│   ├── app.py
│   ├── config.py
│   ├── models.py
│   ├── store.py
│   ├── tokens.py
│   ├── authorization.py
│   └── resources.py
├── templates/authorize.html
├── tests/
├── openapi.yaml
├── opencli.runtime.yaml
└── README.md
```

## 固定开发配置

```text
issuer:        http://127.0.0.1:18080
client_id:     one-cli-demo
API audience:  demo-api
callback path: /oauth/callback
```

只允许 loopback HTTP。Public Client 无 `client_secret`。

Scopes：

```text
openid
profile
expense:read:self
expense:submit:self
```

预置用户：

| username | password | subject | tenant |
| --- | --- | --- | --- |
| alice | alice123 | user-alice | company-a |
| bob | bob123 | user-bob | company-a |

## 接口

| 方法 | Path | 说明 |
| --- | --- | --- |
| GET | `/.well-known/openid-configuration` | Discovery |
| GET/POST | `/oauth/authorize` | 登录、授权与 code |
| POST | `/oauth/token` | code/refresh 换 Token |
| POST | `/oauth/revoke` | 撤销 |
| GET | `/oauth/jwks` | RS256 公钥 |
| GET | `/oauth/userinfo` | OIDC 用户信息 |
| GET | `/api/v1/me` | 业务主体 |
| GET/POST | `/api/v1/me/expenses` | 个人费用 |
| GET | `/healthz` | 健康检查 |

## 安全行为

- Authorization Code 绑定 client_id、redirect URI、scope、nonce 和 PKCE challenge。
- PKCE 只接受 S256；code 两分钟过期且一次性使用。
- redirect URI 只允许 `127.0.0.1`/`::1`、动态端口、固定 callback path。
- Access Token 为 RS256 JWT，`aud=demo-api`，有效期 15 分钟。
- ID Token 为 RS256 JWT，`aud=one-cli-demo`，包含 nonce。
- Refresh Token 为 opaque 随机值，有效期 8 小时，每次刷新轮换。
- 旧 Refresh Token 重用会撤销整个授权 session。
- 业务 API 校验签名、issuer、audience、expiry 和 scope。
- 用户、租户和费用 owner 从 Token 派生，不信任调用参数。
- 密码、code、verifier、Token 和 Authorization Header 不写日志。

RSA 密钥和全部状态只存在内存，服务重启后失效。该服务不提供数据库、SSO、MFA、
管理后台、生产 TLS、Client Credentials、Device Flow 或 Dynamic Client Registration。

## one-cli 对接

生成时使用：

```bash
opencli generate \
  --input ./oauth2/openapi.yaml \
  --output ./tmp/expense-cli \
  --app expense \
  --module example.com/expense-cli \
  --auth oidc \
  --runtime-config ./oauth2/opencli.runtime.yaml
```

`opencli.runtime.yaml` 中 `base_url` 指向业务 API，`auth.issuer` 指向本服务。
CLI 通过 Discovery 获取 authorize、token、revoke 和 JWKS 地址。配置不包含用户名、
密码、client_secret 或 Token。

## 测试

使用 pytest、FastAPI TestClient 和注入 clock 覆盖：

- Discovery、JWKS；
- PKCE 成功、缺失、plain 和 verifier 错误；
- 非 loopback redirect；
- code 过期与重放；
- 无 client_secret 交换；
- Access/ID Token claims；
- refresh rotation、reuse 和 revoke；
- audience、expiry、signature、scope 错误；
- Alice/Bob 数据隔离；
- 敏感值不出现在响应和日志。
