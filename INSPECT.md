# `opencli inspect` 使用说明

`opencli inspect` 用于在生成 CLI 项目前检查 OpenAPI/Swagger 文档。它可以帮助你确认接口是否被正确识别，也可以在文档命名不规范时生成一份可审阅的 `opencli.yaml` 命名建议。

建议所有正式生成流程都先执行 `inspect`，再执行 `generate`。

## 基本检查

检查本地 OpenAPI/Swagger 文档：

```bash
opencli inspect --input ./api.yaml
```

检查远程文档：

```bash
opencli inspect --input https://example.com/openapi.json
```

输出示例：

```text
users GET /users listUsers
users POST /users createUser
users GET /users/{userId} getUser
```

每一行代表一个接口操作，格式为：

```text
tag method path operationId
```

| 字段 | 含义 | 对生成结果的影响 |
| --- | --- | --- |
| `tag` | 接口分组 | 通常生成一级命令组 |
| `method` | HTTP 请求方法 | 决定实际请求方式 |
| `path` | API 路径 | 决定实际请求地址和 path 参数 |
| `operationId` | 操作 ID | 通常生成子命令名称 |

如果输出中的接口数量、分组、路径或 `operationId` 与预期不一致，应先修正接口文档或补充 `opencli.yaml`，再生成 CLI。

## JSON 输出

需要给脚本或流水线消费时，可以使用 JSON 输出：

```bash
opencli --json inspect --input ./api.yaml
```

输出会包含 `operations` 数组，适合做接口数量检查、变更对比或自动化校验。

`--json` 只用于普通 inspect。AI 建议配置模式当前不支持 `--json`。

## AI 生成命名配置建议

有些接口文档虽然合法，但命名不适合直接生成 CLI。例如：

```text
中文. POST /api/v2/get/xx
中文. POST /api/v2/get/xxx
中文. POST /api/v2/push/xxxx
```

这类文档的 `tag`、路径和 `operationId` 往往来自内部平台或接口代码名，生成的命令可能不直观。可以使用 AI 建议模式生成 `opencli.yaml` 草稿：

```bash
OPENCLI_AI_BASE_URL=https://your-openai-compatible-host \
OPENCLI_AI_API_KEY=your-api-key \
OPENCLI_AI_MODEL=your-model \
opencli inspect --input ./supplier.json --ai-suggest-config
```

输出示例：

```yaml
naming:
  tag_alias:
    计划物流.: logistics
  operation_alias:
    POST /api-apply/v2/get/supplierMrpMonth: mrp-month
    POST /api-apply/v2/get/supplierPo: po
    POST /api-apply/v2/push/supplierConPo: confirm-po
```

也可以写入文件：

```bash
OPENCLI_AI_BASE_URL=https://your-openai-compatible-host \
OPENCLI_AI_API_KEY=your-api-key \
OPENCLI_AI_MODEL=your-model \
opencli inspect \
  --input ./supplier.json \
  --ai-suggest-config \
  --output ./opencli.ai.yaml
```

然后带配置生成：

```bash
opencli generate \
  --input ./supplier.json \
  --config ./opencli.ai.yaml \
  --output ./supplier-cli \
  --module github.com/example/supplier-cli \
  --app supplier
```

## AI 环境变量

AI 建议模式通过 OpenAI-compatible Chat Completions 接口调用模型，需要以下环境变量：

| 环境变量 | 是否必需 | 说明 |
| --- | --- | --- |
| `OPENCLI_AI_BASE_URL` | 必需 | OpenAI-compatible 服务地址 |
| `OPENCLI_AI_API_KEY` | 必需 | 模型服务 API key |
| `OPENCLI_AI_MODEL` | 必需 | 模型名称 |

`OPENCLI_AI_BASE_URL` 会自动拼接 `/v1/chat/completions`。

## AI 建议模式的安全边界

AI 建议模式不会直接修改原始 OpenAPI/Swagger 文档，也不会自动执行 `generate`。它只生成一份 `opencli.yaml` 命名建议，供人工审阅后使用。

发送给模型的信息是精简后的接口清单，包括：

- 文档标题
- HTTP 方法
- path
- tag
- operationId
- summary

不会发送请求示例、响应示例、认证密钥、环境变量值或额外请求头。

## 本地校验规则

模型返回的建议会先经过本地校验，只有合法项才会进入输出 YAML。

校验规则包括：

- `tag_alias` 的 key 必须来自文档中真实存在的 tag。
- `operation_alias` 的 key 必须匹配真实接口，优先使用 `METHOD path`。
- alias 只能使用小写 ASCII 字母、数字和连字符。
- alias 不能为空，不能以连字符开头或结尾。
- 同一命令组内的子命令 alias 不能重复。

被拒绝的建议会打印到 stderr，例如：

```text
rejected operation_alias "POST /missing" -> "missing": unknown operation
analyzed 17 operations
```

如果模型没有产生任何有效建议，命令会失败，并提示重新检查文档或更换模型重试。

## 推荐流程

```text
1. opencli inspect --input ./api.yaml
2. 检查接口数量、tag、path、operationId 是否合理
3. 如命名不适合 CLI，运行 --ai-suggest-config 生成 opencli.ai.yaml
4. 人工审阅并调整 opencli.ai.yaml
5. opencli generate --config ./opencli.ai.yaml
6. 检查生成项目 README.md 和 skills/<group>/SKILL.md
```

正式交付时，不建议直接使用未经审阅的 AI 输出。至少应确认命令组、子命令名称和高风险操作名称是否符合业务习惯。

## 常见问题

### `inspect` 输出为空或接口数量不对怎么办？

优先检查输入文档是否是合法 OpenAPI/Swagger 文档，以及接口是否真的写在 `paths` 下。也可以先用 OpenAPI 校验工具确认文档结构。

### 中文 tag 是否可以直接生成？

解析阶段可以识别中文 tag，但生成 CLI 时更推荐通过 `naming.tag_alias` 映射成适合 shell 使用的英文命令组，例如：

```yaml
naming:
  tag_alias:
    计划物流.: logistics
```

### AI 生成的名称不满意怎么办？

直接编辑生成的 `opencli.ai.yaml`。AI 输出只是草稿，最终以人工确认后的 `opencli.yaml` 为准。

### 为什么不直接让 AI 修改 OpenAPI？

第一版选择生成 `opencli.yaml`，是为了保持可审阅、可 diff、可回滚。直接修改 OpenAPI 容易误改 schema、路径、认证或请求体语义，风险更高。

### 什么时候不需要 AI 建议？

如果文档本身已有清晰的 tag、operationId 和 summary，直接使用普通 `inspect` 检查后生成即可。AI 建议主要用于内部平台导出的、命名质量较差的接口文档。
