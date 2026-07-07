# 问题排查与修复

## 生成阶段

### `opencli inspect` 解析失败

**现象：** 报错无法解析文档。

**排查：**

- 文件格式是否正确（JSON/YAML）。
- 文件编码是否为 UTF-8。
- URL 是否返回原始文档（检查 Content-Type）。
- OpenAPI 版本是否支持（2.0/3.0/3.1）。

### 生成的命令名过长

**原因：** 没有 operationId，路径又长。

**修复：** 在 `opencli.yaml` 中使用 `METHOD PATH` 作为键配置 `operation_alias`。

```yaml
naming:
  operation_alias:
    "POST /api-apply/v2/get/supplierDelState": del-state
```

## 编译阶段

### `expected 'IDENT', found 'default'`

**原因：** tag 名为中文/特殊字符，生成包名退化为 `default`，而 `default` 是 Go 关键字。

**修复：** 配置 `naming.tag_alias`：

```yaml
naming:
  tag_alias:
    "计划物流.": planlogistics
```

### `missing import path`

**原因：** `bin/<app>` 启动脚本使用 `go run "$DIR/cmd/<app>"`，在部分 Go 版本下不合法。

**修复：** 直接编译运行：

```bash
cd <app>-cli
go build -o bin/<app> ./cmd/<app>
./bin/<app> --help
```

或修改生成后的 `bin/<app>` 脚本为：

```sh
cd "$DIR" && exec go run "./cmd/<app>" "$@"
```

### 其他 Go 关键字冲突

如果 tag/operation 清理后命中 `type`、`func`、`range`、`match` 等关键字，同样需要用别名规避。

## 运行阶段

### 命令只有 `--data`/`--file`，没有独立 flags

**原因：** 请求体 schema 是空 object，生成器无法识别字段。

**修复：**

1. 在原始文档中补充 schema。
2. 或在 `opencli.yaml` 中强制指定 `overrides.body_mode`（仅当字段简单时有效）。

### 每次都要传认证 header

**原因：** 原始文档在每个 operation 里重复声明 header 参数。

**修复：** 生成后在 `internal/<group>/service.go` 中封装签名函数，CLI 暴露 `--app-key` 和 `--secret`。

### 响应只输出 `message`，看不到 `data`

**原因：** 默认 `Result` 结构只包含 `Message`。

**修复：** 修改 `internal/<group>/types.go` 和 `service.go`，输出完整响应 JSON。

## Skill 阶段

### `demo-request.json` 是 `{"demo": true}`

**原因：** 没有请求体 example。

**修复：** 在原始 OpenAPI 中补充 `requestBody.content.<mediaType>.example`。

### GENERATION_REPORT 提示大量缺失

**处理：** 区分哪些问题会阻塞使用，哪些只需后续补充。优先修复编译和命令可用性问题，描述和示例可后续人工完善。
