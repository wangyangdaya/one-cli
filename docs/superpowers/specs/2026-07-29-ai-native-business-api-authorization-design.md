# 企业业务系统 AI Native 认证授权架构与开发规范

## 1. 文档摘要

本方案用于指导企业内部业务系统把传统 API 改造成可安全提供给客户端 AI Agent、生成式 CLI 和自动化程序使用的 AI Native API。

本方案采用“分布式业务授权、统一协议与客户端契约”的架构：

- 不新增统一代理所有业务请求的鉴权网关；
- 不提供可以调用所有企业系统的万能 Token；
- 每个业务系统独立签发仅面向本系统的短期 Token；
- 每个业务系统负责用户、租户、资源和操作级最终授权；
- OpenCLI 统一登录命令、身份选择、Token 存储、错误格式和 OpenAPI 扩展；
- 个人数据使用用户委托身份；
- 非个人数据使用独立 Agent/工作负载身份；
- 传统 OAuth Client Credentials、AK/SK、API Key 只作为兼容机制，不作为所有 Agent 共享的身份。

核心原则是：

> CLI 是 Agent 的能力接口，不是最终安全边界；Token 证明主体与委托，业务系统根据可信主体决定可访问的数据和可执行的操作。

## 2. 背景与问题

传统 API 常见以下认证方式：

- 固定 Bearer Token；
- OAuth 2.0 Client Credentials；
- AK/SK 请求签名；
- API Key；
- 自定义认证 Header；
- 由调用方传入员工号、用户 ID 或租户 ID。

这些方式主要解决“哪个应用或凭据发起请求”，通常不能完整回答：

1. 哪个 Agent 实例发起了请求；
2. Agent 是否代表某个用户；
3. 用户具体同意了哪些权限；
4. 当前用户是否可以访问目标资源；
5. 高风险写操作是否获得了本次明确确认；
6. 凭据泄漏后能否只撤销单个 Agent 或单个授权会话。

把传统凭据直接编译进可下载 CLI 会产生以下问题：

- 二进制中的 `client_secret`、AK/SK 或 API Key 可以被提取或直接滥用；
- 多个 Agent 共享凭据，无法独立撤销和审计；
- 应用身份可能被错误用于读取个人数据；
- 仅隐藏 `userId` 参数不能阻止绕过 CLI 直接调用 API；
- 模型幻觉、提示注入和工具参数污染会扩大越权风险。

## 3. 第一性原则

### 3.1 认证、授权和委托是三件事

- 认证：确认调用主体是谁；
- 授权：判断该主体能否执行当前操作；
- 委托：确认 Agent 是否被允许代表某个用户执行操作。

完成登录不等于获得所有业务权限，持有有效 Token 也不等于可以访问任意资源。

### 3.2 一次 AI 调用可能包含两个主体

```text
调用主体：Agent、CLI 或工作负载
被代表主体：用户，可选
```

个人数据请求必须同时保留 Agent 和用户两条责任链。非个人数据请求可以没有用户主体，但必须有独立工作负载主体。

### 3.3 Public Client 不能保守共享秘密

分发到员工电脑、Agent 沙箱、容器、npm 包或安装包中的 CLI 属于 Public Client。

- `client_id` 可以公开并内置；
- 共享 `client_secret` 不能作为可信安全边界；
- 用户登录必须使用 Authorization Code + PKCE，Headless 环境可使用 Device Authorization；
- 需要应用身份时，每个工作负载应拥有独立凭据或不可导出的私钥。

### 3.4 Token 必须限定使用范围

Token 至少需要限制：

- 签发者；
- 目标系统；
- 主体；
- 客户端；
- 租户；
- scope；
- 有效期。

一个业务系统签发的 Token 不得被另一个业务系统接受。

### 3.5 最终授权必须在服务端执行

CLI 隐藏参数、Skill 约束和 Agent 提示属于减少误操作面的措施，不是可靠授权边界。

业务系统必须根据 Token 中的可信主体执行：

- 当前用户解析；
- 租户隔离；
- 资源所有权校验；
- 部门或组织数据范围校验；
- 操作级权限判断；
- 高风险确认校验。

## 4. 建设目标

### 4.1 目标

- 支持客户端 Agent 以个人身份访问本人或获授权范围内的数据；
- 支持 Agent 以工作负载身份访问非个人数据；
- 不要求存在统一业务请求网关；
- 不要求所有业务系统使用同一个 Token；
- 为不同业务系统提供一致的认证授权开发规范；
- 为 OpenCLI 提供可自动生成的认证和授权元数据；
- 支持单用户、单 Agent、单租户和单授权会话独立撤销；
- 对 AI 调用提供结构化错误、确认、审计和恢复指引；
- 兼容存量 OAuth2、AK/SK 和 API Key 接口。

### 4.2 非目标

第一阶段不实现：

- 一个 Token 调用所有企业系统；
- 统一代理所有业务流量；
- 由 CLI 代替业务系统做最终授权；
- 自动推断未在接口文档中声明的业务权限；
- 将共享 Secret 加密后编译进 CLI 并视为安全存储；
- 用自然语言提示代替服务端权限校验；
- 跨业务系统分布式事务。

## 5. 术语

| 术语 | 含义 |
| --- | --- |
| Agent | 使用 CLI 调用业务能力的 AI Agent 或自动化执行器 |
| CLI Client | 由 OpenCLI 生成的业务系统客户端 |
| Public Client | 无法可靠保存共享 Secret 的客户端 |
| Authorization Server | 完成用户授权并签发 Token 的服务 |
| Resource Server | 接受 Token 并提供业务 API 的服务 |
| User Delegation | Agent 获准代表某个用户执行限定操作 |
| Workload Identity | Agent、服务或运行实例自身的机器身份 |
| Access Token | 调用业务 API 的短期凭证 |
| Refresh Token | 用于刷新 Access Token 的长期凭证 |
| PKCE | 将 Authorization Code 绑定到发起登录的客户端实例 |
| Scope | Token 获准调用的能力范围 |
| Audience | Token 允许访问的目标业务系统 |
| PEP | 业务 API 中执行策略的授权检查点 |
| PDP | 根据主体、资源和上下文做授权决策的模块 |

## 6. 总体架构

```mermaid
flowchart LR
    U["用户"] --> AS["业务系统授权服务"]
    A["客户端 Agent"] --> CLI["生成式业务 CLI"]
    CLI --> AS
    AS --> TS["Token Service"]
    TS --> CLI

    CLI --> API["业务 API / PEP"]
    API --> PDP["业务权限 PDP"]
    PDP --> IAM["用户、组织、角色与数据权限"]
    PDP --> AUDIT["授权决策审计"]
    API --> DATA["业务数据"]

    CLI --> KC["OS Keychain / Credential Store"]
```

### 6.1 无网关架构的统一边界

本方案统一：

- OAuth/OIDC 协议；
- Agent 与用户身份模型；
- Token Claims；
- scope 命名；
- CLI 命令；
- OpenAPI 扩展；
- 错误契约；
- 测试和验收标准。

本方案不统一：

- Token 真值；
- Token audience；
- 业务权限数据；
- 业务系统授权决策；
- 用户在各业务系统中的资源范围。

### 6.2 业务系统部署选择

业务系统可以选择：

1. 接入已有企业身份平台，由身份平台签发本系统 audience 的 Token；
2. 在本系统部署独立授权服务；
3. 使用共享认证授权 SDK 或标准身份产品部署本系统授权能力。

即使没有统一登录，也必须遵守同一协议和开发规范。不得由各团队自行设计不兼容的密码登录和 Token 格式。

## 7. 信任边界与身份模型

### 7.1 用户委托身份

适用于：

- 查询本人考勤；
- 查询本人审批；
- 读取本人文档；
- 提交本人申请；
- 使用用户已有数据权限进行查询。

用户委托请求包含：

```json
{
  "workload": {
    "client_id": "finance-cli",
    "agent_id": "agent-instance-123"
  },
  "actor": {
    "type": "user",
    "subject": "user-10086"
  },
  "delegation": {
    "tenant_id": "company-a",
    "scopes": ["expense:read:self"],
    "expires_at": "2026-07-29T18:00:00+08:00"
  }
}
```

### 7.2 工作负载身份

适用于：

- 读取企业公共知识；
- 生成非个人统计报表；
- 定时同步；
- 系统健康检查；
- 获得组织授权的批处理任务。

工作负载 Token 的主体应为：

```text
sub=agent:finance-report-agent
```

不得伪装成某个用户，也不得使用一个所有 Agent 共享的全局身份。

### 7.3 双主体请求

高安全用户委托场景需要同时证明：

- 用户是谁；
- 哪个 CLI 获得授权；
- 哪个 Agent 实例正在使用授权；
- Agent 是否持有已登记的设备私钥。

推荐身份组合：

```text
sub       = 用户主体
azp       = CLI client_id
agent_id  = Agent 实例
cnf.jkt   = Agent 公钥指纹，可选
```

## 8. 用户授权：Authorization Code + PKCE

### 8.1 适用范围

Authorization Code + PKCE 是本地桌面 CLI 访问个人数据的默认流程。

如果业务系统还需要完成用户登录和身份确认，应使用 OIDC，在 scope 中包含 `openid`。

### 8.2 时序

```mermaid
sequenceDiagram
    participant U as 用户
    participant A as Agent
    participant C as 业务 CLI
    participant B as 系统浏览器
    participant AS as 授权服务
    participant K as OS Keychain
    participant API as 业务 API

    A->>C: 执行需要用户身份的命令
    C->>K: 查询用户凭据
    K-->>C: 未登录
    C->>C: 生成 verifier、challenge、state
    C->>C: 启动 loopback 随机端口
    C->>B: 打开授权 URL
    B->>AS: 用户登录并确认权限
    AS-->>C: code + state
    C->>C: 校验 state
    C->>AS: code + code_verifier
    AS-->>C: access_token + refresh_token
    C->>K: 保存 Token
    C->>API: Bearer access_token
    API->>API: 校验用户、租户、scope 和资源权限
    API-->>C: 业务结果
    C-->>A: 结构化输出
```

### 8.3 PKCE 要求

CLI 必须：

- 使用密码学安全随机源生成 `code_verifier`；
- `code_verifier` 长度为 43～128 个 URL-safe 字符；
- 使用 `S256` 计算 `code_challenge`；
- 每次授权生成新的 `state`；
- 只使用系统外部浏览器；
- 只监听 `127.0.0.1` 或 `::1`；
- 使用随机端口和固定回调路径；
- 完成回调后立即关闭监听器。

授权服务必须：

- 强制 `code_challenge_method=S256`；
- 禁止 `plain`；
- Authorization Code 一次性使用；
- Code 建议 2 分钟内过期；
- Code 绑定 `client_id`、`redirect_uri`、scope 和 PKCE challenge；
- Token Endpoint 不要求 Public Client 提供 `client_secret`。

### 8.4 授权请求

```http
GET /oauth/authorize
  ?response_type=code
  &client_id=finance-cli
  &redirect_uri=http://127.0.0.1:49152/oauth/callback
  &scope=openid%20profile%20expense:read:self
  &state=random-state
  &code_challenge=pkce-challenge
  &code_challenge_method=S256
```

授权页面必须展示：

- 业务系统名称；
- CLI/Agent 名称；
- 用户身份；
- 申请的具体权限；
- 数据范围；
- 授权有效期；
- 高风险权限提示。

### 8.5 Token 请求

```http
POST /oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code&
client_id=finance-cli&
code=one-time-code&
redirect_uri=http://127.0.0.1:49152/oauth/callback&
code_verifier=original-verifier
```

成功响应：

```json
{
  "access_token": "short-lived-token",
  "token_type": "Bearer",
  "expires_in": 900,
  "refresh_token": "opaque-refresh-token",
  "scope": "openid profile expense:read:self",
  "id_token": "oidc-id-token"
}
```

### 8.6 Headless Agent

不能启动浏览器或监听 loopback 的 Agent 应使用 Device Authorization Flow：

```bash
business-cli auth login --device
```

业务系统可选提供：

```http
POST /oauth/device_authorization
POST /oauth/token
```

Device Flow 和 PKCE Flow 必须签发相同语义、相同 audience、相同 scope 模型的用户 Token。

## 9. Agent/工作负载认证

### 9.1 推荐方案

工作负载认证优先级：

1. 工作负载身份联邦；
2. `private_key_jwt`；
3. mTLS；
4. 每个部署实例独立的 Client Credentials；
5. 每个 Agent 独立 AK/SK；
6. 每个 Agent 独立 API Key。

禁止所有 Agent 共享同一长期 Secret。

### 9.2 Agent Enrollment

需要精确识别 Agent 实例时，业务系统提供管理面 enrollment 能力：

1. Agent 首次运行生成密钥对；
2. 私钥保存到 Keychain、TPM 或系统证书存储；
3. 管理员使用一次性注册码批准注册；
4. 业务系统登记 `agent_id`、`client_id`、公钥、租户和 scope；
5. Agent 使用私钥证明身份；
6. 业务系统可以独立暂停或撤销 Agent。

示例登记数据：

```json
{
  "agent_id": "agent-instance-123",
  "client_id": "finance-agent",
  "tenant_id": "company-a",
  "public_key_thumbprint": "sha256-thumbprint",
  "allowed_scopes": ["report:read:company"],
  "status": "active"
}
```

### 9.3 过渡方案

业务系统暂时只支持 Client Credentials 时：

- 每个部署实例使用独立 `client_id`；
- Secret 由 Vault、Secret Manager、Kubernetes Secret 或 CI Secret 注入；
- Secret 不进入 OpenAPI、Skill、Git、普通 YAML 或生成二进制；
- Access Token 短期有效；
- 支持单实例撤销和轮换。

## 10. 授权服务接口规范

业务系统至少提供：

| 接口 | 必选 | 说明 |
| --- | --- | --- |
| `GET /.well-known/openid-configuration` | 是 | OIDC/OAuth 元数据发现 |
| `GET /oauth/authorize` | 是 | 浏览器登录与授权 |
| `POST /oauth/token` | 是 | Code 和 Refresh Token 换取 |
| `POST /oauth/revoke` | 是 | 登出与撤销 |
| `GET /oauth/jwks` | JWT 时必选 | Token 验签公钥 |
| `GET /oauth/userinfo` | OIDC 时建议 | 标准用户信息 |
| `GET /api/v1/me` | 建议 | 业务主体映射 |
| `POST /oauth/device_authorization` | 可选 | Headless Agent |

### 10.1 元数据示例

```json
{
  "issuer": "https://finance.example.com",
  "authorization_endpoint": "https://finance.example.com/oauth/authorize",
  "token_endpoint": "https://finance.example.com/oauth/token",
  "revocation_endpoint": "https://finance.example.com/oauth/revoke",
  "jwks_uri": "https://finance.example.com/oauth/jwks",
  "scopes_supported": [
    "openid",
    "profile",
    "expense:read:self",
    "expense:submit:self"
  ],
  "response_types_supported": ["code"],
  "grant_types_supported": ["authorization_code", "refresh_token"],
  "code_challenge_methods_supported": ["S256"]
}
```

### 10.2 Public CLI Client 注册

```yaml
client_id: finance-cli
client_type: public
grant_types:
  - authorization_code
  - refresh_token
redirect:
  type: loopback
  path: /oauth/callback
pkce:
  required: true
  methods: [S256]
allowed_scopes:
  - openid
  - profile
  - expense:read:self
  - expense:submit:self
```

## 11. Token 规范

### 11.1 Access Token

推荐使用短期签名 JWT：

- 有效期建议 5～15 分钟；
- 使用非对称签名；
- 通过 JWKS 发布公钥；
- 支持 `kid` 和密钥轮换；
- 不在 Token 中写入敏感业务数据；
- API 必须校验签名、`iss`、`aud`、`exp` 和 scope。

示例 Claims：

```json
{
  "iss": "https://finance.example.com",
  "sub": "user-10086",
  "aud": "finance-api",
  "azp": "finance-cli",
  "agent_id": "agent-instance-123",
  "tenant_id": "company-a",
  "scope": "expense:read:self",
  "auth_time": 1785310000,
  "iat": 1785310100,
  "exp": 1785311000,
  "jti": "token-uuid"
}
```

### 11.2 Refresh Token

Refresh Token 应为不可预测的 opaque 值：

- 只保存摘要或加密值；
- 每次刷新必须轮换；
- 旧 Token 立即失效；
- 检测旧 Token 复用；
- 复用时撤销整个授权会话；
- scope 不得在刷新时扩大；
- 用户禁用、离职或撤权后不得继续刷新。

### 11.3 Audience 隔离

```text
attendance-api Token 只能访问 attendance-api
finance-api Token 只能访问 finance-api
bpm-api Token 只能访问 bpm-api
```

业务系统不得接受：

- `aud` 缺失的 Token；
- audience 为其他系统的 Token；
- 通用 `aud=enterprise-all` Token。

## 12. 权限模型

### 12.1 Scope 命名

建议格式：

```text
<resource>:<action>:<data-range>
```

示例：

```text
expense:read:self
expense:read:department
expense:read:any
expense:submit:self
payment:approve:assigned
report:read:company
```

### 12.2 授权决策

```text
allow =
  token_valid
  AND audience_matches
  AND client_active
  AND agent_active
  AND scope_allows_operation
  AND tenant_matches
  AND resource_policy_allows_subject
  AND data_classification_allows_agent
  AND confirmation_policy_satisfied
```

### 12.3 资源级授权

`expense:read:self` 只允许读取 `owner_id == token.sub` 的资源。

`expense:read:department` 需要根据当前用户的组织关系计算部门范围。

`expense:read:any` 仍然受租户、数据分类和业务角色限制。

业务系统不得只验证 scope 后直接按调用方传入的资源 ID 返回数据。

## 13. 业务 API 设计规范

### 13.1 个人数据接口

优先：

```http
GET /api/v1/me/expenses
GET /api/v1/me/tasks
POST /api/v1/me/leave-requests
```

避免：

```http
GET /api/v1/expenses?employeeId=任意工号
```

兼容旧接口时：

- 服务端根据 `token.sub` 推导当前员工；
- 普通用户传入其他员工 ID 时返回 403；
- 允许跨用户查询的管理员必须持有更高数据范围 scope；
- 不允许 CLI 端的参数隐藏代替服务端校验。

### 13.2 身份参数

以下字段不能被普通调用方声明为可信身份：

- `userId`；
- `employeeId`；
- `tenantId`；
- `departmentId`；
- `agentId`；
- `role`。

服务端必须从可信 Token、用户目录或 Agent 注册信息中派生。

### 13.3 写操作

写操作必须：

- 支持幂等键；
- 明确资源版本或并发控制；
- 返回结构化变更结果；
- 对高风险操作要求确认；
- 执行后支持回读验证；
- 写入审计记录。

### 13.4 高风险操作

风险分级：

```text
read
write
high-risk-write
```

以下操作默认属于 `high-risk-write`：

- 删除；
- 审批；
- 支付；
- 权限变更；
- 批量操作；
- 不可逆状态变更；
- 导出大量敏感数据。

高风险操作不能因用户已经登录而自动执行。确认授权必须绑定：

- 用户；
- Agent；
- operation；
- resource；
- 请求参数摘要；
- 有效期；
- 一次性 nonce。

## 14. OpenAPI 开发规范

### 14.1 Security Scheme

```yaml
components:
  securitySchemes:
    userOAuth:
      type: oauth2
      flows:
        authorizationCode:
          authorizationUrl: https://finance.example.com/oauth/authorize
          tokenUrl: https://finance.example.com/oauth/token
          scopes:
            expense:read:self: 查看本人费用记录
            expense:submit:self: 提交本人报销
```

### 14.2 AI 授权扩展

每个业务 operation 应声明 `x-ai-access`：

```yaml
paths:
  /api/v1/me/expenses:
    get:
      operationId: listMyExpenses
      security:
        - userOAuth:
            - expense:read:self
      x-ai-access:
        subject_modes:
          - user
        audience: finance-api
        scopes:
          - expense:read:self
        tenant_bound: true
        resource_type: expense
        data_classification: personal
        risk: read
        confirmation: never
```

非个人数据：

```yaml
x-ai-access:
  subject_modes:
    - agent
    - user
  audience: finance-api
  scopes:
    - report:read:company
  tenant_bound: true
  data_classification: internal
  risk: read
  confirmation: never
```

高风险操作：

```yaml
x-ai-access:
  subject_modes:
    - user
  audience: finance-api
  scopes:
    - payment:approve:assigned
  tenant_bound: true
  resource_type: payment
  data_classification: confidential
  risk: high-risk-write
  confirmation: always
  idempotency_required: true
```

### 14.3 必填字段

`x-ai-access` 必须明确：

- `subject_modes`；
- `audience`；
- `scopes`；
- `tenant_bound`；
- `data_classification`；
- `risk`；
- `confirmation`。

写操作还必须明确：

- `idempotency_required`；
- 是否支持回读；
- 是否可恢复。

## 15. 生成 CLI 契约

所有业务 CLI 统一提供：

```bash
business-cli auth login
business-cli auth login --device
business-cli auth status
business-cli auth check --operation <operation>
business-cli auth logout
business-cli identity current
```

业务命令：

```bash
business-cli expense list --as user
business-cli report summary --as agent
business-cli payment approve --as user --id PAY-001 --dry-run
```

### 15.1 身份选择

- `--as user` 使用用户委托 Token；
- `--as agent` 使用工作负载 Token；
- operation 只允许一种主体时可以自动确定；
- operation 同时允许多种主体时必须显式选择；
- 用户认证失败后不得自动降级为 Agent 身份；
- Agent 认证失败后不得尝试其他用户的 Token。

### 15.2 Token 存储

Token 按以下维度隔离：

```text
issuer × client_id × tenant × user/agent × profile
```

存储优先级：

1. OS Keychain / Credential Manager；
2. 企业 Secret Manager；
3. 受控服务器凭据存储；
4. 禁止普通 YAML、命令行参数和生成二进制。

### 15.3 输出规范

- stdout 只输出结构化业务结果；
- stderr 输出进度和恢复提示；
- 不输出 Access Token、Refresh Token、Authorization Header；
- 日志对 Cookie、Secret、签名和敏感参数脱敏；
- `auth status` 只输出身份、scope、状态和过期时间。

## 16. 错误契约

业务 API 和 CLI 使用统一错误结构：

```json
{
  "error": {
    "type": "authorization",
    "subtype": "insufficient_scope",
    "message": "当前授权不能读取部门费用记录",
    "required_scopes": ["expense:read:department"],
    "request_id": "req-uuid",
    "retryable": false,
    "hint": "重新授权所需权限，或改为查询本人数据"
  }
}
```

标准错误类型：

| type | subtype | HTTP | 含义 |
| --- | --- | --- | --- |
| `authentication` | `login_required` | 401 | 未登录 |
| `authentication` | `token_expired` | 401 | Access Token 过期 |
| `authentication` | `invalid_token` | 401 | Token 无效 |
| `authorization` | `insufficient_scope` | 403 | scope 不足 |
| `authorization` | `resource_forbidden` | 403 | 资源级权限不足 |
| `authorization` | `tenant_mismatch` | 403 | 租户不匹配 |
| `authorization` | `agent_disabled` | 403 | Agent 已停用 |
| `consent` | `consent_required` | 403 | 需要重新授权 |
| `confirmation` | `confirmation_required` | 409 | 高风险确认 |
| `validation` | `identity_parameter_forbidden` | 400 | 禁止覆盖身份字段 |

错误响应不得包含 Token、用户敏感信息或服务端策略细节。

## 17. 安全开发红线

业务系统和 CLI 必须遵守：

1. 不将共享 `client_secret`、AK/SK 或 API Key 编译进可下载 CLI；
2. 不在 URL Query 中传输 Secret；
3. 不在日志中记录 Token、Authorization Header 或 Refresh Token；
4. 不使用 Implicit Grant；
5. 不使用 Resource Owner Password Grant；
6. Public Client 不依赖 `client_secret`；
7. PKCE 只允许 `S256`；
8. 不使用嵌入式 WebView 收集用户密码；
9. 不因处于企业内网而跳过 Token 和资源权限校验；
10. 不接受其他业务系统 audience 的 Token；
11. 不信任调用方传入的用户、租户、角色和 Agent 身份；
12. 不把 Skill 或 Agent 提示当成授权控制；
13. 不在用户认证失败后自动切换为高权限应用身份；
14. 不允许 Refresh Token 静默扩大 scope；
15. 不以“加密后与解密密钥一起分发”作为 Secret 安全方案。

## 18. 审计规范

每次业务调用至少记录：

```json
{
  "timestamp": "2026-07-29T10:00:00+08:00",
  "request_id": "req-uuid",
  "issuer": "https://finance.example.com",
  "subject": "user-10086",
  "client_id": "finance-cli",
  "agent_id": "agent-instance-123",
  "tenant_id": "company-a",
  "operation": "expense.list",
  "resource_type": "expense",
  "resource_id": null,
  "risk": "read",
  "decision": "allow",
  "policy_id": "expense-read-self-v2",
  "result": "success",
  "duration_ms": 82
}
```

审计日志不得记录：

- Token；
- Refresh Token；
- Client Secret；
- 私钥；
- 完整个人数据；
- 敏感请求 Body。

高风险操作还需记录：

- 用户确认 ID；
- 确认时间；
- 请求参数摘要；
- 写入结果；
- 回读验证结果。

## 19. 传统接口兼容

| 现有方式 | 过渡处理 | 目标处理 |
| --- | --- | --- |
| 固定个人 Token | Keychain 保存、短期使用 | Authorization Code + PKCE |
| OAuth Client Credentials | 每实例独立 Secret | `private_key_jwt` 或工作负载联邦 |
| AK/SK | 每 Agent 独立 AK/SK | 非对称工作负载身份 |
| API Key | 每 Agent 独立 Key、最小权限 | 短期工作负载 Token |
| 任意 employeeId | CLI 隐藏并服务端覆盖 | `/me` 资源接口 |
| 自定义认证 Header | 适配器注入 | 标准 Bearer/PoP Token |

兼容阶段仍必须满足：

- 每个 Agent 独立凭据；
- 可撤销；
- 可轮换；
- 可审计；
- 资源级授权；
- Secret 不进入分发包。

## 20. 共享开发资产

为了避免每个业务团队重复实现不一致的 OAuth 和权限逻辑，应建设共享开发资产，但不部署统一业务网关：

- 授权服务参考实现或 Starter；
- JWT 验签和 Claims 校验中间件；
- 用户主体解析接口；
- scope 和数据范围授权 SDK；
- 审计 SDK；
- OpenAPI `x-ai-access` Linter；
- OAuth/PKCE 协议一致性测试；
- OpenCLI 认证 Provider；
- 安全配置基线；
- 示例业务系统。

共享 SDK 可以统一实现协议，但最终策略和业务数据权限仍由各系统配置和执行。

## 21. 测试规范

### 21.1 协议测试

- PKCE S256 成功；
- 缺少 PKCE 被拒绝；
- `plain` 被拒绝；
- Authorization Code 重放被拒绝；
- `state` 不匹配被 CLI 拒绝；
- redirect URI 不匹配被拒绝；
- Code 过期被拒绝；
- Public Client 无 `client_secret` 仍可完成授权；
- Refresh Token 轮换成功；
- 旧 Refresh Token 复用触发会话撤销；
- revoke 后不能继续刷新。

### 21.2 Token 测试

- 签名错误；
- `iss` 错误；
- `aud` 错误；
- Token 过期；
- scope 缺失；
- 租户不匹配；
- Client 被停用；
- Agent 被停用；
- 密钥轮换期间新旧 `kid` 行为正确。

### 21.3 越权测试

- 普通用户读取他人数据返回 403；
- 修改 query 中的 `employeeId` 不能越权；
- 修改 Body 中的 `userId` 不能越权；
- 修改 Header 中的 `tenantId` 不能跨租户；
- 直接绕过 CLI 调用 API 仍不能越权；
- Agent 身份不能调用只允许用户的 operation；
- 用户身份不能自动获得工作负载的组织级权限。

### 21.4 AI Agent 测试

- Token 不出现在 stdout、stderr 和模型上下文；
- 缺少权限时返回可解析的 `required_scopes`；
- 认证失败不自动切换身份；
- 高风险操作必须先 dry-run 和确认；
- 提示注入不能修改身份、租户或授权 Header；
- Skill 中不包含真实 Secret；
- Agent 只能调用 OpenAPI 声明允许的主体模式。

## 22. 分阶段实施

### 阶段一：个人数据最小闭环

- 业务系统提供 OIDC/OAuth Authorization Code + PKCE；
- 注册 Public CLI Client；
- 提供 `auth login/status/logout`；
- Token 保存到 Keychain；
- 个人数据接口使用 `/me` 或服务端主体覆盖；
- 完成 `self` 数据范围授权；
- 接入审计。

### 阶段二：OpenCLI 标准化

- 解析 authorizationCode Security Scheme；
- 生成 PKCE 和 Device Flow 客户端；
- 增加 `x-ai-access`；
- 生成 `--as user|agent`；
- 统一结构化错误；
- 增加 OpenAPI Linter 和协议测试。

### 阶段三：工作负载身份

- 建设 Agent Enrollment；
- 支持 `private_key_jwt` 或 mTLS；
- 每个 Agent 独立身份和撤销；
- 支持 DPoP 或等价 Token 绑定；
- 淘汰共享 Client Secret。

### 阶段四：高风险治理

- operation 风险分级；
- 一次性确认授权；
- 幂等和回读验证；
- 风险审计和告警；
- 批量与不可逆操作治理。

### 阶段五：存量认证收敛

- AK/SK、API Key 改为每 Agent 独立；
- 固定 Token 迁移为短期 Token；
- 任意用户参数迁移为主体派生；
- 删除分发包中的共享 Secret；
- 统一安全基线。

## 23. 团队职责

| 团队 | 职责 |
| --- | --- |
| 业务系统团队 | 资源级授权、接口改造、scope、审计、风险分级 |
| 身份平台/授权服务团队 | 用户认证、授权页、Token、撤销、密钥轮换 |
| OpenCLI 团队 | 登录客户端、Provider、Keychain、OpenAPI 生成、错误契约 |
| Agent Runtime 团队 | 工具 allowlist、Agent 身份、确认交互、Secret 隔离 |
| 安全团队 | 协议基线、威胁模型、渗透测试、审计规则 |
| 测试团队 | 协议、越权、跨租户、Agent 提示注入测试 |

## 24. 验收标准

业务系统达到 AI Native 认证授权标准必须满足：

- 可以使用 Authorization Code + PKCE 完成用户授权；
- Public CLI Client 不需要 `client_secret`；
- Headless 场景有明确方案；
- Access Token 仅对当前业务系统有效；
- Access Token 短期有效；
- Refresh Token 支持轮换和撤销；
- Token 不进入 Skill、日志和模型上下文；
- 个人数据从 Token 主体派生用户范围；
- 直接绕过 CLI 不能读取其他用户数据；
- 支持用户和 Agent 身份显式区分；
- 认证失败不会自动切换高权限身份；
- 每个 operation 声明 scope、主体模式、风险和数据分类；
- 高风险操作具有确认、幂等和审计机制；
- 通过协议、Token、越权和 AI Agent 测试；
- 传统凭据可以单 Agent 撤销和轮换。

## 25. 架构决策结论

1. 不建设统一业务请求网关；
2. 不建设万能 Token；
3. 各业务系统独立签发或接收本系统 audience 的 Token；
4. 用户个人数据使用 Authorization Code + PKCE；
5. Headless Agent 使用 Device Authorization；
6. 非个人数据使用独立工作负载身份；
7. 业务系统是最终授权边界；
8. CLI 统一客户端体验，但不保存共享业务 Secret；
9. OpenAPI 通过 `x-ai-access` 声明 AI 授权语义；
10. 采用共享 SDK、Linter 和一致性测试降低多系统重复建设成本。

该架构在不引入统一网关的前提下，实现了统一规范、独立授权、最小权限、可撤销和可审计，适合作为企业内部业务 API 向客户端 Agent 开放时的标准技术基线。

## 26. 标准依据

- OAuth 2.0 Security Best Current Practice：[RFC 9700](https://www.rfc-editor.org/info/rfc9700/)
- OAuth 2.0 for Native Apps：[RFC 8252](https://www.rfc-editor.org/info/rfc8252/)
- Proof Key for Code Exchange：[RFC 7636](https://www.rfc-editor.org/info/rfc7636/)
- OAuth 2.0 Device Authorization Grant：[RFC 8628](https://www.rfc-editor.org/info/rfc8628/)
- OAuth 2.0 Token Exchange：[RFC 8693](https://www.rfc-editor.org/info/rfc8693/)
- OAuth 2.0 Mutual TLS：[RFC 8705](https://www.rfc-editor.org/info/rfc8705/)
- JWT Client Authentication：[RFC 7523](https://www.rfc-editor.org/info/rfc7523/)
- Demonstrating Proof of Possession：[RFC 9449](https://www.rfc-editor.org/info/rfc9449/)
- Zero Trust Architecture：[NIST SP 800-207](https://csrc.nist.gov/pubs/sp/800/207/final)
- Cloud-Native Zero Trust Access Control：[NIST SP 800-207A](https://csrc.nist.gov/pubs/sp/800/207/a/final)
