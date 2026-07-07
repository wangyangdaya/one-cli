# 完整生成工作流

本文件定义从接收文档到交付 Skill 的标准流程。

## 阶段 1：接收与确认

**输入：**

- OpenAPI/Swagger 文档路径或 URL
- MCP 配置文件路径
- 目标语言：`go` 或 `rust`
- CLI 名称、版本、module/package 名

**动作：**

1. 确认文档来源和格式。
2. 确认输出目录。
3. 确认是否需要 `--config`。

## 阶段 2：诊断

```bash
opencli inspect --input <document>
```

**输出：** `tag method path operationId` 列表。

**评估项：**

- tag 名是否可安全用作包名
- 是否有 operationId
- 命令名是否过长
- 请求体 schema 质量
- 重复 header

## 阶段 3：清洗与配置

根据诊断结果创建 `opencli.yaml`：

- 中文 tag → `tag_alias`
- 无 operationId 长路径 → `operation_alias`
- 需要展开 body → `overrides.body_mode`

参考 `cleaning-patterns.md` 和 `naming-guide.md`。

## 阶段 4：生成

```bash
opencli generate \
  --input <document> \
  --output ./<app-name>-cli \
  --module github.com/<org>/<app-name>-cli \
  --app <app-name> \
  --app-version 0.0.1 \
  --config ./opencli.yaml
```

## 阶段 5：编译验证

Go：

```bash
cd <app-name>-cli
go build -o bin/<app-name> ./cmd/<app-name>
./bin/<app-name> --help
./bin/<app-name> <group> <command> --help
```

Rust：

```bash
cd <app-name>-cli-rs
cargo build --release
./target/release/<app-name> --help
```

## 阶段 6：Skill 检查

1. 阅读 `skills/<group>/GENERATION_REPORT.md`。
2. 检查 `skills/<group>/assets/demo-request.json`。
3. 确认 `SKILL.md` 中的命令索引与实际 CLI 一致。

## 阶段 7：业务补充

指导业务方/用户补充：

- `references/command-routing.md`：用户意图到命令的映射
- `references/workflows.md`：跨命令业务流程
- `references/production-checklist.md`：上线检查

## 阶段 8：交付

交付物：

- 可编译的 CLI 项目目录
- 配套 Skill 包目录
- `opencli.yaml` 配置（便于重新生成）
- 生成报告和待补充清单
