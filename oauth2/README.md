# 飞书 OAuth CLI 登录验证 Demo

本目录验证生成 CLI 的最小 OAuth 2.0 Authorization Code 登录流程。飞书负责用户授权，本地 broker 保管 App Secret，并代理 code exchange 和 refresh grant。

验证范围包括：

- `login`：浏览器完成飞书授权，CLI 接收并校验 `code` 和 `state`。
- `status`：离线报告本地 token 状态。
- 业务命令：自动携带 access token，并在到期前 5 分钟刷新。
- `logout`：删除本地 token 文件。

该 Demo 不实现账号、登录页、OIDC、JWT、PKCE、多用户或服务端撤销，不应直接部署到生产环境。

## 1. 飞书应用配置

在飞书开放平台创建或选择应用，然后完成以下配置：

1. 在安全设置中登记回调地址：`http://127.0.0.1:18081/oauth/callback`。
2. 开通“持续访问已授权的数据”，即 `offline_access`。
3. 发布应用，使回调地址和权限配置生效。
4. 记录 App ID 和 App Secret。

App ID 同时用于 runtime 和 broker，两处必须一致。App Secret 只配置在 broker 环境变量中，不能写入 runtime 或生成的 CLI。

## 2. Runtime 配置

复制示例文件：

```bash
# 在 one-cli 仓库根目录执行
cp ./oauth2/opencli.runtime.yaml /tmp/feishu-opencli.runtime.yaml
```

编辑 `/tmp/feishu-opencli.runtime.yaml`，将 `client_id` 替换为真实飞书 App ID：

```yaml
base_url: https://open.feishu.cn

auth:
  type: oauth2
  grant_type: authorization_code
  client_id: cli_xxx
  authorization_url: https://accounts.feishu.cn/open-apis/authen/v1/authorize
  token_url: http://127.0.0.1:18080/oauth/token
  redirect_uri: http://127.0.0.1:18081/oauth/callback
  scopes:
    - offline_access
```

字段说明：

| 字段 | 用途 |
| --- | --- |
| `base_url` | 生成 CLI 调用的飞书 OpenAPI 地址 |
| `client_id` | 飞书 App ID，必须与 broker 的 `FEISHU_APP_ID` 一致 |
| `authorization_url` | 浏览器打开的飞书授权页面 |
| `token_url` | 本地 broker 的 code/refresh 接口 |
| `redirect_uri` | CLI 本地回调地址，必须与飞书后台登记值完全一致 |
| `offline_access` | 要求飞书返回 refresh token |

`redirect_uri` 使用固定端口 `18081`。启动 `login` 前，需要确保该端口未被其他进程占用。

## 3. 启动 Token Broker

打开终端 A：

```bash
# 在 one-cli 仓库根目录执行
cd oauth2
uv sync

export FEISHU_APP_ID='cli_xxx'
export FEISHU_APP_SECRET='xxx'
export FEISHU_REDIRECT_URI='http://127.0.0.1:18081/oauth/callback'

uv run oauth2-server
```

broker 监听 `http://127.0.0.1:18080`，并将授权码和 refresh token 转发给飞书：

```text
POST https://open.feishu.cn/open-apis/authen/v2/oauth/token
```

在另一个终端检查服务：

```bash
curl http://127.0.0.1:18080/healthz
```

期望输出：

```json
{"status":"ok"}
```

## 4. 生成 CLI

打开终端 B，在 one-cli 仓库根目录执行：

```bash
go run ./cmd/opencli generate \
  --input ./oauth2/openapi.yaml \
  --output ./tmp/feishu-cli \
  --module example.com/feishu-cli \
  --app feishu \
  --auth oauth2 \
  --runtime-config /tmp/feishu-opencli.runtime.yaml
```

生成目录中的 `config/runtime.yaml` 已包含 runtime 配置。优先使用生成的 `bin/feishu` 启动器，它会自动设置 `OPENCLI_CONFIG`。

如果直接执行 `go run ./cmd/feishu`，必须手动设置：

```bash
export OPENCLI_CONFIG="$PWD/config/runtime.yaml"
```

## 5. 验证登录流程

进入生成目录：

```bash
cd ./tmp/feishu-cli
```

为了便于观察和清理，可将 token 文件固定到临时目录：

```bash
export OPENCLI_OAUTH_TOKEN_FILE=/tmp/feishu-oauth-token.json
```

依次执行：

```bash
./bin/feishu login
./bin/feishu status
./bin/feishu identity info
./bin/feishu logout
./bin/feishu status
```

`login` 会打印授权 URL 并尝试打开浏览器。无浏览器环境可执行：

```bash
./bin/feishu login --no-browser
```

手动打开输出的 URL 并完成授权。CLI 收到回调后，通过 broker 换取 access token 和 refresh token。

首次登录后的 `status` 应输出 `valid`。执行 `logout` 后再次运行 `status`，应输出 `not_logged_in`。

`status` 可能返回：

- `valid`：access token 剩余时间超过 5 分钟。
- `needs_refresh`：access token 即将过期，但 refresh token 仍有效。
- `expired`：refresh token 缺失或已过期，需要重新登录。
- `not_logged_in`：本地没有 token 文件。

业务命令遇到 `needs_refresh` 时，会通过 broker 自动刷新。飞书 refresh token 只能使用一次，因此 CLI 会原子保存新返回的 access/refresh token。

## 6. 常见问题

### 飞书提示回调地址错误

确认以下三处完全一致：

```text
飞书后台安全设置
auth.redirect_uri
FEISHU_REDIRECT_URI
```

统一值为：`http://127.0.0.1:18081/oauth/callback`。

### 登录成功但没有 refresh token

确认飞书应用已开通并发布 `offline_access`，且 runtime 的 `scopes` 中包含该权限。

### Broker 返回 client_id is not registered

确认 runtime 的 `auth.client_id` 与 broker 的 `FEISHU_APP_ID` 相同。

### CLI 无法启动回调监听

端口 `18081` 已被占用。停止占用端口的进程后重新执行 `login`。

### CLI 提示 missing OAuth runtime config

使用 `./bin/feishu` 启动器，或在生成目录中设置：

```bash
export OPENCLI_CONFIG="$PWD/config/runtime.yaml"
```

### 刷新失败并提示重新登录

refresh token 已失效、已被使用或被飞书拒绝。重新执行 `./bin/feishu login`。

## 7. 自动化测试

broker 测试使用 mock transport，不需要真实飞书凭证，也不会访问飞书：

```bash
cd oauth2
uv run pytest -q
uv run ruff check .
uv run ruff format --check .
```

仓库完整测试：

```bash
cd ..
make test
```
