# EDI OpenAPI 标准标注版设计

## 目标

以 `examples/edi_openapi.3.0.json` 为输入，新增
`examples/edi_openapi.3.0.annotated.json`。原文件保持不变，新文件应能被
`opencli generate` 正常解析，并让生成的技能文档展示明确的请求参数和返回字段。

## 改造范围

- 保留现有 OpenAPI 版本、接口路径、HTTP 方法、标签、服务器地址和扩展字段。
- 修正 `getList` 请求 Schema，使字段与示例中的 `tableName`、`field`、`orderBy` 一致。
- 保留 `getPage` 和 `autoLogin` 已声明的请求字段，并补齐清晰的中文说明和合理的必填声明。
- 为需要鉴权的业务接口声明 `cli-token` Header；不把登录接口本身标为需要该 Header。
- 为成功和失败响应补充 `schema.properties`、字段类型、必填项和中文说明。
- 把字符串形式的 `example` 改为真正的 JSON 对象。

## 返回结构

- 列表接口：`code`、`data`、`message`；`data` 为对象数组。
- 分页接口：`code`、`data`、`message`；分页 `data` 包含 `total`、`size`、`current`、`records`。
- 登录接口：`code`、`data`、`message`；`data` 为 Token 字符串。
- 失败响应：`code`、`message`。
- 示例中的记录对象没有业务字段定义，因此使用开放对象，不猜测字段名称。

## 复用策略

本次采用就地 Schema，而不引入大规模 `components/schemas` 重构。这样新文档仍与原文件结构接近，便于对照和维护；嵌套分页对象会完整声明，确保生成器能够展示顶层字段类型。

## 验证

1. 使用 JSON 解析器校验文件语法。
2. 使用 `opencli generate` 生成临时项目。
3. 检查生成的 `skills/public/SKILL.md` 是否包含请求字段、Header 和返回字段。
4. 确认原始 `examples/edi_openapi.3.0.json` 未被修改。
