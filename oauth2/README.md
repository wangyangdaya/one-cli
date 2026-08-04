# 飞书 OAuth CLI 登录验证 Demo

该目录用于验证生成 CLI 的最小登录生命周期：

- 浏览器打开飞书授权页；
- CLI 接收授权码并校验 `state`；
- 本地 broker 使用飞书 App Secret 换取 `access_token` 和
  `refresh_token`；
- CLI 以 `0600` 文件保存 token，在 access token 到期前 5 分钟自动刷新；
- `login`、`status`、`logout` 三个命令闭环。

飞书是唯一授权服务器。本服务不提供账号、登录页、OIDC、JWT 或 PKCE，且仅用于本机
开发验证，不应直接部署到生产环境。

## 1. 配置飞书应用

在飞书开放平台创建或选择应用：

1. 在安全设置中登记重定向 URL：
   `http://127.0.0.1:18081/oauth/callback`。
2. 开通“持续访问已授权的数据”权限，即 `offline_access`，并发布应用使配置生效。
3. 记录应用的 App ID 和 App Secret。

复制 runtime 示例并替换其中的 App ID：

```bash
cp ./oauth2/opencli.runtime.yaml /tmp/feishu-opencli.runtime.yaml
```

将 `/tmp/feishu-opencli.runtime.yaml` 中的
`cli_replace_with_feishu_app_id` 替换成真实 App ID。App Secret 不写入该文件。

## 2. 启动 token broker

```bash
cd oauth2
export FEISHU_APP_ID='cli_xxx'
export FEISHU_APP_SECRET='xxx'
export FEISHU_REDIRECT_URI='http://127.0.0.1:18081/oauth/callback'
uv sync
uv run oauth2-server
```

broker 监听 `http://127.0.0.1:18080`，并向飞书当前 token 接口转发 code 和
refresh grant：

`https://open.feishu.cn/open-apis/authen/v2/oauth/token`

健康检查：

```bash
curl http://127.0.0.1:18080/healthz
```

## 3. 生成并运行 CLI

在仓库根目录执行：

```bash
go run ./cmd/opencli generate \
  --input ./oauth2/openapi.yaml \
  --output ./tmp/feishu-cli \
  --module example.com/feishu-cli \
  --app feishu \
  --auth oauth2 \
  --runtime-config /tmp/feishu-opencli.runtime.yaml

cd ./tmp/feishu-cli
go run ./cmd/feishu login
go run ./cmd/feishu status
go run ./cmd/feishu identity get-user-info
go run ./cmd/feishu logout
```

`login` 会打印授权 URL 并自动打开浏览器。无法自动打开时，可使用
`login --no-browser`，然后手动打开输出的 URL。

`status` 不访问网络，只报告本地 token 状态：

- `valid`
- `needs_refresh`
- `expired`
- `not_logged_in`

业务命令在状态为 `needs_refresh` 时调用 broker 刷新。飞书 refresh token 只能使用
一次，因此 CLI 会以新返回的 access/refresh token 原子替换旧文件。刷新失败时需重新
执行 `feishu login`。

## 测试

自动化测试使用 mock transport，不需要真实飞书凭证，也不会访问飞书：

```bash
cd oauth2
uv run pytest -q
uv run ruff check .
uv run ruff format --check .
```
