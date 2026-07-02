# Swagger 外部修复链路

当 Swagger 2.0 文档不规范，导致 `opencli generate` 在解析或 Swagger 2.0 转 OpenAPI 3.0 阶段失败时，优先用外部工具修复输入文档，而不是在 `one-cli` 里自研完整转换器。

## 适用场景

- Swagger 2.0 文档来自第三方系统或自动导出工具。
- 文档存在真实世界脏数据，例如多个 `in: body` 参数、非标准 schema、缺失字段或轻微格式问题。
- 需要产出一个可复用、可检查、可提交的 OpenAPI 3.0 文档。
- 需要让 `one-cli` 消费修复后的 OpenAPI 文档，而不是依赖内部兼容逻辑。

如果只是直接生成 CLI，`one-cli` 已经内置 Swagger 2.0 到 OpenAPI 3 的解析路径。只有需要显式导出修复后的 `openapi.yaml` 时，才走这条链路。

## 最短命令

```bash
npx swagger2openapi swagger.yaml \
  -o openapi.yaml \
  -y \
  --patch \
  --targetVersion 3.0.4
```

然后校验：

```bash
openapi-generator-cli validate -i openapi.yaml
```

最后再生成 CLI：

```bash
./dist/opencli generate \
  --target rust \
  --input ./openapi.yaml \
  --output ./tmp/my_cli \
  --module my-cli \
  --app my-cli \
  --app-version 0.0.1 \
  --skill-lang zh
```

## 为什么这样做

`swagger2openapi` 的职责就是把 Swagger 2.0 definitions 转成 OpenAPI 3.0.x。它支持 CLI、Node API、URL 输入、YAML 输出、`--patch` 修复小问题，以及 `--targetVersion` 指定 3.0.x 目标版本。

`openapi-generator-cli validate` 可以校验转换后的 OpenAPI 文档是否结构合法。OpenAPI 3.0.4 是 3.0.x 分支的补丁版本，适合继续留在 OpenAPI 3.0 生态内。

## 重要限制

校验通过不等于业务语义正确。

例如导出工具可能把上传接口写成多个 `in: body` 参数：

```yaml
parameters:
  - in: body
    name: file
  - in: body
    name: isSelf
```

这类文档不符合 Swagger 2.0 规范。外部转换器可能把它修成合法 OpenAPI，但不能保证它就是业务真实语义。真实语义很可能是 `multipart/form-data` 或 `application/x-www-form-urlencoded`。

遇到这类接口时，至少人工确认：

- 文件上传参数是否应该是 multipart file。
- 其他 body 参数是否应该是 form field 或 query 参数。
- 生成后的 CLI flag 是否符合真实调用方式。
- 关键业务路径是否有 E2E 验证。

## 仓库内策略

`one-cli` 内部继续使用 `kin-openapi` 的 `openapi2conv.ToV3` 做 Swagger 2.0 解析。不要为了完整转换能力在仓库里重写 Swagger 到 OpenAPI 的映射。

只在真实输入触发兼容性问题时，补最小兼容逻辑和回归测试。需要稳定导出 `openapi.yaml` 时，使用本文档的外部链路。
