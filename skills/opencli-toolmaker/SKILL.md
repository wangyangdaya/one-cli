---
name: opencli-toolmaker
version: 1.0.0
description: "使用 opencli 将 OpenAPI/Swagger 或 MCP 服务文档转换为可运行的 Go/Rust CLI 项目和标准化 Skill 包，并指导如何清洗、修复和优化输入文档。"
metadata:
  requires:
    bins: ["opencli", "go"]
    files: ["./dist/opencli"]
---

# opencli-toolmaker

本 Skill 指导 Agent 使用 `opencli` 工具，将业务方提供的 OpenAPI/Swagger 或 MCP 服务文档转换为可编译、可运行的 CLI 项目，并进一步打包成标准化 Skill。

## 适用范围

- 用户提供了 `.json`、`.yaml`、`.yml` 格式的 OpenAPI/Swagger 文档。
- 用户提供了 MCP 服务配置（如 `mcp.json`）。
- 用户希望自动生成 CLI 工具或配套 Skill 包。
- 用户遇到生成后的 CLI 编译失败、命令名过长、中文 tag、请求体无法展开等问题。

## 核心原则

1. **先诊断，再生成**：拿到文档后不要直接 `generate`，先用 `inspect` 和诊断清单评估质量。
2. **优先不改原始文档**：清洗和别名优先通过 `opencli.yaml` 完成，除非用户明确要求修改原始文件。
3. **生成必须验证**：每次 `generate` 后必须执行 `go build` 或 `cargo build` 验证可编译性。
4. **报告必须阅读**：生成后的 `skills/<group>/GENERATION_REPORT.md` 必须过一遍，列出需要业务方补充的内容。

## 标准工作流

### 步骤 1：接收文档并确认来源

询问或确认：

- 文档路径或 URL
- 目标语言：`go`（默认）或 `rust`
- 生成的 CLI 名称（`--app`）
- Go module 路径（Go target）或 Cargo package 名称（Rust target）
- 是否需要配套 Skill 包（默认生成）

### 步骤 2：诊断文档

执行：

```bash
opencli inspect --input <document>
```

按 `references/diagnosis-checklist.md` 逐项检查，重点关注：

- tag 名是否包含中文、特殊字符或 Go/Rust 关键字
- 是否存在 `operationId`
- 命令名是否会过长
- 请求体 schema 是否只是空 `"type": "object"`
- 是否有重复出现的认证 header
- 路径前缀是否冗余

### 步骤 3：准备 opencli.yaml

根据诊断结果创建 `opencli.yaml`。常见配置：

```yaml
app:
  binary: <app-name>
  root_command: <app-name>
  version: 0.0.1

naming:
  # 中文/特殊字符 tag 映射为英文命令组
  tag_alias:
    "计划物流.": planlogistics

  # 无 operationId 时，用 "METHOD PATH" 作为键简化命令名
  operation_alias:
    "POST /api-apply/v2/get/supplierDelState": del-state
    "POST /api-apply/v2/get/supplierInvData": inv-data

overrides:
  # 需要强制展开为 flags 的请求体
  body_mode:
    planlogistics.del-state: flags
```

详细规范见 `references/naming-guide.md`。

### 步骤 4：生成项目

```bash
opencli generate \
  --input <document> \
  --output ./<app-name>-cli \
  --module github.com/<org>/<app-name>-cli \
  --app <app-name> \
  --app-version 0.0.1 \
  --config ./opencli.yaml
```

Rust target：

```bash
opencli generate \
  --target rust \
  --input <document> \
  --output ./<app-name>-cli-rs \
  --module <app-name> \
  --app <app-name> \
  --app-version 0.0.1 \
  --config ./opencli.yaml
```

MCP source：

```bash
opencli generate \
  --mcp-config ./mcp.json \
  --output ./<app-name>-cli \
  --module github.com/<org>/<app-name>-cli \
  --app <app-name> \
  --config ./opencli.yaml
```

### 步骤 5：编译验证

Go：

```bash
cd <app-name>-cli
go build -o bin/<app-name> ./cmd/<app-name>
./bin/<app-name> --help
./bin/<app-name> <group> --help
./bin/<app-name> <group> <command> --help
```

Rust：

```bash
cd <app-name>-cli-rs
cargo build --release
./target/release/<app-name> --help
```

### 步骤 6：交付与交接

1. 阅读 `skills/<group>/GENERATION_REPORT.md`，确认缺陷项。
2. 检查 `skills/<group>/assets/demo-request.json` 是否基于真实 example。
3. 如需业务方补充路由/工作流，指导其填写 `references/command-routing.md` 和 `references/workflows.md`。
4. 按 `references/production-checklist.md` 逐项确认后视为可用。

## 清洗与修复速查

| 问题 | 快速处理 | 长期建议 |
|------|---------|---------|
| 中文 tag 导致编译失败 | `naming.tag_alias` 映射为英文 | 业务方修改原始 tag 名 |
| 无 operationId，命令名过长 | `naming.operation_alias` 用 `METHOD PATH` 映射 | 补充 `operationId` |
| 路径前缀冗余 | 通过 operation_alias 把长路径映射为短命令 | 业务方简化 API 路径 |
| 请求体 schema 为空 object | 接受 `--data`/`--file` 模式 | 补充完整 schema |
| 重复认证 header | 生成后统一封装签名函数 | 在 OpenAPI 中用 securitySchemes 描述 |
| 响应只输出 `message` | 修改 `internal/<group>/service.go` 输出完整 JSON | 无需修改原始文档 |

## 安全约束

1. **不要编造 ID**：多步骤工作流中，ID 必须从列表/查询命令返回，禁止凭空构造。
2. **写入/删除/批量/不可逆操作需确认**：执行前向用户展示目标标识和预期影响，请求明确授权。
3. **不要修改用户原始文档**：除非用户明确授权，否则清洗通过 `opencli.yaml` 完成。
4. **生成后必须编译验证**：不能仅因 `opencli generate` 成功就认为项目可用。

## 常见命令

```bash
# 预览命令结构
opencli inspect --input ./api.yaml

# 生成 Go CLI
opencli generate --input ./api.yaml --output ./my-cli --module github.com/myorg/my-cli --app mycli

# 生成 Rust CLI
opencli generate --target rust --input ./api.yaml --output ./my-cli-rs --module mycli --app mycli

# 生成 MCP CLI
opencli generate --mcp-config ./mcp.json --output ./my-mcp-cli --module github.com/myorg/my-mcp-cli --app mycli
```

## 参考文档

- `references/diagnosis-checklist.md` — 输入文档诊断清单
- `references/cleaning-patterns.md` — 常见清洗与修复模式
- `references/naming-guide.md` — 命名与别名规范
- `references/generation-workflow.md` — 完整生成工作流
- `references/troubleshooting.md` — 问题排查与修复
