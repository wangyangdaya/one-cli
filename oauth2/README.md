# one-cli 本地 OIDC/OAuth2 服务

用于开发和验证 one-cli 的 OIDC + OAuth 2.0 Authorization Code + PKCE
用户认证闭环。服务同时提供授权端点、JWT/JWKS、Refresh Token、撤销和个人费用 API。

> 仅限本机开发验证。服务使用 HTTP、内存状态、进程内 RSA 密钥和固定测试账号，
> 不可部署到生产环境。

完整的业务系统与 one-cli 对接要求参见
[`docs/design/oidc-cli-business-integration.md`](../docs/design/oidc-cli-business-integration.md)。

## 启动

```bash
cd oauth2
uv sync
uv run oauth2-server
```

默认监听 `http://127.0.0.1:18080`。检查：

```bash
curl http://127.0.0.1:18080/healthz
curl http://127.0.0.1:18080/.well-known/openid-configuration
```

测试账号：

| 用户名 | 密码 | 数据主体 |
| --- | --- | --- |
| `alice` | `alice123` | `user-alice` |
| `bob` | `bob123` | `user-bob` |

## one-cli 对接

OIDC 生成参数的目标契约：

```bash
go run ./cmd/opencli generate \
  --input ./oauth2/openapi.yaml \
  --output ./tmp/expense-cli \
  --module example.com/expense-cli \
  --app expense \
  --auth oidc \
  --runtime-config ./oauth2/opencli.runtime.yaml
```

生成 CLI 后：

```bash
expense auth login
expense auth status
expense auth check --operation expenses.list
expense expenses list
expense auth logout
```

`--auth oidc` 用于 OIDC 用户授权；现有 `--auth oauth2` 仍表示
Client Credentials 应用授权，两者不可混用。

授权服务地址填写在 `opencli.runtime.yaml`：

```yaml
base_url: http://127.0.0.1:18080
auth:
  type: oidc
  issuer: http://127.0.0.1:18080
  client_id: one-cli-demo
  audience: demo-api
```

CLI 根据 `issuer` 访问 `/.well-known/openid-configuration`，自动发现
authorize、token、revoke、JWKS 和 UserInfo。账号密码只在浏览器登录页输入；
配置中没有 `client_secret`、密码或 Token。

当前目录提供授权服务及对接契约。若当前 one-cli 分支尚未实现 `--auth oidc`，
需先完成仓库中的
`docs/superpowers/plans/2026-07-31-oidc-pkce-generated-cli.md`。

## 支持的协议行为

- Authorization Code + PKCE S256；
- 动态 loopback 端口与固定 `/oauth/callback`；
- OIDC Discovery、ID Token、UserInfo 和 JWKS；
- `aud=demo-api` 的 RS256 Access Token；
- Refresh Token 轮换与重用检测；
- Token 撤销；
- `expense:read:self`、`expense:submit:self`；
- Alice/Bob 个人数据服务端隔离。

所有 code、Token、授权 session 和 RSA 私钥在服务重启后失效。

## 测试

```bash
cd oauth2
uv run pytest -q
uv run ruff check .
uv run ruff format --check .
```
