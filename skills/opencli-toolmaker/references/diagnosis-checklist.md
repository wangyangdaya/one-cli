# 输入文档诊断清单

在运行 `opencli generate` 之前，先按本清单评估输入文档质量。每项问题都对应一个清洗动作。

## 基础可解析性

- [ ] 文件格式是 `.json`、`.yaml` 或 `.yml`。
- [ ] 文件能被 `opencli inspect --input <path>` 正常解析，不报错。
- [ ] 如果是 URL，确保网络可达且返回的是原始文档内容（不是 HTML 页面）。

## Tag 与命令组

- [ ] 所有 operation 都归属到至少一个 tag。
- [ ] tag 名不是纯中文/日文/韩文或全特殊字符（如 `计划物流.`）。
- [ ] tag 名不是 Go/Rust 关键字（如 `default`、`type`、`match`）。
- [ ] tag 描述不为空，或至少能从业务上下文推断含义。

## Operation 与命令名

- [ ] 每个 operation 有 `operationId`（推荐），或路径简短且有语义。
- [ ] 没有 operationId 时，由 `METHOD PATH` 推导的命令名不会过长（建议 <30 字符）。
- [ ] operation 的 `summary` 不为空，方便生成 help 和 Skill 文档。

## 参数与请求体

- [ ] path/query/header 参数有 `description`。
- [ ] 请求体 schema 不是单纯的 `"type": "object"`。
- [ ] 请求体字段有 `description` 和明确的 `type`。
- [ ] 请求体有 `example` 或示例字段，便于生成 `demo-request.json`。

## 认证与 Header

- [ ] 如果所有接口都重复同一组 header（如 `appKey`、`sign`、`timestamp`、`nonce`），识别为全局认证。
- [ ] 全局认证应通过 `securitySchemes` 描述，而不是在每个 operation 里重复声明。

## 路径与版本

- [ ] 路径前缀（如 `/api-apply/v2/`）在 CLI 语境下无区分意义时，应通过别名简化。
- [ ] 服务器地址（`servers` 或 `basePath`）已配置，或计划通过环境变量传入。

## 响应

- [ ] 200 响应有 `description`。
- [ ] 200 响应的 `example` 或 schema 可用于生成文档示例。
- [ ] 错误响应描述符合 HTTP 语义（如 404 不建议描述为"失败"）。

## 输出动作

根据诊断结果输出：

1. 必须修复后才能生成的问题（如关键字冲突）。
2. 建议通过 `opencli.yaml` 别名处理的问题。
3. 生成后会进入 `GENERATION_REPORT.md` 的问题。
