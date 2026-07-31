# OpenCLI 用户操作手册

本文面向使用 OpenCLI 生成命令行工具的业务、交付、测试和实施人员，说明如何使用 `opencli` 从接口文档或 MCP 配置生成 CLI 项目，以及生成后需要重点核对哪些内容。

本文不介绍如何编译或构建 `opencli` 本身。使用前请确认你已经拿到可执行的 `opencli` 命令，或所在环境已经可以直接运行 `opencli`。

个人数据和当前用户授权的目标方案参见
[`docs/design/oidc-cli-business-integration.md`](./docs/design/oidc-cli-business-integration.md)。

## 1. OpenCLI 是什么

OpenCLI 是一个 CLI 项目生成工具。它读取接口描述文件，然后自动生成一个可继续交付、测试和集成的命令行项目。

OpenCLI 当前支持两类输入：

| 输入类型 | 说明 | 常见文件 |
| --- | --- | --- |
| OpenAPI/Swagger 文档 | 描述 HTTP API 的接口文档 | `openapi.yaml`、`swagger.json`、`api.yaml` |
| MCP 配置文件 | 描述 MCP 服务及工具发现方式 | `mcp.json` |

基本流程：

```text
接口文档或 MCP 配置 -> opencli generate -> 生成 CLI 项目 -> 编译生成可执行文件 -> 按业务确认和测试后交付
```

重要说明：

生成结果高度依赖输入文档的准确性。如果接口文档中的路径、请求方法、参数、请求体、认证方式、服务地址或字段必填规则不准确，生成出来的 CLI 命令、参数和调用行为也会随之不准确。

## 2. 使用前准备

使用 OpenCLI 前，请先准备以下材料。

| 材料 | 是否必需 | 说明 |
| --- | --- | --- |
| `opencli` 可执行命令 | 必需 | 本机或流水线环境中可直接运行 |
| OpenAPI/Swagger 文档 | 二选一 | 用于生成 HTTP API CLI |
| MCP 配置文件 | 二选一 | 用于生成 MCP 工具 CLI |
| `opencli.yaml` 配置文件 | 可选但推荐 | 用于调整命令命名、认证、请求体模式等 |
| 生成输出目录 | 必需 | 用于存放生成后的 CLI 项目 |
| 项目 module 名称 | 必需 | Go 目标项目的 module 路径 |
| CLI 应用名称 | 必需 | 生成后根命令名称，例如 `supplier`、`oa`、`petcli` |

检查 OpenCLI 是否可用：

```bash
opencli --help
```

### 2.1 收到 `dist` 文件夹后如何运行

如果你收到的是一个 `dist` 文件夹，里面通常已经放好了不同系统可直接运行的 OpenCLI 程序。使用人员不需要编译代码，只需要根据自己的电脑系统选择对应文件。

常见文件说明：

| 文件名 | 适用系统 | 如何运行 |
| --- | --- | --- |
| `opencli` | 当前构建机器对应的系统 | 在终端中执行 `./opencli --help` |
| `opencli_darwin_arm64` | macOS，Apple Silicon 芯片，例如 M1/M2/M3/M4 | 在终端中执行 `./opencli_darwin_arm64 --help` |
| `opencli_darwin_amd64` | macOS，Intel 芯片 | 在终端中执行 `./opencli_darwin_amd64 --help` |
| `opencli_linux_amd64` | Linux，x86_64 服务器或电脑 | 在终端中执行 `./opencli_linux_amd64 --help` |
| `opencli_windows_amd64.exe` | Windows，x86_64 电脑 | 在 PowerShell 中执行 `.\opencli_windows_amd64.exe --help` |

Windows 使用方式：

1. 解压或打开 `dist` 文件夹。
2. 在文件夹空白处按住 `Shift`，点击鼠标右键，选择“在终端中打开”或“在 PowerShell 中打开”。
3. 执行：

```powershell
.\opencli_windows_amd64.exe --help
```

macOS 使用方式：

1. 打开“终端”。
2. 进入 `dist` 文件夹，例如：

```bash
cd ~/Downloads/dist
```

3. 根据芯片选择 `opencli_darwin_arm64`（Apple Silicon）或 `opencli_darwin_amd64`（Intel）。第一次运行前，如果系统提示没有执行权限，先执行，例如：

```bash
chmod +x ./opencli_darwin_arm64
```

4. 查看帮助：

```bash
./opencli_darwin_arm64 --help
```

Linux 使用方式：

```bash
cd ./dist
chmod +x ./opencli_linux_amd64
./opencli_linux_amd64 --help
```

看到帮助信息后，说明程序可以正常运行。后续命令只需要把文档中的 `opencli` 替换成你实际使用的文件名即可，例如：

```bash
./opencli_darwin_arm64 inspect --input ./api.yaml
./opencli_darwin_arm64 generate --input ./api.yaml --output ./my-cli --module github.com/example/my-cli --app mycli
```

Windows 示例：

```powershell
.\opencli_windows_amd64.exe inspect --input .\api.yaml
.\opencli_windows_amd64.exe generate --input .\api.yaml --output .\my-cli --module github.com/example/my-cli --app mycli
```

注意：

- 这个程序是命令行工具，通常不是双击打开使用，而是在终端或 PowerShell 中输入命令执行。
- 如果命令提示“找不到文件”，请先确认当前终端所在目录就是 `dist` 文件夹。
- 如果 macOS 提示无法打开来自未知开发者的程序，可以在“系统设置 > 隐私与安全性”中允许打开，或联系交付人员提供已签名版本。

查看某个子命令帮助：

```bash
opencli generate --help
opencli inspect --help
```

## 3. 检查 OpenAPI 文档

如果输入是 OpenAPI/Swagger 文档，建议先使用 `inspect` 查看文档中识别到的接口列表。

```bash
opencli inspect --input ./api.yaml
```

也可以使用远程 URL：

```bash
opencli inspect --input https://example.com/openapi.json
```

输出示例：

```text
users GET /users listUsers
users POST /users createUser
users GET /users/{userId} getUser
```

每一列含义如下：

| 列 | 含义 | 会影响生成结果 |
| --- | --- | --- |
| 第 1 列 | `tag`，命令组来源 | 生成一级命令，例如 `users` |
| 第 2 列 | HTTP 方法 | 影响请求方式和风险判断 |
| 第 3 列 | API 路径 | 影响实际请求地址和 path 参数 |
| 第 4 列 | `operationId` | 生成子命令名称的主要来源 |

如果这里看到的接口数量、分组、路径或 `operationId` 与预期不一致，应先修正接口文档或补充 `opencli.yaml`，再生成 CLI 项目。

需要 JSON 输出时可加 `--json`：

```bash
opencli --json inspect --input ./api.yaml
```

如果接口文档来自内部平台或第三方系统，tag、path、operationId 可能不适合直接生成 CLI。此时可以使用 AI 建议模式生成一份可审阅的 `opencli.yaml` 命名配置草稿：

```bash
OPENCLI_AI_BASE_URL=https://your-openai-compatible-host \
OPENCLI_AI_API_KEY=your-api-key \
OPENCLI_AI_MODEL=your-model \
opencli inspect \
  --input ./api.yaml \
  --ai-suggest-config \
  --output ./opencli.ai.yaml
```

然后人工确认 `opencli.ai.yaml` 后用于生成：

```bash
opencli generate \
  --input ./api.yaml \
  --config ./opencli.ai.yaml \
  --output ./my-cli \
  --module github.com/example/my-cli \
  --app mycli
```

AI 建议模式不会修改原始 OpenAPI 文档，也不会自动执行生成。它只产生命名建议，并会在本地校验未知接口、非法命令名和重复 alias。

完整说明见根目录下的 [`INSPECT.md`](./INSPECT.md)。

## 4. 生成 CLI 项目

### 4.1 从 OpenAPI/Swagger 生成

```bash
opencli generate \
  --input ./api.yaml \
  --output ./my-cli \
  --module github.com/example/my-cli \
  --app mycli
```

常用参数说明：

| 参数 | 是否必需 | 说明 |
| --- | --- | --- |
| `--input` | 与 `--mcp-config` 二选一 | OpenAPI/Swagger 文档路径或 URL |
| `--output` | 必需 | 生成项目输出目录 |
| `--module` | 必需 | 生成项目的 Go module 路径 |
| `--app` | 必需 | 生成 CLI 的应用名和根命令名 |
| `--config` | 可选 | `opencli.yaml` 配置文件路径 |
| `--target` | 可选 | 生成目标，支持 `go` 或 `rust`，默认 `go` |
| `--skill-lang` | 可选 | 生成技能文档语言，支持 `en` 或 `zh`，默认 `en` |
| `--app-version` | 可选 | 生成 CLI 的版本号 |
| `--auth` | 可选 | 认证模式：`token`、`api_key`、`ak_sk`、`oauth2` 或 `none`；默认 `token` |
| `--signer` | 可选 | AK/SK 签名配置，例如 `supplier_edi` |
| `--runtime-config` | `api_key`、`oauth2` 必需 | 生成后 CLI 的 Base URL 和认证元数据；需要时从环境变量读取凭证并密封 |

`--auth` 的取值说明：

| 值 | 凭证来源 | 生成后的行为 |
| --- | --- | --- |
| `token` | 已有 Bearer Token | 直接注入 `Authorization: Bearer <token>` |
| `api_key` | API Key | 注入 Runtime Config 声明的请求头 |
| `ak_sk` | Access Key / Secret Key | 按 signer 配置为每个请求计算签名 |
| `oauth2` | OAuth Client ID / Client Secret | 通过 Client Credentials 获取 Access Token，再注入 Bearer Token |
| `none` | 无 | 不注入认证信息 |

未传 `--auth` 时，OpenCLI 先读取 `opencli.yaml` 的 `auth.type`；两者都没有配置时默认使用 `token`。无需认证的接口应明确使用 `--auth none`。

当前版本尚不支持 `--auth oidc`。OIDC + Authorization Code + PKCE 是用户级授权的目标模式，不能用现有 `--auth oauth2` 代替。

带配置文件生成：

```bash
opencli generate \
  --input ./api.yaml \
  --output ./my-cli \
  --module github.com/example/my-cli \
  --app mycli \
  --config ./opencli.yaml \
  --skill-lang zh
```

### 4.2 从 MCP 配置生成

```bash
opencli generate \
  --mcp-config ./mcp.json \
  --output ./my-mcp-cli \
  --module github.com/example/my-mcp-cli \
  --app mymcp
```

注意：

- `--input` 和 `--mcp-config` 必须二选一，不能同时传，也不能都不传。
- 从 MCP 配置生成时，OpenCLI 会在生成阶段连接 MCP 服务并发现工具列表。
- MCP 服务配置中的地址、命令、参数、环境变量和请求头需要根据实际环境确认。

### 4.3 生成 Rust 目标项目

```bash
opencli generate \
  --input ./api.yaml \
  --output ./my-cli-rust \
  --module github.com/example/my-cli \
  --app mycli \
  --target rust
```

### 4.4 编译生成后的 CLI

`opencli generate` 生成的是 CLI 项目源码。要交付给最终用户使用，还需要在生成目录中编译成对应平台的可执行文件。

Go 目标项目：

```bash
cd ./my-cli

# 当前系统调试或本机交付
go build -o bin/mycli ./cmd/mycli

# macOS Apple Silicon
GOOS=darwin GOARCH=arm64 go build -o dist/darwin-arm64/mycli ./cmd/mycli

# macOS Intel
GOOS=darwin GOARCH=amd64 go build -o dist/darwin-amd64/mycli ./cmd/mycli

# Linux x86_64
GOOS=linux GOARCH=amd64 go build -o dist/linux-amd64/mycli ./cmd/mycli

# Windows x86_64
GOOS=windows GOARCH=amd64 go build -o dist/windows-amd64/mycli.exe ./cmd/mycli
```

Go 交叉编译说明：

- 在 Mac 上可以直接用 `GOOS=windows GOARCH=amd64` 生成 Windows `.exe`。
- 如果生成项目没有使用 CGO，通常不需要额外安装 Windows 编译器或链接器。
- 输出文件名建议显式加 `.exe`，方便 Windows 用户直接运行。

Rust 目标项目：

```bash
cd ./my-cli-rust

# 当前系统调试构建
cargo build

# 当前系统发布构建
cargo build --release

# 如需交叉编译，先安装目标平台，再构建
rustup target add aarch64-apple-darwin x86_64-unknown-linux-gnu x86_64-pc-windows-msvc
cargo build --release --target aarch64-apple-darwin
cargo build --release --target x86_64-unknown-linux-gnu
cargo build --release --target x86_64-pc-windows-msvc
```

Rust 交叉编译说明：

- `cargo check --target <target>` 只做检查，不生成最终可执行文件；交付产物需要使用 `cargo build --release --target <target>`。
- `x86_64-pc-windows-msvc` 依赖 Windows/MSVC 的 `link.exe`。在 Mac 上即使安装了 Rust target，也通常不能直接完成链接。
- 如果必须生成 MSVC 版本的 Windows `.exe`，建议在 Windows 机器或 CI 的 Windows runner 上构建。
- 如果要在 Mac 上本地生成 Windows `.exe`，可以改用 `x86_64-pc-windows-gnu`，并安装 MinGW 工具链：

```bash
rustup target add x86_64-pc-windows-gnu
brew install mingw-w64
cargo build --release --target x86_64-pc-windows-gnu
```

生成文件通常位于：

```text
target/x86_64-pc-windows-gnu/release/mycli.exe
```

说明：

- 在 Mac 上直接构建，默认得到 Mac 可执行文件。
- 在 Windows 上直接构建，默认得到 Windows `.exe` 可执行文件。
- 如果要在一台机器上生成多个系统的产物，需要使用 Go 的 `GOOS`/`GOARCH` 或 Rust 的 target 机制。

## 5. `opencli.yaml` 配置文件

`opencli.yaml` 用于在不修改接口文档的情况下调整生成结果。建议每个正式项目都维护一份配置文件，并由业务、接口提供方和交付方共同确认。

示例：

```yaml
app:
  binary: mycli
  root_command: mycli
  version: "1.0.0"

auth:
  type: token

naming:
  tag_alias:
    user-management: users
  operation_alias:
    listUsers: list
    createUser: create

runtime:
  auth_header: Authorization
  default_output: json

overrides:
  body_mode:
    users.create: flags
    users.import: file-or-data
  body_fields:
    users.create:
      - name: name
        description: 用户名称
        required: true
        type: string
```

配置项说明：

| 配置项 | 说明 | 需要谁确认 |
| --- | --- | --- |
| `app.binary` | 生成的二进制名称 | 交付方、运维方 |
| `app.root_command` | CLI 根命令名称 | 使用方、交付方 |
| `app.version` | 生成 CLI 版本号 | 产品或发布负责人 |
| `auth.type` | 认证类型，支持 `token`、`api_key`、`ak_sk`、`oauth2`、`none` | 接口提供方、安全负责人 |
| `auth.signer` | AK/SK 签名规则 | 接口提供方、安全负责人 |
| `naming.tag_alias` | 接口 tag 到命令组的命名映射 | 业务使用方 |
| `naming.operation_alias` | `operationId` 到子命令的命名映射 | 业务使用方 |
| `runtime.auth_header` | Token 认证头名称 | 接口提供方 |
| `runtime.default_output` | 默认输出格式 | 使用方 |
| `overrides.body_mode` | 请求体使用 flags、文件或 JSON 数据 | 业务使用方、测试方 |
| `overrides.body_fields` | 请求体字段说明和必填规则补充 | 接口提供方、业务使用方 |

### 5.1 Runtime Config 与 OAuth 2.0

`opencli.yaml` 控制生成行为；`--runtime-config` 指向的 `runtime.yaml` 控制生成后 CLI 使用的 Base URL 和认证元数据。Runtime Config 当前只有一种格式，不需要顶层 `version` 字段。

Runtime Config 是可选的。`token` 模式可以完全没有 `runtime.yaml`，此时生成阶段不需要设置 token：

```bash
opencli generate \
  --target rust \
  --input ./examples/leave-makeup.yaml \
  --output ./tmp/attendance-cli-new \
  --module attendance-cli \
  --app attendance-cli \
  --skill-lang zh \
  --auth token \
  --app-version 0.0.1
```

生成完成后，在实际运行 CLI 时提供环境变量：

```bash
export OPENCLI_BASE_URL='https://api.example.com'
export OPENCLI_AUTH_TOKEN='your-token'
attendance-cli <group> <command>
```

因此，只有传入包含 `auth.type: bearer` 的 `--runtime-config`、需要在生成阶段密封 token 时，生成器才要求 `OPENCLI_AUTH_TOKEN` 已存在。

如果接口不需要认证，只需要为生成后的 CLI 固定 Base URL，可以只配置：

```yaml
base_url: https://api.example.com
```

此时生成命令必须显式传入 `--auth none`。如果接口需要 token 认证，则保留默认的 `token` 模式；仅包含 `base_url` 的 Runtime Config 也可以正常生成，token 在 CLI 运行时从 `OPENCLI_AUTH_TOKEN` 读取。

Bearer Token 示例：

```yaml
base_url: https://api.example.com
auth:
  type: bearer
```

Bearer Token 的明文不写入 YAML，也不需要通过生成命令参数传入。生成器会自动从当前进程环境读取 `OPENCLI_AUTH_TOKEN`；token 应只包含原始值，不要带 `Bearer ` 前缀。如果当前终端已经执行过 `export OPENCLI_AUTH_TOKEN='your-token'`，后续 `opencli generate` 命令无需再次指定 token。

使用 Bearer Runtime Config 时，如果出现 `OPENCLI_AUTH_TOKEN is required to seal runtime auth`，表示启动生成器的进程没有继承该变量。请在同一个终端导出变量后重试，也可以用 `printenv OPENCLI_AUTH_TOKEN` 检查；不要把真实 token 写入 Runtime Config 或提交到仓库。

如果只需要在生成项目中固定 Base URL，而 token 要到 CLI 运行时再提供，Runtime Config 可以只写 `base_url` 并保持 `--auth token`。这种组合在生成阶段不读取 token，运行 CLI 时设置 `OPENCLI_AUTH_TOKEN` 即可。

如果 Base URL 和 token 都准备在生成后的 CLI 运行时通过环境变量提供，则不传 `--runtime-config`，运行 CLI 时同时设置 `OPENCLI_BASE_URL` 和 `OPENCLI_AUTH_TOKEN`。

API Key 示例：

```yaml
base_url: https://api.example.com
auth:
  type: api_key
  header: X-API-Key
```

OAuth 2.0 Client Credentials 示例：

```yaml
base_url: https://api.example.com
auth:
  type: oauth2
  client_id: my-service-cli
```

其中：

- `client_id` 是公开客户端标识，不加密。
- `client_secret` 不写入 YAML，也不通过命令行参数传入。
- 生成时从 `OPENCLI_OAUTH_CLIENT_SECRET` 读取 Client Secret。
- `--auth oauth2` 当前只代表 Client Credentials；生成器从 OpenAPI 唯一的 `clientCredentials` Security Scheme 补齐 `grant_type: client_credentials`、scheme、Token URL 和 scopes。
- 当前不提供单独的 Grant Type CLI 参数，因为除 `client_credentials` 外的值都不受支持；生成时会拒绝其他值。
- 如果 Token operation 把 `client_id`、`client_secret` 都声明为 Query 参数，生成器会识别为 `placement: query`；否则默认使用标准的 HTTP Basic 方式。

生成示例：

```bash
export OPENCLI_OAUTH_CLIENT_SECRET='your-client-secret'

opencli generate \
  --input ./api.yaml \
  --output ./my-cli \
  --module github.com/example/my-cli \
  --app mycli \
  --auth oauth2 \
  --runtime-config ./runtime.yaml
```

生成后的 `config/runtime.yaml` 类似：

```yaml
base_url: https://api.example.com
auth:
  type: oauth2
  grant_type: client_credentials
  scheme: serviceOAuth
  token_url: https://identity.example.com/oauth/token
  client_id: my-service-cli
  client_auth:
    method: client_secret
    placement: query
  encrypted_value: ENC[v1:...]
```

`ENC[v1:...]` 中的 `v1` 是加密信封格式版本，不是 Runtime Config 版本。密封机制用于避免明文凭证误泄漏；解密材料随 CLI 交付，因此不构成对安装包持有者的强安全边界。

运行时流程：

```text
Agent 调用业务命令
  -> CLI 解密 Client Secret
  -> 调用三方 Token Endpoint
  -> 获得 Access Token
  -> 注入 Authorization: Bearer <access_token>
  -> 调用三方业务接口
```

启用 `--auth oauth2` 后，对应的 Token Endpoint operation 不生成公开命令，避免产生要求用户输入 `--client-secret` 的重复入口。

### 5.2 OIDC 用户授权目标契约

本节描述后续版本的目标接口，不代表当前生成器已经支持。个人数据接口不应使用 Client Credentials 冒充用户。

目标生成命令：

```bash
opencli generate \
  --input ./business.openapi.yaml \
  --output ./business-cli \
  --module example.com/business-cli \
  --app business \
  --auth oidc \
  --runtime-config ./business.runtime.yaml
```

业务 API 和授权服务地址填写在 Runtime Config：

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

- `base_url` 是业务 API 地址。
- `issuer` 是授权服务地址。
- CLI 通过 `<issuer>/.well-known/openid-configuration` 发现 authorize、token、revoke、JWKS 和 UserInfo。
- `client_id` 是公开的 Public Client 标识。
- 配置中不得出现用户名、密码、`client_secret`、Access Token 或 Refresh Token。

目标登录命令：

```bash
business auth login
business auth status
business auth check --operation <operation>
business identity current
business auth logout
```

`auth login` 才会启动随机 loopback 端口并打开系统浏览器。业务命令未登录时返回 `login_required`，不在后台自动登录。

仓库根目录的 `oauth2/` FastAPI 服务已经可以验证 Discovery、PKCE、JWT/JWKS、Refresh、Revoke 和个人数据隔离：

```bash
cd oauth2
uv sync
uv run oauth2-server
```

完整对接和安全规范参见
[`docs/design/oidc-cli-business-integration.md`](./docs/design/oidc-cli-business-integration.md)。

请求体模式说明：

| 模式 | 适用场景 | 生成后使用方式 |
| --- | --- | --- |
| `flags` | 请求体字段少、字段结构简单 | `mycli users create --name Alice --email a@example.com` |
| `file-or-data` | 请求体复杂、嵌套对象、数组、大段 JSON | `mycli users import --file users.json` 或 `--data '{...}'` |
| `simple-json` | OpenCLI 根据简单 JSON schema 自动识别 | 通常会展开为较易使用的参数 |

`body_mode` 匹配优先级：

| 优先级 | 写法 | 示例 |
| --- | --- | --- |
| 1 | `命令组.子命令` | `users.create` |
| 2 | `原始 tag.子命令` | `user-management.create` |
| 3 | `子命令` | `create` |
| 4 | `operationId` | `createUser` |
| 5 | `method path` | `post /users` |
| 6 | `path` | `/users` |

## 6. 生成结果说明

生成目录通常包含以下内容：

```text
my-cli/
├── bin/                    # 本地调试启动脚本
├── cmd/                    # 生成 CLI 的入口
├── dist/                   # 编译后按平台存放可执行文件（需执行第 4.4 节的构建命令）
│   ├── darwin-arm64/       # macOS Apple Silicon
│   ├── darwin-amd64/       # macOS Intel
│   ├── linux-amd64/        # Linux x86_64
│   └── windows-amd64/      # Windows x86_64
├── internal/               # 命令、接口调用、配置、输出等实现
├── skills/                 # Skill 索引和按命令组生成的技能文档；多分组时含 agent loader 入口
├── README.md               # 生成项目说明
├── go.mod                  # Go 目标项目依赖声明
└── ...
```

使用人员通常重点关注：

| 文件或目录 | 用途 | 是否建议人工确认 |
| --- | --- | --- |
| `bin/<app>` | macOS、Linux、Git Bash 等 shell 环境的本地启动脚本 | 了解用途即可 |
| `bin/<app>.cmd` | Windows CMD 或 PowerShell 的本地启动脚本 | 了解用途即可 |
| `dist/<platform>/<app>` | 按平台编译的最终可执行文件；Windows 文件名为 `<app>.exe` | 是，交付前确认平台与芯片架构 |
| `README.md` | 生成项目使用说明 | 是 |
| `skills/SKILL.md` | 多分组时生成；Agent loader 入口和命令组路由索引 | 是 |
| `skills/README.md` | 生成 Skill 的轻量索引 | 是 |
| `skills/<group>/SKILL.md` | 每个命令组的操作说明和示例 | 是 |
| `internal/<group>/command.go` | 命令、参数、帮助信息 | 必要时由开发或交付人员确认 |
| `internal/<group>/service*.go` | 实际请求实现 | 认证、签名、地址异常时确认 |
| `opencli.yaml` | 生成配置源文件 | 是，建议纳入交付物 |

不建议直接修改大量生成代码来修正文档问题。优先修正 OpenAPI/Swagger 文档或 `opencli.yaml` 后重新生成，这样结果更稳定，也便于后续接口变更时再次生成。

### 6.1 `bin` 目录里的文件如何使用

Go 目标项目生成后，`bin` 目录中通常会有两个启动脚本。它们不是最终编译出来的可执行文件，而是用于本地调试的快捷入口，会在内部执行 `go run ./cmd/<app>`。

假设生成时使用 `--app mycli`，常见文件如下：

| 文件 | 适用平台 | 运行示例 |
| --- | --- | --- |
| `bin/mycli` | macOS、Linux、Git Bash、WSL | `./bin/mycli --help` |
| `bin/mycli.cmd` | Windows CMD、PowerShell | `.\bin\mycli.cmd --help` |

Windows 示例：

```powershell
cd .\my-cli
.\bin\mycli.cmd --help
.\bin\mycli.cmd users list
```

macOS 或 Linux 示例：

```bash
cd ./my-cli
./bin/mycli --help
./bin/mycli users list
```

说明：

- 启动脚本依赖本机已经安装 Go，并且能正常执行 `go run`。
- `bin/mycli.cmd` 适合 Windows 终端使用；`bin/mycli` 适合 macOS、Linux 或类 Unix shell 使用。
- 如果要交付给没有 Go 环境的最终用户，应先按第 4.4 节编译成真正的可执行文件，例如 Windows 下的 `mycli.exe`。
- Windows 平台最终交付文件建议使用 `.exe`，例如 `go build -o dist/windows-amd64/mycli.exe ./cmd/mycli`。

## 7. 使用生成后的 CLI

生成后的 CLI 根命令由 `--app` 或 `opencli.yaml` 中的应用配置决定。下面用 `mycli` 举例。

查看帮助：

```bash
mycli --help
mycli users --help
mycli users create --help
```

执行查询类接口：

```bash
mycli users list
mycli users list --page 1 --limit 20
mycli users get --user-id 10001
```

执行带请求体的接口：

```bash
mycli users create --name Alice --email alice@example.com
mycli users import --file ./users.json
mycli users import --data '{"users":[{"name":"Alice"}]}'
```

传递额外请求头：

```bash
mycli -H "X-Tenant-ID: tenant-a" users list
mycli users list --header "X-Tenant-ID: tenant-a"
mycli users list --header "X-Trace-ID=trace-001"
```

输出 JSON：

```bash
mycli --json users list
```

打印请求和响应追踪日志：

```bash
mycli --trace users list
```

查看生成的 Skill 文档：

```bash
mycli skills list
mycli skills read users
mycli skills read users references/command-routing.md
mycli skills --skills-dir /path/to/skills read users
```

说明：

- `skills list` 会列出磁盘 `skills/` 目录中的 Skill。
- `skills read <skill>` 默认打印该 Skill 的 `SKILL.md`。
- `skills read <skill> <path>` 可以读取该 Skill 下的 reference 文件，例如 `references/command-routing.md`。
- 该命令默认读取当前工作目录的 `./skills`；如果二进制安装在其他位置或不在生成项目根目录运行，传 `--skills-dir /path/to/skills`。这样业务调整后的 `SKILL.md` 会立即对 AI agent 生效。

认证相关环境变量：

| 认证模式 | 常见环境变量 | 说明 |
| --- | --- | --- |
| `token` | `OPENCLI_AUTH_TOKEN` | 生成 CLI 会按配置的认证头发送 token |
| `api_key` | `OPENCLI_API_KEY` | 使用 Runtime Config 声明的请求头发送 API Key |
| `ak_sk` | `OPENCLI_AK`、`OPENCLI_SK` | 生成 CLI 会按签名配置生成 AK/SK 请求头 |
| `oauth2` | `OPENCLI_OAUTH_CLIENT_SECRET` | CLI 自动执行 Client Credentials；环境变量可覆盖密封的 Client Secret |
| `none` | 无 | 不注入认证信息 |

目标 `oidc` 模式不使用 Secret 环境变量。Access/Refresh Token 应由登录流程写入 OS Keychain，不能通过命令行或普通 YAML 提供。

示例：

```bash
export OPENCLI_AUTH_TOKEN="your-token"
mycli users list
```

AK/SK 示例：

```bash
export OPENCLI_AK="your-access-key"
export OPENCLI_SK="your-secret-key"
mycli orders list
```

OAuth2 模式正常情况下不需要 Agent 手工调用 Token 接口：

```bash
mycli resources list
```

CLI 会自动获取 Access Token。需要临时替换 Client Secret 时：

```bash
OPENCLI_OAUTH_CLIENT_SECRET="temporary-secret" mycli resources list
```

## 8. 接口文档质量对生成结果的影响

OpenCLI 不会凭空知道真实业务规则，它主要根据接口文档生成命令。因此以下内容必须准确。

| 接口文档内容 | 如果不准确会导致 |
| --- | --- |
| `servers.url` 或服务地址 | 生成 CLI 请求到错误环境或错误网关 |
| `paths` | 命令调用错误接口 |
| HTTP 方法 | GET/POST/PUT/DELETE 行为错误 |
| `tags` | 命令分组混乱 |
| `operationId` | 子命令名称重复、难懂或不稳定 |
| `summary`、`description` | 帮助信息和技能文档不清楚 |
| path/query/header 参数 | 生成参数缺失、名称错误或必填规则错误 |
| `requestBody` schema | `--data`、`--file` 或 flags 的生成方式不符合实际 |
| `required` 字段 | 必填校验不准确 |
| 字段类型 | CLI 参数类型、示例值和 JSON 结构不准确 |
| 认证头、Security Scheme、OAuth Flow | Bearer、API Key、AK/SK 或 OAuth2 注入方式不正确 |
| 响应说明和错误码 | 使用文档和排错信息不完整 |

如果生成后的 CLI 和实际接口行为不一致，优先检查接口文档是否真实反映了接口实现。

## 9. 需要根据实际业务修改或确认的文档

正式交付前，建议确认以下文档和配置。

| 文档或配置 | 必须确认的内容 | 责任方 |
| --- | --- | --- |
| OpenAPI/Swagger 文档 | 接口路径、方法、参数、请求体、必填字段、认证方式、服务地址 | 接口提供方 |
| `opencli.yaml` | 命令命名、认证模式、签名规则、请求体模式、字段补充说明 | 业务使用方、交付方 |
| `runtime.yaml` | Base URL、认证类型、公开 Client ID、Token URL 和 Client Authentication 位置 | 接口提供方、安全负责人 |
| MCP 配置文件 | 服务名、transport、URL、command、args、headers、env | MCP 服务提供方、实施方 |
| 生成项目 `README.md` | 安装方式、运行方式、环境变量、示例命令是否符合交付场景 | 交付方、使用方 |
| `skills/SKILL.md` | 多分组时确认：Agent 应用添加 `skills/` 文件夹时能否识别，以及路由说明是否清晰 | 交付方、使用方 |
| `skills/README.md` | Skill 分组索引是否便于定位目标命令组 | 交付方、使用方 |
| `skills/<group>/SKILL.md` | 命令说明、风险提示、示例参数、业务术语是否准确 | 业务使用方、测试方 |
| 示例 JSON 文件 | 请求体字段、枚举值、日期格式、金额单位、组织或租户字段 | 业务使用方、接口提供方 |
| 环境变量说明 | Bearer Token、API Key、OAuth Client Secret、AK/SK、租户、网关和代理等配置是否完整 | 运维方、安全负责人 |
| 测试用例或验收清单 | 关键查询、创建、更新、删除接口是否覆盖 | 测试方、业务验收方 |

特别需要人工确认的业务项：

- 命令名称是否符合使用人员习惯。
- 同一个 tag 下的接口是否真的属于同一类业务。
- 写入、删除、审批、提交等高风险操作是否有清晰提示。
- 请求体中的枚举、状态码、业务类型是否为最新值。
- 日期、时间、金额、数量、币种、时区等字段格式是否明确。
- 租户、组织、门店、供应商、用户身份等上下文字段是否需要固定传入。
- 生产环境和测试环境的服务地址是否区分清楚。
- 认证方式是否与实际网关一致。

## 10. 推荐操作流程

建议按以下顺序使用 OpenCLI：

1. 获取最新接口文档或 MCP 配置。
2. 使用 `opencli inspect --input ./api.yaml` 检查接口识别结果。
3. 与业务和接口提供方确认 tag、operationId、参数、请求体和认证方式。
4. 编写或更新 `opencli.yaml`。
5. 使用 `opencli generate` 生成 CLI 项目。
6. 进入生成目录，使用 `go build` 或 `cargo build` 编译生成可执行文件。
7. 查看生成项目的 `README.md`、`skills/README.md` 和 `skills/<group>/SKILL.md`；多分组项目还要查看 `skills/SKILL.md`。
8. 使用测试环境执行关键命令，覆盖查询、写入、更新、删除等场景。
9. 根据测试结果修正接口文档或 `opencli.yaml`，然后重新生成并重新编译。
10. 形成交付版本，并附上确认后的使用说明、环境变量和示例命令。

## 11. 常见问题

### 生成命令组名称不好理解怎么办？

优先在 `opencli.yaml` 中使用 `naming.tag_alias` 调整。例如：

```yaml
naming:
  tag_alias:
    user-management: users
```

### 子命令名称太长怎么办？

使用 `naming.operation_alias` 调整：

```yaml
naming:
  operation_alias:
    listUsersUsingGET: list
```

### 请求体没有展开成想要的参数怎么办？

在 `overrides.body_mode` 中明确指定：

```yaml
overrides:
  body_mode:
    users.create: flags
    users.import: file-or-data
```

如果字段说明不完整，可同时补充 `overrides.body_fields`。

### 生成后接口调用失败怎么办？

优先检查：

- 接口文档中的服务地址是否正确。
- 当前环境变量中的 Bearer Token、API Key、OAuth Client Secret 或 AK/SK 是否正确。
- OAuth2 模式下的 Client ID、Token URL、`grant_type` 和 Client Authentication placement 是否符合三方文档。
- 是否需要额外传入租户、组织、供应商等请求头。
- path、query、header 参数是否和实际接口一致。
- 请求体字段是否缺失、类型错误或枚举值不正确。

### 是否可以直接改生成出来的代码？

可以，但不推荐作为首选。接口文档或命名配置导致的问题，应优先修改 OpenAPI/Swagger 文档或 `opencli.yaml` 后重新生成。只有认证签名、特殊网关规则、复杂业务封装等无法通过配置表达时，才建议由开发人员修改生成代码。

### `opencli init` 可以生成配置文件吗？

当前 `opencli init` 还未实现完整初始化能力。建议手工维护 `opencli.yaml`，或从已有项目的配置文件复制后按业务调整。

## 12. 交付检查清单

交付前建议逐项确认：

- 已使用最新接口文档或 MCP 配置生成。
- `opencli inspect` 的接口列表符合预期。
- `opencli.yaml` 已经按实际业务调整并确认。
- 使用 `--runtime-config` 时，`runtime.yaml` 不包含明文 Secret，且无需顶层 `version`。
- 个人数据接口未错误使用 `--auth oauth2` Client Credentials；采用 OIDC 时已确认当前生成器版本确实支持 `--auth oidc`。
- 命令组和子命令名称对使用人员清晰。
- 查询、写入、更新、删除等关键命令均已在测试环境验证。
- 认证环境变量和额外请求头说明完整。
- 生成项目 `README.md` 已按实际交付方式修订。
- 多分组项目的 `skills/SKILL.md` 可作为 agent loader 入口，并能路由到正确命令组。
- `skills/README.md` 能帮助使用方快速找到对应命令组。
- `skills/<group>/SKILL.md` 中的业务说明、风险提示和示例命令已确认。
- 示例 JSON 请求体已使用真实业务字段更新。
- 已注明生成结果受接口文档准确性影响，接口文档变更后需要重新生成或重新确认。
