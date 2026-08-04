# OpenCLI

**从 OpenAPI/Swagger 或 MCP 服务自动生成 CLI 工具（支持 Go 和 Rust）**

[![Go Version](https://img.shields.io/badge/Go-1.23%2B-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](#-许可证)

---

## 📖 简介

`opencli` 是一个代码生成器，它读取 OpenAPI/Swagger API 文档或 MCP 服务定义，在生成时发现可用能力，并自动生成完整的、可运行的 Go 或 Rust CLI 项目。

### 核心特性

- ✅ **自动生成** - 从 OpenAPI 文档或 MCP 服务一键生成完整 CLI 项目
- ✅ **即开即用** - 生成的代码可直接编译运行，无需手动修改
- ✅ **灵活配置** - 支持命名自定义、请求体模式配置等
- ✅ **标准架构** - 基于 Cobra 框架，遵循 Go 最佳实践
- ✅ **本地/远程** - 支持本地文件和远程 URL 作为输入
- ✅ **类型安全** - 生成强类型的 Go 代码
- ✅ **Skill 产物** - 为每个命令组生成标准化 `skills/<group>/` 工作包，支持英文/中文模板，包含 `SKILL.md`、说明、引用文档、demo request 和生成报告
- ✅ **全版本兼容** - 支持 OpenAPI 2.0（Swagger）、3.0 和 3.1
- ✅ **复杂 Schema** - 完整的 `$ref` 解析、`allOf` 合并、`oneOf`/`anyOf` 处理

### 工作流程

```
OpenAPI 文档 / MCP 服务 → opencli → Go/Rust CLI 项目 + Skill 工作包 → 编译 → 可执行的 CLI 工具
```

---

## 🚀 快速开始

### 前置要求

- Go 1.23.0 或更高版本
- 一个 OpenAPI/Swagger 文档，或一个可连接的 MCP 服务配置

### 安装

```bash
# 克隆仓库
git clone https://github.com/yourusername/opencli.git
cd opencli

# 构建
make build
```

### 5 分钟快速体验

```bash
# 1. 查看示例 API 文档中的接口
./dist/opencli inspect --input ./examples/petstore.yaml

# 2. 生成 CLI 项目
./dist/opencli generate \
  --input ./examples/petstore.yaml \
  --output ./my-petcli \
  --module github.com/myorg/my-petcli \
  --app petcli \
  --app-version 0.0.1 \
  --skill-lang zh

# 3. 直接运行生成的 CLI
cd my-petcli
./bin/petcli --help
./bin/petcli pet list

# 4. 如需分发，编译为真实二进制
# 生成后的 bin/petcli 是启动脚本；下面会用编译产物覆盖它
go build -o bin/petcli ./cmd/petcli
./bin/petcli --version

# 5. 按目标平台生成不同二进制；版本来自生成时的 --app-version
mkdir -p dist/darwin-arm64 dist/linux-amd64 dist/windows-amd64
GOOS=darwin GOARCH=arm64 go build -o dist/darwin-arm64/petcli ./cmd/petcli
GOOS=linux GOARCH=amd64 go build -o dist/linux-amd64/petcli ./cmd/petcli
GOOS=windows GOARCH=amd64 go build -o dist/windows-amd64/petcli.exe ./cmd/petcli
```

Rust + OpenAPI 生成示例：

```bash
./dist/opencli generate \
  --target rust \
  --input ./examples/petstore.yaml \
  --output ./my-petcli-rs \
  --module petcli \
  --app petcli \
  --app-version 0.0.1

cd my-petcli-rs

# 当前平台调试构建
cargo build

# 发布构建
cargo build --release

# Rust CLI 的 --version 来自 Cargo.toml；--app-version 会写入 Cargo.toml
target/release/petcli --version

# 安装额外 target 后，可按目标平台构建
rustup target add aarch64-apple-darwin x86_64-unknown-linux-gnu x86_64-pc-windows-msvc
cargo build --release --target aarch64-apple-darwin
cargo build --release --target x86_64-unknown-linux-gnu
cargo build --release --target x86_64-pc-windows-msvc

# 产物示例
ls target/aarch64-apple-darwin/release
ls target/x86_64-unknown-linux-gnu/release
ls target/x86_64-pc-windows-msvc/release
```

Go + MCP 生成示例：

```bash
./dist/opencli generate \
  --mcp-config ./mcp.json \
  --output ./my-mcp-cli \
  --module github.com/myorg/my-mcp-cli \
  --app quark
```

Rust + MCP 生成示例：

```bash
./dist/opencli generate \
  --target rust \
  --mcp-config ./mcp.json \
  --output ./my-mcp-cli-rs \
  --module quark \
  --app quark
```

---

## 📚 文档

- **[用户指南](USER_GUIDE.md)** - 完整的使用说明
- **[`inspect` 使用说明](INSPECT.md)** - 检查接口文档、生成 AI 命名建议配置
- **[Swagger 外部修复链路](docs/SWAGGER_EXTERNAL_FIX.md)** - 使用 `swagger2openapi` 和 `openapi-generator-cli validate` 修复/校验脏 Swagger 输入
- **[Skill 标准产物](docs/skills/SKILL_STANDARD_OUTPUT.md)** - 生成的 `skills/<group>/` 目录结构、文件职责和扩展规则
- **[Skill 生产化流程](docs/skills/SKILL_PRODUCTION_WORKFLOW.md)** - 如何把生成的 API scaffold 补齐为业务可用 Skill
- **[Swagger Config Skill](docs/skills/swagger-config/SKILL.md)** - 如何根据 Swagger/OpenAPI 文档产出 `opencli.yaml`，再用于生成可读 CLI
- **[Skill 最佳实践样例](docs/skills/examples/skill-best-practice-demo/README.md)** - 可复制参考结构
- **[代码审查报告](docs/CODE_REVIEW_2026-04-20.md)** - 项目代码质量分析
- **设计文档** - 位于 `docs/superpowers/specs/`
- **[开发指南](AGENTS.md)** - 贡献者指南和开发规范

---

## 🎯 核心命令

### `opencli inspect`

检查 OpenAPI 文档中的接口，预览将要生成的命令结构。

```bash
opencli inspect --input ./api.yaml
```

**输出示例**:
```
users GET /users listUsers
users POST /users createUser
users GET /users/{userId} getUser
users DELETE /users/{userId} deleteUser
```

输出格式：`tag method path operationId`

### `opencli generate`

从 OpenAPI 文档或 MCP 服务生成 CLI 项目。默认生成 Go；传 `--target rust` 时生成 Rust。

```bash
opencli generate \
  --input ./api.yaml \
  --output ./my-cli \
  --module github.com/myorg/my-cli \
  --app mycli \
  --app-version 0.0.1 \
  --skill-lang zh \
  --config ./opencli.yaml  # 可选
```

OpenAPI + Rust：

```bash
opencli generate \
  --target rust \
  --input ./api.yaml \
  --output ./my-cli-rs \
  --module mycli \
  --app mycli \
  --app-version 0.0.1
```

MCP + Go：

```bash
opencli generate \
  --mcp-config ./mcp.json \
  --output ./my-cli \
  --module github.com/myorg/my-cli \
  --app mycli
```

MCP + Rust：

```bash
opencli generate \
  --target rust \
  --mcp-config ./mcp.json \
  --output ./my-cli-rs \
  --module mycli \
  --app mycli
```

**参数说明**:

| 参数 | 必需 | 说明 |
|------|------|------|
| `--target` | ❌ | 生成目标：`go` 或 `rust`，默认 `go` |
| `--input` | 二选一 | OpenAPI/Swagger 文档路径或 URL |
| `--mcp-config` | 二选一 | MCP 配置文件路径 |
| `--output` | ✅ | 生成项目的输出目录 |
| `--module` | ✅ | Go target 下是 Go module 路径；Rust target 下用作 Cargo package 名称来源 |
| `--app` | ✅ | CLI 二进制名称和根命令名 |
| `--app-version` | ❌ | 生成出来的 CLI 项目版本；覆盖 `opencli.yaml` 的 `app.version` |
| `--auth` | ❌ | 生成认证模式：`token`、`api_key`、`ak_sk`、`oauth2` 或 `none`；默认 `token` |
| `--signer` | ❌ | AK/SK 签名 profile；当前支持 `supplier_edi` |
| `--skill-lang` | ❌ | 生成的 Skill 文档语言：`en` 或 `zh`，默认 `en` |
| `--config` | ❌ | 配置文件路径（可选） |
| `--runtime-config` | ❌ | 生成后 CLI 的 Base URL 和认证元数据；凭证型模式从环境变量读取并密封，Authorization Code 只写公开配置 |

`--input` 和 `--mcp-config` 互斥，必须且只能提供一个。

`--app-version` 设置的是生成出来的 CLI 项目版本；如果同时配置了 `opencli.yaml` 的 `app.version`，命令行参数优先。

`--skill-lang zh` 会生成中文 `skills/<group>/` 文档；不传时默认生成英文文档。生成出来的文件名保持不变，例如 `SKILL.md`、`README.md`、`generation-report.md`。

#### 认证模式

`--auth` 的值及边界：

| 值 | 凭证来源 | 生成后的请求行为 |
|---|---|---|
| `token` | 已有 Bearer Token | 直接注入 `Authorization: Bearer <token>` |
| `api_key` | API Key | 注入运行配置声明的 API Key 请求头 |
| `ak_sk` | Access Key / Secret Key | 每次请求计算并注入签名 |
| `oauth2` | Runtime 决定；Authorization Code 不使用 Client Secret | 按 `grant_type` 执行 Client Credentials 或用户 Authorization Code，再注入 Bearer Token |
| `none` | 无 | 不注入认证信息 |

未传 `--auth` 时，先读取 `opencli.yaml` 的 `auth.type`；两者都未配置时使用默认值 `token`。因此无需认证的接口应显式使用 `--auth none` 或配置 `auth.type: none`。

`token` 与 `oauth2` 最终都会发送 Bearer Token，但来源不同：`token` 使用调用方已有的 Token；`oauth2` 按 runtime 配置通过 Client Credentials 或用户浏览器授权取得 Token。

`token` 模式不要求提供 Runtime Config。完全不使用 `runtime.yaml` 时，生成阶段不需要设置 `OPENCLI_AUTH_TOKEN`：

```bash
opencli generate \
  --input ./api.yaml \
  --output ./my-cli \
  --module github.com/myorg/my-cli \
  --app mycli \
  --auth token
```

生成后的 CLI 在运行时从环境变量读取 Base URL 和 token：

```bash
export OPENCLI_BASE_URL='https://api.example.com'
export OPENCLI_AUTH_TOKEN='your-token'
./my-cli <group> <command>
```

如果 CLI 只需要固定 Base URL、接口不需要认证，Runtime Config 可以只写：

```yaml
base_url: https://api.example.com
```

生成时应显式使用 `--auth none`：

```bash
opencli generate \
  --input ./api.yaml \
  --output ./my-cli \
  --module github.com/myorg/my-cli \
  --app mycli \
  --auth none \
  --runtime-config ./runtime.yaml
```

如果仅配置 Base URL 但仍需要 token 认证，可以保留默认的 `token` 模式；生成阶段不需要 token，生成后的 CLI 会在运行时读取 `OPENCLI_AUTH_TOKEN`。

#### Token 和 API Key

OpenCLI 不提供 `--token <secret>` 或 `--api-key <secret>` 参数，避免秘密出现在 shell history 和进程参数中。认证类型由 `--auth` 指定；Runtime Config 声明了 `auth`、需要密封凭证时，秘密在生成阶段从环境变量读取。

需要密封 token 时，环境变量由 `opencli generate` 进程自动读取，不需要把 token 写进 `runtime.yaml`，也不需要在生成命令中增加 token 参数。如果当前终端已经导出了 `OPENCLI_AUTH_TOKEN`，直接执行生成命令即可。

Token（Bearer）运行配置源文件统一命名为 `runtime.yaml`：

```yaml
base_url: https://api.example.com
auth:
  type: bearer
```

生成命令：

```bash
export OPENCLI_AUTH_TOKEN='your-token'

opencli generate \
  --input ./api.yaml \
  --output ./my-cli \
  --module github.com/myorg/my-cli \
  --app mycli \
  --auth token \
  --runtime-config ./runtime.yaml
```

Token 只填写原始值，不要包含 `Bearer ` 前缀；生成后的 CLI 会自动构造 `Authorization: Bearer <token>`。如果生成时报错：

```text
Error: OPENCLI_AUTH_TOKEN is required to seal runtime auth
```

说明当前 `opencli` 进程没有读取到该环境变量。请先在同一个终端执行 `export OPENCLI_AUTH_TOKEN='your-token'`，再重新执行生成命令。也可以先用 `printenv OPENCLI_AUTH_TOKEN` 确认变量是否已导出；不要把真实 token 写入仓库文件。

API Key 也使用统一的运行配置文件名 `runtime.yaml`。`header` 是服务端实际接收 API Key 的请求头名称：

```yaml
base_url: https://api.example.com
auth:
  type: api_key
  header: X-API-Key
```

生成命令：

```bash
export OPENCLI_API_KEY='your-api-key'

opencli generate \
  --input ./api.yaml \
  --output ./my-cli \
  --module github.com/myorg/my-cli \
  --app mycli \
  --auth api_key \
  --runtime-config ./runtime.yaml
```

生成器读取环境变量中的凭证，通过 AES-256-GCM 密封后写入生成项目的 `config/runtime.yaml`。该文件只保存 `ENC[v1:...]` 密文，不生成 `runtime.key`；解密材料经过拆分后编译进对应的 Go/Rust CLI。

运行生成后的 CLI 时，也可以用环境变量临时覆盖文件配置：

```bash
# Bearer Token 临时覆盖
OPENCLI_AUTH_TOKEN='temporary-token' ./bin/mycli <group> <command>

# API Key 临时覆盖
OPENCLI_API_KEY='temporary-api-key' ./bin/mycli <group> <command>

# OAuth Client Secret 临时覆盖
OPENCLI_OAUTH_CLIENT_SECRET='temporary-client-secret' ./bin/mycli <group> <command>

# Base URL 或整个配置文件临时覆盖
OPENCLI_BASE_URL='https://staging-api.example.com' ./bin/mycli <group> <command>
OPENCLI_CONFIG='./config/staging.yaml' ./bin/mycli <group> <command>
```

认证值优先级：

```text
命令显式 --header
  > OPENCLI_AUTH_TOKEN / OPENCLI_API_KEY / OPENCLI_OAUTH_CLIENT_SECRET
  > config/runtime.yaml 中的密封凭证
```

其中 `--auth api_key` 必须同时提供 `--runtime-config`，因为生成器需要从 YAML 中读取 API Key 请求头名称。`--auth token` 如果不提供 `--runtime-config`，则不会生成凭证文件，CLI 继续在运行时读取 `OPENCLI_AUTH_TOKEN`。

当前配置组合的边界如下：

| 需求 | 生成方式 |
|---|---|
| Token 模式，不使用 Runtime Config | 不传 `--runtime-config`；生成阶段不需要 token，运行 CLI 时设置 `OPENCLI_BASE_URL` 和 `OPENCLI_AUTH_TOKEN` |
| 只固定 Base URL，Token 在运行时提供 | `runtime.yaml` 只写 `base_url`；使用 `--auth token`，生成阶段不需要 token，运行 CLI 时设置 `OPENCLI_AUTH_TOKEN` |
| 只固定 Base URL，不认证 | `runtime.yaml` 只写 `base_url`，并使用 `--auth none` |
| 固定 Base URL，并把 Token 密封进 CLI | `runtime.yaml` 写 `base_url` 和 `auth.type: bearer`，生成时导出 `OPENCLI_AUTH_TOKEN` |

#### OAuth 2.0

`--auth oauth2` 选择 OAuth2 运行时，具体流程由 `runtime.yaml` 的 `auth.grant_type` 决定。当前支持：

- `client_credentials`：应用身份；生成器可从唯一的 OpenAPI Client Credentials Security Scheme 补齐 Token URL 和 scopes。
- `authorization_code`：用户浏览器授权；生成顶层 `login`、`status`、`logout`，支持固定或动态 loopback 回调、Refresh Token、可选 PKCE S256 和可选 OIDC ID Token 校验。

不使用独立的 `--auth oidc` 或 `--auth oauth2-pkce` 参数。

##### Client Credentials

生成器根据 Token 操作中 `client_id`、`client_secret` 参数的位置识别三方兼容方式。当前支持 `basic`、`body` 和 `query` 三种 Client Secret 位置。

运行配置中的 `client_id` 是公开标识，不加密；`client_secret` 在生成时从 `OPENCLI_OAUTH_CLIENT_SECRET` 读取并密封到 `encrypted_value`：

```yaml
base_url: https://api.example.com
auth:
  type: oauth2
  client_id: my-service-cli
```

当 OpenAPI 声明唯一的 `clientCredentials` Security Scheme 时，生成器会从文档补齐：

```yaml
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

生成命令：

```bash
export OPENCLI_OAUTH_CLIENT_SECRET='your-client-secret'

opencli generate \
  --input ./api.yaml \
  --output ./my-cli \
  --module github.com/myorg/my-cli \
  --app mycli \
  --auth oauth2 \
  --runtime-config ./runtime.yaml
```

运行时，CLI 对声明该 OAuth Security Scheme 的受保护操作自动获取 Access Token 并注入 `Authorization: Bearer <access_token>`。当 `--auth oauth2` 启用时，对应的 Token Endpoint operation 不再生成公开命令，避免产生要求用户传入 `--client-secret` 的重复入口。可使用 `OPENCLI_OAUTH_CLIENT_SECRET` 临时覆盖密封的 Client Secret。

`query` 表示三方 Token Endpoint 要求把 `client_id` 和 `client_secret` 放在 URL Query 中。该方式可能被代理或访问日志记录，生成运行时不会在错误信息中输出完整 Token URL；如果三方支持，应优先使用 `basic` 或 `body`。

##### Authorization Code、PKCE 与 OIDC

最小 Authorization Code 配置：

```yaml
base_url: https://api.example.com
auth:
  type: oauth2
  grant_type: authorization_code
  client_id: business-cli
  authorization_url: https://identity.example.com/authorize
  token_url: https://identity.example.com/token
  redirect_uri: http://127.0.0.1:18081/oauth/callback # 可选；省略则使用随机空闲端口
```

开启 PKCE 和 OIDC：

```yaml
auth:
  type: oauth2
  grant_type: authorization_code
  client_id: business-cli
  authorization_url: https://identity.example.com/authorize
  token_url: https://identity.example.com/token
  scopes: [openid, profile]
  pkce:
    enabled: true
    method: S256
  oidc:
    enabled: true
    issuer: https://identity.example.com
```

PKCE 只支持 `S256`。OIDC 开启时要求 `openid` scope，通过 Discovery/JWKS 校验 RS256 ID Token 的签名、`iss`、`aud`、`azp`、`exp`、`iat` 和 `nonce`，校验成功后才保存 OAuth 会话。ID Token 不作为业务 API Bearer Token，也不写入本地 token 文件。

```bash
./bin/business-cli login
./bin/business-cli login --no-browser
./bin/business-cli status
./bin/business-cli logout
```

默认会话文件位于 `$HOME/.opencli/oauth2/<配置哈希>/oauth-token.json`；可用 `OPENCLI_OAUTH_TOKEN_FILE` 显式覆盖。登录响应包含非空 Refresh Token 和正数 `refresh_token_expires_in` 时，受保护业务命令会在 Access Token 距离到期不足 5 分钟时续期；`status` 本身不发起刷新。当前刷新响应必须返回轮换后的 Access/Refresh Token。

详细配置、业务 Token 接口映射与安全边界见 [`docs/design/oauth2-authorization-code-runtime.md`](docs/design/oauth2-authorization-code-runtime.md) 和 [`oauth2/OAUTH2_AUTHORIZATION_CODE_CONTRACT.md`](oauth2/OAUTH2_AUTHORIZATION_CODE_CONTRACT.md)。

### `opencli package`

将一个已经生成的 Go 或 Rust 项目组装成可以直接安装给 Agent 使用的分组 Skills Bundle：

```bash
opencli package \
  --project ./my-cli \
  --output ./dist/my-cli-skills
```

默认会构建当前平台的 CLI。已有预构建二进制时，可以跳过 Go/Cargo 构建：

```bash
opencli package \
  --project ./my-cli \
  --binary ./release/mycli \
  --output ./dist/my-cli-skills
```

输出结构：

```text
my-cli-skills/
├── SKILL.md                 # 根路由
├── README.md
├── bin/mycli               # 所有分组共用一个 CLI
├── config/runtime.yaml      # 可选的密封运行配置
├── libexec/python/
├── libexec/node/
├── export/SKILL.md          # 一个 API 分组
└── vbt_vehicle_info/SKILL.md
```

再次打包到同一输出目录时：

- 保留每个分组已有的 `SKILL.md`、`README.md`、`references/`、`assets/`、`scripts/` 和未知业务文件；
- 刷新分组 `generation-report.md`、根 `SKILL.md`、根 `README.md`、共享二进制和 `config/runtime.yaml`；
- 不生成 `manifest.yaml`、`runtime.key` 或多余的 `libexec/<cli>/` 目录；
- 新复制的分组文档使用 `../bin/<cli>` 和 `../config/runtime.yaml`，不要求 CLI 预先安装到系统 `PATH`。

AK/SK 接口可以显式生成内置签名逻辑：

```bash
opencli generate \
  --input ./supplier.json \
  --output ./supplier-cli \
  --module github.com/myorg/supplier-cli \
  --app supplier \
  --auth ak_sk \
  --signer supplier_edi

export OPENCLI_AK='your-access-key'
export OPENCLI_SK='your-secret-key'
```

也可以放在 `opencli.yaml`：

```yaml
auth:
  type: ak_sk
  signer:
    profile: supplier_edi
    algorithm: sha512_hex
    headers:
      access_key: appKey
      signature: sign
      timestamp: timestamp
      nonce: nonce
    path:
      strip_prefix: /api-apply
    body:
      order: spec   # spec 保持文档/example 顺序；alpha 按字段名首字母排序
    canonical:
      template: "method={method}&path={path}&appKey={access_key}&appSecret={secret_key}&timestamp={timestamp}&nonce={nonce}&jsonBody={json_body}"
```

`--auth token` 不改变现有 token 行为；生成后的 CLI 继续读取 `OPENCLI_AUTH_TOKEN`。`--auth ak_sk` 会读取 `OPENCLI_AK` / `OPENCLI_SK`。`--signer` 可以覆盖 `auth.signer.profile`；`supplier_edi` 会补齐上面的默认规则。其它 API 可以写自己的 profile、header 名、path 处理、body 顺序和 canonical template；当前算法支持 `sha512_hex`。

推荐按下面理解参数组合：

| 场景 | 必填参数 |
|------|----------|
| OpenAPI/Swagger -> Go | `--input` |
| OpenAPI/Swagger -> Rust | `--target rust --input` |
| MCP -> Go | `--mcp-config` |
| MCP -> Rust | `--target rust --mcp-config` |

注意：

- `--target rust` 只决定生成语言，不会把 `--input` 变成 MCP 模式。
- MCP 配置文件必须配合 `--mcp-config` 使用，不能传给 `--input`。
- 如果把 MCP JSON 传给 `--input`，通常只会得到一个空项目骨架，因为它不是 OpenAPI 文档。

### MCP 配置文件

首版 MCP 生成支持：

- Go target: `streamable_http`、`stdio`
- Rust target: `streamable_http`

Rust 目标当前不支持 `stdio`。

生成时会连接 MCP server，执行 `initialize` 和 `tools/list`，把发现到的 tools 固化为静态 CLI。生成后的 CLI 不依赖 MCP discovery，它直接按生成结果运行。

示例 `mcp.json`：

```json
{
  "servers": {
    "tool-quark-web-search": {
      "transport": "streamable_http",
      "url": "https://example.com/mcp",
      "headers": {
        "Authorization": "Bearer ${MCP_KEY}"
      }
    },
    "local-demo": {
      "transport": "stdio",
      "command": "python",
      "args": ["server.py"],
      "env": {
        "DEBUG": "1"
      }
    }
  }
}
```

MCP tool 参数映射规则：

- 简单 object schema 会展开为独立 flags
- 复杂 schema 会回退为 `--data`（支持内联 JSON、`@文件` 和 stdin）

请求体输入契约：

- `--data '<json>'`：内联 JSON。
- `--data @path.json`：从 JSON 文件读取；`--data -`：从 stdin 读取；`@@...`：传递字面量 `@`。
- `--file path` 或 `--file field=path`：仅用于 OpenAPI/MCP 声明的二进制上传字段。
- multipart 请求中，`--data` 携带文本表单字段，`--file` 携带二进制字段；`--data -` 与 `--file -` 不能同时使用。

### `opencli init`

初始化配置文件（计划中）。

```bash
opencli init
```

---

## ⚙️ 配置文件

通过 `opencli.yaml` 配置文件自定义生成行为。

### 配置示例

```yaml
app:
  binary: mycli
  root_command: mycli
  version: 0.0.1

naming:
  # Tag 别名：重命名命令组
  tag_alias:
    user-management: users
    pet-store: pets
  
  # Operation 别名：重命名子命令
  operation_alias:
    listUsers: list
    createUser: create

runtime:
  # 认证头名称
  auth_header: Authorization
  
  # 默认输出格式
  default_output: pretty

overrides:
  # 请求体处理模式
  body_mode:
    users.create: file-or-data    # 使用 --data（兼容既有配置名）
    users.update: file-or-data
    posts.create: flags            # 展开为 CLI 标志

  # 当 Swagger 导出只保留 raw-json example、丢失 body 字段必传/说明时，可在这里补齐。
  # key 匹配顺序同 body_mode：group.command、tag.command、command、operationId、"method /path"、/path。
  body_fields:
    supplier.kanban-delivery:
      - name: date
        required: true
        type: string
        description: 拉取日期，格式 yyyy-MM-dd
      - name: pageSize
        required: true
        type: integer
        description: 每次数量，最大 1000
```

仓库内置可直接运行的配置示例：

- [`examples/petstore.opencli.yaml`](examples/petstore.opencli.yaml) — 与 `examples/petstore.yaml` 配套，演示 `naming.operation_alias` 缩短命令名、`overrides.body_fields` 补齐字段说明。
- `examples/supplier.opencli.yaml` — 真实供应商 AK/SK 接口配置，演示 `auth.type: ak_sk`、`signer` profile 和复用 body 字段 YAML 锚点（需配合本地 `supplier.json`）。

### Body Mode 说明

| 模式 | 说明 | CLI 示例 |
|------|------|----------|
| `file-or-data` | JSON 数据输入（兼容既有配置名） | `--data @user.json` 或 `--data '{...}'` |
| `flags` | 展开为独立标志 | `--name John --email john@example.com` |

详细配置说明请参考 [用户指南](USER_GUIDE.md) 中的“配置文件”章节。

---

## 📁 生成的项目结构

Go target 示例：

```
my-cli/
├── bin/
│   ├── mycli              # 启动脚本（Unix/Linux/macOS）
│   └── mycli.cmd          # 启动脚本（Windows）
├── cmd/
│   └── mycli/
│       └── main.go        # 主入口
├── internal/
│   ├── cli/               # CLI 框架代码
│   ├── config/            # 配置加载
│   ├── httpx/             # HTTP 客户端
│   ├── output/            # 输出格式化
│   └── users/             # 按 tag 分组的命令
│       ├── command.go     # Cobra 命令定义
│       ├── service.go     # HTTP 请求实现
│       └── types.go       # 类型定义
├── skills/
│   ├── SKILL.md           # 多分组时生成：Agent loader 入口 / Skill 路由
│   ├── README.md          # Skill 索引
│   └── users/
│       ├── SKILL.md
│       ├── README.md
│       ├── assets/
│       │   └── demo-request.json
│       ├── references/
│       │   ├── command-routing.md
│       │   ├── commands.md
│       │   ├── workflows.md
│       │   └── production-checklist.md
│       └── generation-report.md
├── go.mod
├── go.sum
└── README.md
```

Rust target 示例：

```
my-cli-rs/
├── src/
│   ├── main.rs
│   ├── cli.rs
│   ├── client.rs
│   ├── output.rs
│   ├── trace.rs
│   ├── types.rs
│   └── commands/
│       ├── mod.rs
│       └── users.rs
├── skills/
│   ├── SKILL.md           # 多分组时生成：Agent loader 入口 / Skill 路由
│   ├── README.md          # Skill 索引
│   └── users/
│       ├── SKILL.md
│       ├── README.md
│       ├── assets/
│       │   └── demo-request.json
│       ├── references/
│       │   ├── command-routing.md
│       │   ├── commands.md
│       │   ├── workflows.md
│       │   └── production-checklist.md
│       └── generation-report.md
├── Cargo.toml
└── README.md
```

多分组项目会生成根级 `skills/SKILL.md`，作为给 agent loader 使用的路由入口，用于兼容要求所选文件夹下必须存在 `SKILL.md` 的应用。`skills/README.md` 是轻量 Skill 索引。`skills/<group>/` 是标准化 Skill 工作包。分组 `SKILL.md` 是具体 Agent 入口，`README.md` 说明交付和修改流程，`assets/demo-request.json` 是可立即用于 `--data @assets/demo-request.json` 的合法 JSON 占位文件，`references/` 用于补充命令路由、业务流程和生产验收清单，`generation-report.md` 用于列出 API/MCP 输入中缺失的 group、命令、参数或请求体字段描述。

生成的 CLI 还提供读取命令：

```bash
mycli skills list
mycli skills read users
mycli skills read users references/commands.md
mycli skills --skills-dir /path/to/skills read users
```

这些命令读取磁盘上的 `skills/` 文件夹，因此业务调整后的 `SKILL.md` 会立即对 AI agent 生效。默认从当前工作目录读取 `./skills`；如果 CLI 安装在 `/usr/bin`、`/usr/local/bin` 或不在生成项目根目录运行，使用 `--skills-dir /path/to/skills` 显式指定。

中文 Skill 文档可通过 `--skill-lang zh` 生成。风险等级按 HTTP 方法轻量标注：`GET/HEAD/OPTIONS` 为只读，`POST/PUT/PATCH` 为写入，`DELETE` 为高风险；不会按命令名猜测业务语义。

---

## 🎨 映射规则

OpenCLI 按照以下规则将 OpenAPI 或 MCP 元素映射到 CLI 命令：

| OpenAPI 元素 | 生成的 CLI 元素 | 示例 |
|-------------|----------------|------|
| `tags` | 命令组 | `users` → `mycli users` |
| `operationId` | 子命令 | `listUsers` → `mycli users list` |
| `path parameters` | 必需标志 | `{userId}` → `--user-id` |
| `query parameters` | 可选标志 | `?page=1` → `--page 1` |
| `requestBody` | JSON 数据输入 | `--data @body.json` |
| `requestBody` 中的二进制字段 | 文件上传 | `--file path` 或 `--file field=path` |
| `tags` 中的 `*Controller` | 优先提取 Controller 名作为命令组 | `TeMmMriCurrentController` → `te-mm-mri-current` |

MCP 映射：

| MCP 元素 | 生成的 CLI 元素 | 示例 |
|---------|----------------|------|
| `server` | 命令组 | `search` → `mycli search` |
| `tool name` | 子命令 | `web-search` → `mycli search web-search` |
| `inputSchema` 简单字段 | CLI flags | `query` → `--query golang` |
| `inputSchema` 复杂结构 | JSON 输入 | `--data '{"filters":[...]}'` |

生成目录名规则：

- CLI 命令组名保留短横线，例如 `te-mm-mri-current`。
- Go package 和 Rust module 使用语言标识符：非字母数字压缩为 `_`，转小写，必要时增加安全前缀或数字后缀。
- `skills/<skill-name>/` 遵循 Agent Skills 规范：只使用小写字母、数字和短横线，最长 64 个字符，并与 `SKILL.md` frontmatter 的 `name` 完全一致。
- 例如命令组 `te-mm-mri-current` 的代码目录是 `internal/te_mm_mri_current/`（Go）或对应 Rust module，Skill 目录是 `skills/te-mm-mri-current/`。

---

## 💡 使用示例

### 示例 1: 用户管理 API

**OpenAPI 文档** (`users-api.yaml`):
```yaml
openapi: 3.0.0
info:
  title: User Management API
  version: "1.0"
paths:
  /users:
    get:
      tags: [users]
      operationId: listUsers
      parameters:
        - in: query
          name: page
          schema:
            type: integer
      responses:
        "200":
          description: Success
    post:
      tags: [users]
      operationId: createUser
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
      responses:
        "201":
          description: Created
```

**生成 CLI**:
```bash
opencli generate \
  --input ./users-api.yaml \
  --output ./usercli \
  --module github.com/myorg/usercli \
  --app usercli
```

**使用生成的 CLI**:
```bash
cd usercli
go build -o bin/usercli ./cmd/usercli

./bin/usercli users list --page 2
./bin/usercli users create --data @new-user.json
```

### 示例 2: 从远程 URL 生成

```bash
opencli generate \
  --input https://petstore.swagger.io/v2/swagger.json \
  --output ./petstore-cli \
  --module github.com/myorg/petstore-cli \
  --app petstore
```

更多示例请参考 [用户指南](USER_GUIDE.md) 中的“实战示例”章节。

---

## 🛠️ 开发

### 项目结构

```
opencli/
├── cmd/opencli/           # 生成器入口点
├── internal/
│   ├── app/              # CLI 命令定义
│   ├── loaders/          # 文件和 HTTP 加载器
│   ├── openapi/          # OpenAPI 文档解析
│   ├── planner/          # 命令规划和映射
│   ├── render/           # 代码生成和模板渲染
│   ├── model/            # 内部数据模型
│   ├── configgen/        # 配置加载
│   ├── templates/        # Go 模板文件
│   └── runtime/          # 生成项目的运行时代码
├── examples/             # 示例 OpenAPI 文档和项目
├── tests/                # 测试套件
│   ├── unit/            # 单元测试
│   ├── command/         # 命令测试
│   └── integration/     # 集成测试
└── docs/                 # 文档
```

### 开发命令

```bash
# 格式化代码
make fmt

# 运行测试
make test

# 构建
make build

# 清理
make clean
```

### 构建目标

`make build` 会生成以下产物：

- `dist/opencli` - 当前主机版本
- `dist/opencli_darwin_arm64` - macOS ARM64
- `dist/opencli_darwin_amd64` - macOS Intel AMD64
- `dist/opencli_linux_amd64` - Linux AMD64
- `dist/opencli_windows_amd64.exe` - Windows AMD64

单独构建：
```bash
make build-host              # 当前主机
make build-darwin-arm64      # macOS ARM64
make build-darwin-amd64      # macOS Intel AMD64
make build-linux-amd64       # Linux AMD64
make build-windows-amd64     # Windows AMD64
```

---

## 🧪 测试

```bash
# 运行所有测试
make test

# 运行特定测试
go test ./tests/unit/...
go test ./tests/integration/...
go test ./tests/command/...

# 查看测试覆盖率
go test -cover ./...
```

---

## 📦 示例项目

仓库包含 Petstore API 示例，演示如何从 OpenAPI 文档生成 CLI：

```bash
# 1. 查看 API 文档中的接口
opencli inspect --input ./examples/petstore.yaml

# 输出示例：
# pet GET /pets listPets
# pet POST /pets createPet
# pet GET /pets/{petId} getPet

# 2. 生成 CLI 项目
opencli generate \
  --input ./examples/petstore.yaml \
  --output ./tmp/petcli \
  --module github.com/acme/petcli \
  --app petcli

# 3. 构建并使用
cd tmp/petcli
go build -o bin/petcli ./cmd/petcli
./bin/petcli --help
./bin/petcli pet list
./bin/petcli pet create --data @pet.json
./bin/petcli pet get --pet-id 123
```

---

## 🤝 贡献

欢迎贡献！请查看 [AGENTS.md](AGENTS.md) 了解：

- 项目结构和模块组织
- 编码风格和命名规范
- 测试指南
- 提交和 PR 指南

### 贡献流程

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 开启 Pull Request

---

## 📋 路线图

### 已完成 ✅
- [x] OpenAPI 2.0（Swagger）、3.0 和 3.1 全版本支持
- [x] 基于 kin-openapi 的完整 `$ref` 解析（含嵌套引用链）
- [x] `allOf` 属性自动合并，支持复合 schema 展开为 CLI flag
- [x] `oneOf`/`anyOf` 正确识别并回退为 `--data` 模式
- [x] 本地文件和远程 URL 加载
- [x] 命令组和子命令生成
- [x] 参数映射（path, query, body）
- [x] 配置文件支持
- [x] 命名自定义
- [x] 多种 body 处理模式
- [x] 生成项目 `--version` 支持（Go/Rust）
- [x] 标准化 Skill 工作包生成（Go/Rust，含 `skills/README.md`、`assets/demo-request.json`、`generation-report.md`，支持 `--skill-lang en|zh`）
- [x] Go/Rust 生成项目支持 `skills list/read` 查看生成的 Skill 文档

### 计划中 🚧
- [ ] `opencli init` 命令实现
- [ ] Rust 生成目标 `--trace` 支持
- [ ] HTTP 重试机制
- [ ] 更详细的错误消息
- [ ] 进度指示器
- [ ] Shell 补全支持
- [ ] 更多 OpenAPI 特性支持（枚举、响应验证等）

详细改进计划请参考 [代码审查报告](docs/CODE_REVIEW_2026-04-20.md)。

---

## 🐛 已知问题

- `opencli init` 命令尚未实现
- Rust 生成目标不支持 `--trace` flag（Go 目标已支持）
- Rust 生成目标不支持 MCP `stdio` 传输
- 生成的 Skill 是 API scaffold，需要补充业务意图、流程和安全边界后才算生产可用；生产验收还需验证 `--data @文件`/stdin 以及二进制字段的 `--file` multipart 行为

---

## 📄 许可证

本项目采用 MIT 许可证。

---

## 🙏 致谢

本项目使用了以下优秀的开源库：

- [kin-openapi](https://github.com/getkin/kin-openapi) - OpenAPI 2.0/3.0/3.1 解析与验证
- [Cobra](https://github.com/spf13/cobra) - CLI 框架
- [yaml.v3](https://github.com/go-yaml/yaml) - YAML 解析
- [godotenv](https://github.com/joho/godotenv) - 环境变量加载

---

## 📞 联系方式

- 问题反馈: [GitHub Issues](https://github.com/yourusername/opencli/issues)
- 功能建议: [GitHub Discussions](https://github.com/yourusername/opencli/discussions)

---

**最后更新**: 2026-07-01
