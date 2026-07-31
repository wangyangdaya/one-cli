# SQL MCP Server Demo

独立的 Python/uv MCP Server 示例，复用官方 MCP Python SDK 和 WrenEngine，
面向 PostgreSQL 提供只读 SQL 查询、语义模型发现、MRTR 审批和 MCP App。

## 能力边界

- 官方依赖：`mcp[cli]==2.0.0`，不自行实现 MCP 协议或传输。
- 双协议：同一个 Server 支持 MCP `2026-07-28` 和旧版 initialize
  handshake 客户端。
- 双传输：stdio 与 Streamable HTTP（`/mcp`）。
- 双查询入口：
  - Wren MDL 语义模型，例如 `FROM orders`。
  - PostgreSQL 原始表，例如 `FROM mcp_demo.orders`。
- MRTR：`run_sql` 在真正执行前通过 SDK `Resolve(Elicit(...))` 请求批准；
  拒绝或取消不会访问数据库。
- MCP Apps：`run_sql` 绑定 `ui://sql-explorer/index.html`，提供计划、
  校验、审批执行、结果排序和 CSV 复制。
- 只读约束：只接受单条 SELECT/CTE/EXPLAIN；拒绝 DML、DDL、COPY、
  事务和会话命令；数据库连接同时设置只读事务和查询超时。
- 有界结果：默认最多 1000 行，硬上限 10000 行，并明确返回
  `truncated`。

`mcp==2.0.0` 尚未提供 Tasks API，因此本 demo 没有自行模拟 Tasks。

## 目录

```text
sql-mcp-server/
├── app/sql-explorer.html
├── demo-wren/
├── migrations/001_demo.sql
├── src/sql_mcp_server/
├── tests/
├── pyproject.toml
└── uv.lock
```

## 1. 安装

需要 Python 3.11+、uv 和可访问的 PostgreSQL：

```bash
cd /Users/chery-90507455/Documents/workspace/one-cli/sql-mcp-server
UV_CACHE_DIR=/tmp/sql-mcp-uv-cache uv sync
cp .env.example .env
```

编辑 `.env`：

```dotenv
DATABASE_URL=postgresql://mcp_demo:change-me@127.0.0.1:5432/mcp_demo
MCP_REQUEST_STATE_KEY=replace-with-at-least-32-random-bytes
SQL_MCP_HOST=127.0.0.1
SQL_MCP_PORT=8080
```

可用 `openssl rand -hex 32` 生成 request-state key。不要提交 `.env`。

## 2. 初始化 demo 数据

先创建专用数据库和用户，再以具备建表权限的连接执行幂等 migration：

```bash
psql "$DATABASE_URL" -f migrations/001_demo.sql
```

Migration 只创建/更新 `mcp_demo` schema 下的 `customers` 与 `orders`，
不包含 DROP。

## 3. 启动

stdio：

```bash
UV_CACHE_DIR=/tmp/sql-mcp-uv-cache uv run sql-mcp-server --transport stdio
```

Streamable HTTP：

```bash
UV_CACHE_DIR=/tmp/sql-mcp-uv-cache uv run sql-mcp-server \
  --transport http \
  --host 127.0.0.1 \
  --port 8080
```

HTTP MCP endpoint 为 `http://127.0.0.1:8080/mcp`。

stdio 客户端配置示例：

```json
{
  "mcpServers": {
    "sql-demo": {
      "command": "uv",
      "args": [
        "--directory",
        "/Users/chery-90507455/Documents/workspace/one-cli/sql-mcp-server",
        "run",
        "sql-mcp-server",
        "--transport",
        "stdio"
      ],
      "env": {
        "DATABASE_URL": "${DATABASE_URL}",
        "MCP_REQUEST_STATE_KEY": "${MCP_REQUEST_STATE_KEY}"
      }
    }
  }
}
```

## Tools

| Tool | 作用 | 是否访问数据库 |
| --- | --- | --- |
| `list_models` | 列出 Wren 语义模型 | 否 |
| `describe_model` | 返回模型字段和关系 | 否 |
| `dry_plan` | 将语义 SQL 展开为 PostgreSQL SQL | 否 |
| `dry_run` | 在数据库端校验查询 | 是，不返回行 |
| `run_sql` | 审批后执行有界只读查询 | 是 |

语义 SQL：

```sql
SELECT id, status, total
FROM orders
ORDER BY id;
```

原始表 SQL：

```sql
SELECT id, status, total
FROM mcp_demo.orders
ORDER BY id;
```

## 测试

单元、MDL 编译、双协议、MRTR 和 App 测试：

```bash
UV_CACHE_DIR=/tmp/sql-mcp-uv-cache uv run pytest -m "not integration"
UV_CACHE_DIR=/tmp/sql-mcp-uv-cache uv run ruff check .
UV_CACHE_DIR=/tmp/sql-mcp-uv-cache uv run ruff format --check .
```

真实 PostgreSQL 集成测试仅使用显式测试连接串：

```bash
SQL_MCP_TEST_DATABASE_URL="$DATABASE_URL" \
  UV_CACHE_DIR=/tmp/sql-mcp-uv-cache \
  uv run pytest tests/test_postgres_integration.py -v
```

该测试会执行幂等 demo migration，然后验证 Wren 模型发现、dry-plan、
dry-run、真实查询、行数限制和截断标记。
