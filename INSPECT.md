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

命令格式：

```bash
opencli inspect \
  --input <openapi-file-or-url> \
  --ai-suggest-config \
  [--output <opencli-ai-yaml>]
```

参数说明：

| 参数 | 是否必需 | 说明 |
| --- | --- | --- |
| `--input` | 必需 | OpenAPI/Swagger 文档路径或 URL |
| `--ai-suggest-config` | 必需 | 启用 AI 命名建议模式 |
| `--output` | 可选 | 将建议写入指定 YAML 文件；不传时输出到 stdout |

最短可运行示例：

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

如果使用已兼容 OpenAI SDK 习惯的地址，也可以把 base URL 写到 `/v1`，OpenCLI 不会重复拼接 `/v1`：

```bash
OPENCLI_AI_BASE_URL=https://api.openai.com/v1 \
OPENCLI_AI_API_KEY=your-api-key \
OPENCLI_AI_MODEL=gpt-4.1-mini \
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

`OPENCLI_AI_BASE_URL` 可以是纯 host、带 `/v1` 的 base URL，或完整的 `/v1/chat/completions` 地址。OpenCLI 会自动补齐缺失部分，并避免重复拼接 `/v1`。

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

## 文档诊断清单

运行 `generate` 前，建议根据 `inspect` 输出和原始文档检查以下问题。这里的目标不是追求文档完美，而是先找出会明显影响 CLI 可用性的部分。

| 检查项 | 风险 | 推荐处理 |
| --- | --- | --- |
| tag 是中文、特殊字符或关键字 | 命令组难输入、包名不稳定 | 用 `naming.tag_alias` 映射为英文业务域名 |
| 缺少 `operationId` | 子命令可能由长路径推导，难使用 | 用 `METHOD path` 配置 `operation_alias` |
| `operationId` 来自代码或平台编号 | 命令名短但语义差 | 用 summary 和业务语义重命名 |
| 路径包含统一前缀，如 `/api-apply/v2/` | 命令名充满传输噪音 | 在 alias 中去掉无意义前缀 |
| summary/description 为空 | AI 和人工都难判断业务含义 | 优先补原文档；不能补时人工审阅 alias |
| 请求体 schema 只是空 object | 生成 CLI 很难展开 flags | 用 `overrides.body_mode` 或补充 schema |
| 所有接口重复认证 header | 参数列表会被认证噪音污染 | 用 `auth` / signer 配置表达全局认证 |
| 404/错误响应描述为“失败” | 文档无法解释真实错误 | 生成后补充 README/Skill 说明或修正文档 |

这些问题不一定都要在 OpenAPI 中修复。命名和请求体输入方式通常优先通过 `opencli.yaml` 调整；接口路径、参数、schema、认证语义错误则应尽量回到源文档修正。

## 常见清洗模式

### 中文或特殊字符 tag

```yaml
naming:
  tag_alias:
    计划物流.: logistics
    供应商管理: supplier
    订单中心-订单: order
```

tag alias 应优先使用业务域名，而不是机械翻译。比如“计划物流.”更适合作为 `logistics` 或 `planning-logistics`，不建议生成难读的拼音或长词串。

### 无 operationId 或命令名过长

当文档没有 `operationId`，或 `operationId` 类似 `supplierMrpMonthUsingPOST_1` 时，建议使用 `METHOD path` 作为 key：

```yaml
naming:
  operation_alias:
    "POST /api-apply/v2/get/supplierMrpMonth": mrp-month
    "POST /api-apply/v2/get/supplierMrpDate": mrp-day
    "POST /api-apply/v2/get/supplierPo": po
    "POST /api-apply/v2/push/supplierConPo": confirm-po
```

`METHOD path` 比单独 path 更安全，因为同一路径可能同时存在 GET、POST、DELETE 等多个方法。

### 路径前缀和平台噪音

内部平台导出的路径常包含 `api`、`v1`、`v2`、`get`、`push`、`supplier` 等噪音。CLI 命名时应优先保留业务名词：

| 原始路径 | 建议命令 |
| --- | --- |
| `/api-apply/v2/get/supplierMrpMonth` | `mrp-month` |
| `/api-apply/v2/get/supplierInvData` | `inventory` |
| `/api-apply/v2/push/supplierConPo` | `confirm-po` |

动词只在能表达用户动作或风险时保留，例如 `confirm-po`、`submit-order`、`delete-user`、`approve-request`。

### 空 object 请求体

如果请求体 schema 只有：

```json
{ "type": "object" }
```

OpenCLI 很难知道应展开哪些 flags。可选处理：

- 能改源文档时，补充 `properties`、`required`、字段类型和说明。
- 不能改源文档时，用 `overrides.body_mode` 明确使用 `file-or-data`。
- 如果已有可靠 example，可用它辅助人工编写 demo request，但不要让 AI 凭空创造字段。

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

## 命名建议规则

人工审阅 `opencli.ai.yaml` 时，可以按以下规则快速判断是否可用：

- 命令组使用业务领域名词，如 `logistics`、`supplier`、`order`、`catalog`。
- 子命令使用 kebab-case，控制在 2-3 个词，例如 `mrp-month`、`confirm-po`。
- 查询类接口通常不需要保留 `get` / `query` / `list`，除非同组内存在歧义。
- 写入、删除、提交、审批、确认等高风险动作应保留动词。
- 不使用中文、空格、下划线、驼峰或标点作为 CLI alias。
- 不让 AI 建议改变 schema、认证、server、请求体模式或原始路径。

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
