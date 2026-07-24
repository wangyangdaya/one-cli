# Token 模式仅配置 Base URL

## 背景

`opencli generate` 默认使用 `token` 认证。当前只要传入 `--runtime-config`，token 模式就要求 YAML 同时包含 `auth` 元数据，并在生成阶段从 `OPENCLI_AUTH_TOKEN` 密封 token。因此下面的配置会被拒绝：

```yaml
base_url: https://api.example.com
```

这阻止了一个有效场景：生成项目固定 Base URL，但 token 由生成后的 CLI 在运行时从环境变量读取。

## 目标行为

当认证模式为 `token` 且 Runtime Config 只包含 `base_url` 时：

- 生成命令成功，不要求生成阶段存在 `OPENCLI_AUTH_TOKEN`。
- 生成项目保留配置的 Base URL。
- 生成后的 CLI 继续从运行时环境读取 `OPENCLI_AUTH_TOKEN`。
- HTTP 请求继续发送 `Authorization: Bearer <token>`。

当 Runtime Config 包含 `auth.type: bearer` 时，保持现有行为：生成阶段要求 `OPENCLI_AUTH_TOKEN`，并将其密封进生成项目。

`api_key` 和 `oauth2` 的严格校验保持不变，因为它们需要 Runtime Config 提供请求头或客户端认证元数据。

## 实现设计

在 `internal/runtimeconfig` 的源配置校验中，把缺少 `auth` 时的 `token` 模式视为有效配置。缺少 `auth` 的配置不会产生密封凭证，现有 Bundle 只携带渲染后的 `base_url`。

不增加新的 YAML 字段或命令行参数。生成后的 Go/Rust 运行时已经支持在没有密封凭证时读取 `OPENCLI_AUTH_TOKEN`，继续复用该回退路径。

## 测试

- 单元测试：`token` 模式、仅含 `base_url`、环境中没有 token 时，`LoadAndSeal` 成功，Bundle 不含 secret。
- 保留回归测试：`api_key` 和 `oauth2` 缺少认证元数据时仍失败。
- 命令或集成测试：使用仅含 `base_url` 的 Runtime Config 生成 Rust 项目成功，生成配置包含 Base URL，不包含密封认证值。
- 运行时测试：生成后的 CLI 可以使用 `OPENCLI_AUTH_TOKEN` 构造 Bearer 请求。

## 文档

README 和用户指南提供完整生成示例，明确区分：

- 仅固定 Base URL，token 在 CLI 运行时提供。
- 在生成阶段将 token 密封进 Runtime Config。
- 完全不使用 Runtime Config，Base URL 和 token 都在 CLI 运行时提供。
