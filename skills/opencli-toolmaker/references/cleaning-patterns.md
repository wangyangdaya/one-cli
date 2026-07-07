# 清洗与修复模式

本文件汇总处理真实业务文档时的常见清洗模式。优先通过 `opencli.yaml` 解决，必要时再修改原始文档。

## 1. 中文/特殊字符 tag

**问题：** tag 名为中文或包含特殊字符，生成后包名退化为 `default` 等关键字，导致编译失败。

**方案：** 在 `opencli.yaml` 中配置 `naming.tag_alias`：

```yaml
naming:
  tag_alias:
    "计划物流.": planlogistics
    "供应商管理": supplier
    "订单中心-订单": order
```

**长期建议：** 建议业务方将 tag 改为英文或有意义的英文缩写。

## 2. 无 operationId 导致命令名过长

**问题：** 接口没有 `operationId`，生成命令名如 `post-api-apply-v2-get-supplierdelstate`。

**方案：** 使用 `METHOD PATH` 作为键配置别名：

```yaml
naming:
  operation_alias:
    "POST /api-apply/v2/get/supplierDelState": del-state
    "POST /api-apply/v2/get/supplierInvData": inv-data
```

**长期建议：** 在原始 OpenAPI 文档中补充 `operationId`。

## 3. 路径前缀冗余

**问题：** 所有路径都有 `/api-apply/v2/` 前缀，命令名里全是噪音。

**方案：** 通过 operation_alias 把长路径映射为短命令。opencli 目前不支持自动剥离路径前缀，别名是最可靠方式。

## 4. 请求体 schema 为空 object

**问题：** `requestBody.schema` 只是 `"type": "object"`，CLI 只能使用 `--data`/`--file`。

**方案 A（推荐）：** 补充 schema：

```json
{
  "requestBody": {
    "content": {
      "application/json": {
        "schema": {
          "type": "object",
          "properties": {
            "date": { "type": "string" },
            "pageSize": { "type": "integer" },
            "pageNum": { "type": "integer" },
            "isForce": { "type": "boolean" }
          }
        }
      }
    }
  }
}
```

**方案 B（无法改文档时）：** 接受 `--data`/`--file` 模式，并确保 `requestBody.example` 存在，以便生成 `demo-request.json`。

## 5. 重复认证 header

**问题：** 每个 operation 都声明 `appKey`、`sign`、`timestamp`、`nonce`。

**短期方案：** 生成后统一在 `internal/<group>/service.go` 中增加签名函数，CLI 暴露 `--app-key` 和 `--secret`。

**中期方案：** 在 `opencli.yaml` 中配置全局运行时认证（如未来支持 `runtime.auth_header` 和签名插件）。

**长期建议：** 在原始 OpenAPI 中用 `securitySchemes` 和 `security` 字段描述认证。

## 6. 响应输出不完整

**问题：** 生成代码默认只输出 `result.Message`，而业务接口返回的是 `code/message/data` 结构。

**方案：** 修改 `internal/<group>/service.go` 和 `types.go`，输出完整 JSON 或格式化 data 字段。

## 7. 404 描述为"失败"

**问题：** 业务方把 404 响应描述为 `"失败"`，语义不准确。

**处理：** opencli 不会纠正业务语义，但在 Skill 文档中应说明该响应码的实际含义。无需修改原始文档，除非用户要求。

## 8. 示例缺失

**问题：** 没有 `example`，生成的 `demo-request.json` 是 `{"demo": true}`。

**方案：** 补充 `requestBody.content.<mediaType>.example` 或 `responses.<code>.content.<mediaType>.example`。
