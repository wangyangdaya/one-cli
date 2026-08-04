# 飞书 OAuth CLI 登录验证 Demo

本目录验证生成 CLI 的最小 OAuth 2.0 Authorization Code 登录流程。飞书负责用户授权，本地 broker 保管 App Secret，并代理 code exchange 和 refresh grant。

验证范围包括：

- `login`：浏览器完成飞书授权，CLI 接收并校验 `code` 和 `state`。
- `status`：离线报告本地 token 状态。
- 业务命令：自动携带 access token，并在到期前 5 分钟刷新。
- `logout`：删除本地 token 文件。

该飞书 Demo 本身不启用 OIDC、JWT、PKCE、多用户或服务端撤销，不应直接部署到生产环境。one-cli 生成器已经支持可选 PKCE S256 和 OIDC；是否启用由 runtime 配置决定，不会根据 scope 自动推断。

## 1. 飞书应用配置

在飞书开放平台创建或选择应用，然后完成以下配置：

1. 在安全设置中登记回调地址：`http://127.0.0.1:18081/oauth/callback`。
2. 开通“持续访问已授权的数据”，即 `offline_access`。
3. 发布应用，使回调地址和权限配置生效。
4. 记录 App ID 和 App Secret。

App ID 同时用于 runtime 和 broker，两处必须一致。App Secret 只配置在 broker 环境变量中，不能写入 runtime 或生成的 CLI。

## 2. Runtime 配置

直接编辑本目录中的 runtime 文件：

```bash
# 在 one-cli 仓库根目录执行
vim ./oauth2/opencli.runtime.yaml
```

将 `oauth2/opencli.runtime.yaml` 中的 `client_id` 替换为真实飞书 App ID：

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
| `redirect_uri` | 可选；配置后使用指定的固定回调地址。飞书要求该值与后台登记值完全一致 |
| `scopes` | 可选；配置后作为授权请求的 `scope` 参数。飞书使用 `offline_access` 获取 refresh token |

### 回调地址模式

生成的 CLI 同时支持两种回调方式：

- **固定回调地址**：配置 `redirect_uri`。适用于飞书等要求精确登记回调地址的授权服务器。本 Demo 固定使用 `http://127.0.0.1:18081/oauth/callback`，执行 `login` 前需确保端口 `18081` 未被占用。
- **随机端口回调**：省略 `redirect_uri`。CLI 会监听 `127.0.0.1` 的随机空闲端口，并生成 `http://127.0.0.1:<随机端口>/oauth/callback`，授权请求和 code 换 token 请求都会使用该实际地址。仅适用于允许原生应用 loopback 回调使用任意端口的授权服务器。

通用 OAuth 授权服务器如果不要求固定回调，也没有预先约定 scopes，可使用最小配置：

```yaml
base_url: https://api.example.com

auth:
  type: oauth2
  grant_type: authorization_code
  client_id: example-cli
  authorization_url: https://idp.example.com/oauth/authorize
  token_url: https://idp.example.com/oauth/token
```

省略 `scopes` 时，CLI 不会在授权 URL 中发送 `scope` 参数。如果授权服务器未返回 refresh token，登录和 access token 调用仍可使用，但 access token 到期后不能自动刷新，需要重新执行 `login`。

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
  --runtime-config ./oauth2/opencli.runtime.yaml
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

未设置 `OPENCLI_OAUTH_TOKEN_FILE` 时，CLI 默认保存在当前用户目录：

```text
$HOME/.opencli/oauth2/<配置哈希>/oauth-token.json
```

Windows 的 `$HOME` 对应 `%USERPROFILE%`。配置哈希由 `client_id`、`authorization_url` 和 `token_url` 计算，同名 CLI 连接不同 OAuth 应用时不会共用登录态。目录由 CLI 在首次登录成功时自动创建。

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
