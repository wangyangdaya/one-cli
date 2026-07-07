# 命名与别名规范

好的命名能显著降低生成后 CLI 的使用门槛。

## tag 别名（命令组名）

用于把原始 tag 映射为 CLI 中的二级命令。

**规则：**

- 使用小写英文单词。
- 优先使用业务领域名词，如 `planlogistics`、`supplier`、`order`。
- 避免使用 Go/Rust 关键字：`default`、`type`、`func`、`match`、`impl` 等。
- 避免使用纯数字开头。

**示例：**

```yaml
naming:
  tag_alias:
    "计划物流.": planlogistics
    "供应商管理": supplier
    "订单模块": order
```

## operation 别名（子命令名）

用于重命名子命令。支持两种键：

### 按 operationId

当原始文档有 `operationId` 时使用：

```yaml
naming:
  operation_alias:
    listUsers: list
    createUser: create
```

### 按 METHOD PATH

当原始文档没有 `operationId` 时使用：

```yaml
naming:
  operation_alias:
    "POST /api-apply/v2/get/supplierDelState": del-state
    "POST /api-apply/v2/get/supplierInvData": inv-data
    "POST /api-apply/v2/push/supplierConDate": con-date
```

**规则：**

- 子命令名使用 kebab-case，如 `del-state`、`mrp-date`。
- 尽量控制在 2-3 个词。
- 与业务语义对齐，如 `del-state` 对应「看板配送单状态」。
- 避免与 tag 别名组合后产生歧义，如 `planlogistics plan` 比 `planlogistics post-api-apply-v2-get-supplierproplaning` 清晰。

## body_mode 覆盖

用于强制指定某个命令的请求体输入模式。

```yaml
overrides:
  body_mode:
    planlogistics.del-state: flags
    planlogistics.po: file-or-data
```

**模式说明：**

- `flags`：把简单 JSON 字段展开为独立 flags。
- `file-or-data`：使用 `--file` 或 `--data` 传入完整请求体。

## 完整示例

```yaml
app:
  binary: suppliercli
  root_command: suppliercli
  version: 0.0.1

naming:
  tag_alias:
    "计划物流.": planlogistics
  operation_alias:
    "POST /api-apply/v2/get/supplierDelState": del-state
    "POST /api-apply/v2/get/supplierInvData": inv-data
    "POST /api-apply/v2/get/supplierMrpDate": mrp-date
    "POST /api-apply/v2/get/supplierMrpMonth": mrp-month
    "POST /api-apply/v2/get/supplierPo": po
    "POST /api-apply/v2/push/supplierConDate": con-date

overrides:
  body_mode:
    planlogistics.del-state: file-or-data
```
