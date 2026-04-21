# OpenCLI

**从 OpenAPI/Swagger 或 MCP 服务自动生成 Go CLI 工具**

[![Go Version](https://img.shields.io/badge/Go-1.23%2B-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

---

## 📖 简介

`opencli` 是一个代码生成器，它读取 OpenAPI/Swagger API 文档或 MCP 服务定义，在生成时发现可用能力，并自动生成完整的、可运行的 Go CLI 项目。

### 核心特性

- ✅ **自动生成** - 从 OpenAPI 文档或 MCP 服务一键生成完整 CLI 项目
- ✅ **即开即用** - 生成的代码可直接编译运行，无需手动修改
- ✅ **灵活配置** - 支持命名自定义、请求体模式配置等
- ✅ **标准架构** - 基于 Cobra 框架，遵循 Go 最佳实践
- ✅ **本地/远程** - 支持本地文件和远程 URL 作为输入
- ✅ **类型安全** - 生成强类型的 Go 代码

### 工作流程

```
OpenAPI 文档 / MCP 服务 → opencli → Go CLI 项目 → 编译 → 可执行的 CLI 工具
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
  --app petcli

# 3. 直接运行生成的 CLI
cd my-petcli
./bin/petcli --help
./bin/petcli pet list

# 4. 如需分发，编译为真实二进制
# 生成后的 bin/petcli 是启动脚本；下面会用编译产物覆盖它
go build -o bin/petcli ./cmd/petcli
./bin/petcli --help

# 5. 按目标平台生成不同二进制
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
  --app petcli

cd my-petcli-rs

# 当前平台调试构建
cargo build

# 发布构建
cargo build --release

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

- **[用户指南](docs/USER_GUIDE.md)** - 完整的使用说明和实战示例
- **[代码审查报告](docs/CODE_REVIEW_2026-04-20.md)** - 项目代码质量分析
- **[设计文档](docs/superpowers/specs/)** - 架构设计和实现计划
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
  --config ./opencli.yaml  # 可选
```

OpenAPI + Rust：

```bash
opencli generate \
  --target rust \
  --input ./api.yaml \
  --output ./my-cli-rs \
  --module mycli \
  --app mycli
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
| `--config` | ❌ | 配置文件路径（可选） |

`--input` 和 `--mcp-config` 互斥，必须且只能提供一个。

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
- 复杂 schema 会回退为 `--data` / `--file`

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
    users.create: file-or-data    # 支持 --file 和 --data
    users.update: file-or-data
    posts.create: flags            # 展开为 CLI 标志
```

### Body Mode 说明

| 模式 | 说明 | CLI 示例 |
|------|------|----------|
| `file-or-data` | 支持文件或直接数据 | `--file user.json` 或 `--data '{...}'` |
| `flags` | 展开为独立标志 | `--name John --email john@example.com` |

详细配置说明请参考 [用户指南](docs/USER_GUIDE.md#配置文件)。

---

## 📁 生成的项目结构

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
│   └── users/
│       └── SKILL.md       # AI 助手技能文档
├── go.mod
├── go.sum
└── README.md
```

---

## 🎨 映射规则

OpenCLI 按照以下规则将 OpenAPI 或 MCP 元素映射到 CLI 命令：

| OpenAPI 元素 | 生成的 CLI 元素 | 示例 |
|-------------|----------------|------|
| `tags` | 命令组 | `users` → `mycli users` |
| `operationId` | 子命令 | `listUsers` → `mycli users list` |
| `path parameters` | 必需标志 | `{userId}` → `--user-id` |
| `query parameters` | 可选标志 | `?page=1` → `--page 1` |
| `requestBody` | 文件或数据输入 | `--file body.json` |

MCP 映射：

| MCP 元素 | 生成的 CLI 元素 | 示例 |
|---------|----------------|------|
| `server` | 命令组 | `search` → `mycli search` |
| `tool name` | 子命令 | `web-search` → `mycli search web-search` |
| `inputSchema` 简单字段 | CLI flags | `query` → `--query golang` |
| `inputSchema` 复杂结构 | JSON 输入 | `--data '{"filters":[...]}'` |

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
./bin/usercli users create --file new-user.json
```

### 示例 2: 从远程 URL 生成

```bash
opencli generate \
  --input https://petstore.swagger.io/v2/swagger.json \
  --output ./petstore-cli \
  --module github.com/myorg/petstore-cli \
  --app petstore
```

更多示例请参考 [用户指南](docs/USER_GUIDE.md#实战示例)。

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
- `dist/opencli_linux_amd64` - Linux AMD64

单独构建：
```bash
make build-host              # 当前主机
make build-darwin-arm64      # macOS ARM64
make build-linux-amd64       # Linux AMD64
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
./bin/petcli pet create --file pet.json
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
- [x] OpenAPI 3.0 和 Swagger 2.0 支持
- [x] 本地文件和远程 URL 加载
- [x] 命令组和子命令生成
- [x] 参数映射（path, query, body）
- [x] 配置文件支持
- [x] 命名自定义
- [x] 多种 body 处理模式

### 计划中 🚧
- [ ] `opencli init` 命令实现
- [ ] `--version` 标志支持
- [ ] HTTP 重试机制
- [ ] 更详细的错误消息
- [ ] 进度指示器
- [ ] Shell 补全支持
- [ ] 更多 OpenAPI 特性支持（枚举、响应验证等）

详细改进计划请参考 [代码审查报告](docs/CODE_REVIEW_2026-04-20.md)。

---

## 🐛 已知问题

- `opencli init` 命令尚未实现
- 部分复杂的 OpenAPI schema 可能需要手动调整
- 生成的代码需要手动添加实际的 HTTP 请求逻辑

---

## 📄 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。

---

## 🙏 致谢

本项目使用了以下优秀的开源库：

- [Cobra](https://github.com/spf13/cobra) - CLI 框架
- [yaml.v3](https://github.com/go-yaml/yaml) - YAML 解析
- [godotenv](https://github.com/joho/godotenv) - 环境变量加载

---

## 📞 联系方式

- 问题反馈: [GitHub Issues](https://github.com/yourusername/opencli/issues)
- 功能建议: [GitHub Discussions](https://github.com/yourusername/opencli/discussions)

---

**最后更新**: 2026-04-20
