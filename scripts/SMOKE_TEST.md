# Smoke Test 脚本说明

`scripts/smoke.sh` 是一个端到端验证脚本。每次修改生成器代码后，运行它即可确认：

1. `dist/opencli` 能正常编译
2. 指定的文档/配置能成功生成项目
3. 生成的项目能编译为可执行文件
4. 可执行文件的 `--help` 输出符合预期

---

## 前置要求

| 工具 | 说明 |
|------|------|
| Go 1.23+ | 必须，用于构建 opencli 和生成的 Go 项目 |
| cargo / rustup | 可选，仅 `--target rust` 时需要 |
| make | 用于 `make build-host`（可用 `--no-build` 跳过） |

---

## 用法

```bash
./scripts/smoke.sh [选项]
```

### 选项

| 选项 | 默认值 | 说明 |
|------|--------|------|
| `--target go\|rust` | `go` | 生成目标语言 |
| `--input <路径>` | — | OpenAPI / Swagger 文档路径（与 `--mcp` 二选一） |
| `--mcp <路径>` | — | MCP 配置文件路径（与 `--input` 二选一） |
| `--app <名称>` | 从文件名推导 | 生成的 CLI 二进制名称和根命令名 |
| `--module <模块>` | 从 app 名推导 | Go module 路径或 Rust package 名 |
| `--no-build` | — | 跳过 `make build-host`，直接使用已有的 `dist/opencli` |

`--input` 和 `--mcp` 互斥，必须且只能提供一个。

### 默认推导规则

- `--app` 未指定时：取文件名去掉扩展名，转小写，加 `-cli` 后缀。  
  例：`petstore.yaml` → `petstore-cli`，`quark.json` → `quark-cli`
- `--module` 未指定时：  
  - Go：`github.com/acme/<app>`  
  - Rust：`<app>`
- 输出目录固定为 `tmp/smoke-<app>-<target>/`，每次运行前自动清空

---

## 示例

### Go + OpenAPI 文档

```bash
./scripts/smoke.sh --target go --input ./examples/petstore.yaml
```

### Rust + OpenAPI 文档

```bash
./scripts/smoke.sh --target rust --input ./examples/petstore.yaml
```

### Go + MCP 配置

```bash
./scripts/smoke.sh --target go --mcp ./examples/quark.json
```

### Rust + MCP 配置

```bash
./scripts/smoke.sh --target rust --mcp ./examples/quark.json
```

### 自定义 app 名和 module，跳过重新构建

```bash
./scripts/smoke.sh \
  --target go \
  --input ./examples/openapi.json \
  --app   nodus \
  --module github.com/myorg/nodus \
  --no-build
```

### 使用自己的 OpenAPI 文档

```bash
./scripts/smoke.sh --target go --input /path/to/my-api.yaml --app my-cli
```

---

## 执行步骤

脚本按顺序执行以下四步，任意步骤失败即停止并以非零退出码退出。

```
Step 0  构建 opencli          make build-host → dist/opencli
Step 1  生成项目              opencli generate ...  → tmp/smoke-<app>-<target>/
Step 2  验证生成文件          检查关键文件是否存在
Step 3  编译生成的项目        go build 或 cargo build
Step 4  运行 --help           验证二进制可执行且输出包含 app 名称
```

---

## 输出示例

```
opencli smoke test
  target  : go
  source  : --input petstore.yaml
  app     : petstore-cli
  module  : github.com/acme/petstore-cli
  output  : .../tmp/smoke-petstore-cli-go

── Step 0: build opencli ──
  ✓ make build-host

── Step 1: generate (go) ──
  $ opencli generate --input .../petstore.yaml ...
  ✓ generate petstore-cli (go)

── Step 2: verify generated files ──
  ✓ file exists: cmd/petstore-cli/main.go
  ✓ file exists: go.mod
  ✓ file exists: go.sum
  ✓ file exists: README.md
  ✓ internal/<group>/command.go exists

── Step 3: build ──
  ✓ go build petstore-cli

── Step 4: run --help ──
  ✓ 'petstore-cli' appears in --help

  $ petstore-cli --help
    petstore-cli CLI

    Usage:
      petstore-cli [flags]
      petstore-cli [command]

    Available Commands:
      pet   pet operations
      ...

── Summary ──
  passed: 7  failed: 0  skipped: 1

SMOKE TEST PASSED
```

---

## 退出码与跳过行为

| 情况 | 退出码 | 说明 |
|------|--------|------|
| 全部通过 | `0` | SMOKE TEST PASSED |
| 任意步骤失败 | `1` | SMOKE TEST FAILED |
| MCP 服务不可达 | `0` | SMOKE TEST SKIPPED，不计为失败 |
| cargo 未安装 | `0` | Rust 构建步骤跳过，不计为失败 |
| cargo 网络受限 | `0` | cargo build 跳过，不计为失败 |

MCP 场景（`--mcp`）依赖外部服务，网络不通或 token 过期时脚本会跳过而非报错，适合在 CI 离线环境中运行。

---

## 生成产物位置

所有生成的项目保存在 `tmp/` 目录下，运行结束后不会自动删除，方便手动检查：

```
tmp/
├── smoke-petstore-cli-go/      # Go 项目源码 + 编译产物
├── smoke-petstore-cli-rust/    # Rust 项目源码 + target/release/
├── smoke-quark-cli-go/         # MCP Go 项目
└── smoke-quark-cli-rust/       # MCP Rust 项目
```

清理：

```bash
make clean        # 删除 dist/ tmp/ bin/
# 或只清理 smoke 产物
rm -rf tmp/smoke-*
```

---

## 典型工作流

```bash
# 1. 修改生成器代码
vim internal/render/templates/go/group_service_http.go.tmpl

# 2. 运行 smoke test（自动重新构建 opencli）
./scripts/smoke.sh --target go --input ./examples/petstore.yaml

# 3. 如果只改了模板，不需要重新构建 opencli
./scripts/smoke.sh --target go --input ./examples/petstore.yaml --no-build

# 4. 同时验证 Rust 路径
./scripts/smoke.sh --target rust --input ./examples/petstore.yaml --no-build
```
