# opencli

`opencli` 用 Swagger/OpenAPI 文档生成可运行的 Go CLI 项目。

当前仓库保留两部分内容：

- `opencli` 生成器本身
- `examples/` 下的示例资源和 demo，方便以后复用

## Commands

- `opencli init`
- `opencli inspect --input ./examples/petstore.yaml`
- `opencli generate --input ./examples/petstore.yaml --output ./tmp/petcli --module github.com/acme/petcli --app petcli`

## Config

初始化后的配置文件建议命名为 `opencli.yaml`。示例见 [examples/opencli.yaml](/Users/chery-90507455/Documents/workspace/one-cli/examples/opencli.yaml)。

## Usage

### 1. 查看 OpenAPI 文档中的接口

```bash
go run ./cmd/opencli inspect --input ./examples/petstore.yaml
```

输出会按 `tag method path operationId` 列出接口，方便确认生成结果的命令分组和命令名。

### 2. 生成 CLI 项目

```bash
go run ./cmd/opencli generate \
  --input ./examples/petstore.yaml \
  --output ./tmp/petcli \
  --module github.com/acme/petcli \
  --app petcli
```

生成结果示例：

```text
tmp/petcli/
  bin/petcli
  cmd/petcli/main.go
  internal/pet/command.go
  internal/pet/service.go
  internal/pet/types.go
  internal/cli/...
  internal/config/...
  internal/httpx/...
  skills/pet/SKILL.md
  README.md
  go.mod
  go.sum
```

### 3. 运行生成后的 CLI

```bash
./tmp/petcli/bin/petcli --help
./tmp/petcli/bin/petcli pet --help
```

### 4. 用于类似 curl 的 HTTP API

如果接口原本是这样的：

```bash
curl -X POST 'https://example.com/v1/chat-messages' \
  --header 'Authorization: Bearer {api_key}' \
  --header 'Content-Type: application/json' \
  --data-raw @body.json
```

需要先把接口描述成 OpenAPI 文档，再交给 `opencli generate`。当前生成器已经支持：

- 按 `tag` 生成命令组
- 按 `operationId` 生成子命令
- 识别 request body 并生成运行时骨架

当前还没有自动补齐所有 header、query、path、body 字段到最终 CLI flags，所以它更适合先生成 Go CLI 工程骨架、`bin/` 启动脚本和 `skills/` 文档，再在生成项目里补具体参数映射。

## Examples

示例 demo 已迁移到 `examples/` 下，例如：

```bash
go run ./examples/one-leave/cmd/one-leave --help
```

## Development

- `make fmt`
- `make test`
- `make build`

## Build Targets

`make build` 会生成 `opencli` 当前主机版本，以及常用平台产物：

- `dist/opencli`
- `dist/opencli_darwin_arm64`
- `dist/opencli_linux_amd64`

也可以单独执行：

- `make build-host`
- `make build-darwin-arm64`
- `make build-linux-amd64`
- `make build-example-one-leave`
- `make clean`
