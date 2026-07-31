# SQL MCP Server Demo Design

## Summary

Add an independent Python project named `sql-mcp-server/` at the repository
root, alongside `skills-verify/`. The project demonstrates a PostgreSQL-backed
MCP server using the stable official Python MCP SDK v2 and Wren's existing SQL
engine.

The demo must support both MCP protocol eras from one implementation:

- `2026-07-28`: stateless requests, `server/discover`, per-request metadata,
  multi-round-trip requests (MRTR), and MCP Apps.
- `2025-11-25` and earlier SDK-supported revisions: legacy initialization and
  session behavior provided by the SDK.

The project must not implement MCP framing, version negotiation, MRTR state
encoding, or extension negotiation itself. Those responsibilities belong to
the official `mcp==2.0.0` package.

## Goals

- Demonstrate MCP `2026-07-28` with a real local PostgreSQL database.
- Serve the same tools to new and legacy MCP clients.
- Support both Streamable HTTP and stdio transports.
- Reuse `wrenai[postgres]` for semantic SQL planning and execution.
- Demonstrate MRTR with explicit approval before query execution.
- Demonstrate an MCP App for entering SQL and inspecting results.
- Keep credentials, machine-specific paths, and database secrets out of Git.
- Keep the demo isolated from the one-cli Go module and `skills-verify/`.

## Non-Goals

- Building another MCP protocol implementation.
- Copying or modifying the WrenAI source tree.
- Supporting SQL writes in the first version.
- Implementing the Tasks extension before the official Python SDK supports it.
- Providing a production multi-tenant authorization system.
- Adding chart generation, NL2SQL generation, or semantic-memory indexing.
- Changing one-cli's existing MCP discovery or generated runtime behavior.

## Project Location

```text
one-cli/
├── skills-verify/
├── sql-mcp-server/
│   ├── src/sql_mcp_server/
│   ├── tests/
│   ├── app/
│   ├── migrations/
│   ├── pyproject.toml
│   ├── uv.lock
│   ├── .env.example
│   ├── .gitignore
│   └── README.md
├── internal/
└── tests/
```

`sql-mcp-server/` is an independent uv project with its own lock file, test
configuration, and runtime entry point.

## Dependencies

The project uses:

- Python 3.11 or newer.
- `mcp[cli]==2.0.0` for protocol negotiation, transports, MRTR, MCP Apps,
  request-state protection, and OpenTelemetry integration.
- `wrenai[postgres]` for WrenEngine, MDL processing, PostgreSQL connection
  handling, query planning, dry-run, and query execution.
- `pydantic-settings` or a small Pydantic settings model for environment-based
  application configuration.
- pytest for tests.

The project imports Wren's public engine and model APIs. It does not import or
copy Wren's existing v1-oriented `mcp_server.py`.

## Architecture

```text
MCP client
  │
  │ 2026-07-28 or a supported legacy revision
  ▼
Official MCP Python SDK v2
  ├── Streamable HTTP
  ├── stdio
  ├── version negotiation
  ├── MRTR and protected requestState
  └── MCP Apps extension
  │
  ▼
SQL MCP application layer
  ├── configuration
  ├── read-only SQL policy
  ├── tool handlers
  ├── result serialization
  └── UI resource registration
  │
  ▼
WrenEngine
  ├── MDL semantic planning
  ├── dry plan
  ├── PostgreSQL dry run
  └── PostgreSQL query
  │
  ▼
Local PostgreSQL demo schema
```

Protocol-version branching stays inside the official SDK. Tool handlers receive
one protocol-neutral request context and share one Wren-backed service.

## Components

### Configuration

Runtime configuration comes from environment variables. `.env.example`
documents values without containing working credentials.

Required configuration:

- PostgreSQL connection parameters, preferably through `DATABASE_URL`.
- Wren project or compiled MDL location.
- MRTR request-state signing key.

Optional configuration:

- HTTP host and port.
- default and maximum row limits.
- query timeout.
- log level.

The server fails at startup when required configuration is absent. Secrets are
never printed in errors or logs.

### Demo Database and MDL

The migration directory creates an isolated `mcp_demo` schema with small,
deterministic seed tables such as customers and orders. Migration scripts never
drop databases or unrelated schemas.

The bundled Wren project maps the seed tables into semantic models and defines
their relationship. Queries are written against Wren model names and are
expanded by WrenEngine before PostgreSQL execution.

### Wren Service

A focused service owns WrenEngine lifecycle and exposes:

- model listing and description;
- semantic SQL planning;
- database dry-run;
- bounded query execution.

Tool handlers depend on this service rather than on WrenEngine directly. This
keeps protocol behavior, query policy, and engine lifecycle independently
testable.

### Read-Only SQL Policy

The first version accepts one SQL statement and permits only query operations.
It rejects data modification, DDL, transaction control, session configuration,
and multiple statements before calling Wren or PostgreSQL.

The database user should also have read-only privileges. Application validation
is defense in depth, not a substitute for PostgreSQL permissions.

Every execution has:

- a default result limit of 1000 rows;
- a hard maximum of 10000 rows;
- a configurable execution timeout;
- truncation metadata in the result.

### MCP Tools

The server exposes:

- `list_models`: list Wren MDL models.
- `describe_model`: return model columns and relationships.
- `dry_plan`: expand semantic SQL to PostgreSQL SQL without database access.
- `dry_run`: validate a read-only query against PostgreSQL without returning
  rows.
- `run_sql`: request approval through MRTR, then execute and return structured
  rows.

Tools return typed structured output so the SDK publishes input and output
schemas.

### MRTR Approval

`run_sql` demonstrates MRTR:

1. Validate the SQL and requested row limit without database access.
2. Request explicit confirmation containing the SQL, target identity without
   credentials, and effective row limit.
3. Let the SDK protect and verify `requestState`.
4. On acceptance, resume the same tool and execute once.
5. On rejection, invalid state, expiry, or changed inputs, do not execute.

The request state is bound to the authenticated principal when available, tool
name, normalized arguments, and expiry. SDK facilities are used instead of a
custom state codec.

The same tool body serves both protocol eras through the SDK's compatibility
behavior.

### MCP App

The server registers an MCP App at `ui://sql-explorer/index.html` and associates
it with the SQL tools through SDK-provided Apps APIs.

The first App contains:

- SQL editor;
- row-limit control;
- Dry Plan and Dry Run actions;
- explicit Run action and confirmation state;
- structured result table;
- client-side sorting and pagination;
- copy-to-CSV action;
- clear error and truncation states.

The App is a static bundled artifact. It runs inside the host sandbox, declares
a restrictive CSP, does not load third-party scripts, and calls tools only
through the host bridge.

## Request Flows

### New Protocol Query

1. Client optionally calls `server/discover`.
2. Client calls `run_sql` with `2026-07-28` per-request metadata.
3. SDK validates protocol headers and capabilities.
4. Handler validates read-only SQL and returns an MRTR input request.
5. Client obtains approval and resumes with the protected request state.
6. Handler verifies the resumed request, calls WrenEngine, and returns typed
   rows.

No `initialize`, `notifications/initialized`, or `Mcp-Session-Id` is used on
this path.

### Legacy Query

1. Client performs the legacy SDK-managed initialization.
2. Client calls the same `run_sql` tool.
3. SDK maps the interaction to its legacy-compatible request flow.
4. The same policy and Wren service execute the query.

### MCP App Query

1. Host reads the `ui://` resource declared by the tool.
2. Host renders it in a sandboxed iframe.
3. The App invokes Dry Plan, Dry Run, or Run through the host bridge.
4. Tool results are passed back to the App as structured data.
5. Sorting, pagination, and CSV generation occur locally in the App.

## Error Handling

Errors are categorized for clear MCP responses:

- configuration errors fail startup;
- unsupported or unsafe SQL is an invalid-arguments error;
- approval rejection is a non-execution result, not a database failure;
- Wren planning failures identify the planning phase;
- PostgreSQL validation and execution failures identify the database phase;
- timeouts are reported distinctly;
- result serialization errors do not expose credentials or raw connection
  details.

Logs include a trace or request identifier and tool name. SQL text logging is
disabled by default because queries may contain sensitive literals.

## Testing

### Unit Tests

- settings validation and secret redaction;
- read-only SQL policy, including multiple-statement rejection;
- row-limit normalization;
- Arrow/PostgreSQL value serialization;
- MRTR rejection and tampered-state behavior;
- App metadata and resource registration.

### In-Memory MCP Tests

Use the SDK's in-memory client/server support to verify:

- tool and resource discovery;
- typed schemas;
- MRTR approval and rejection;
- one handler implementation across both supported protocol eras;
- App resource retrieval and tool association.

### PostgreSQL Integration Tests

Against the local test schema:

- migrations and seed data load successfully;
- `dry_plan`, `dry_run`, and `run_sql` work end to end;
- results are bounded and truncation is reported;
- write and multi-statement SQL never reach PostgreSQL;
- query timeout behavior is enforced.

Integration tests require an explicitly configured test database and must not
fall back to a developer's default database.

### Transport and Conformance Tests

- Streamable HTTP new client to new server.
- Streamable HTTP legacy client to dual-era server.
- stdio new client to new server.
- stdio legacy client to dual-era server.
- no legacy session header on the `2026-07-28` HTTP path.
- `server/discover`, required headers, and per-request metadata are handled by
  the SDK.
- MCP Inspector or the official conformance tooling can discover and call the
  demo.

## Verification Commands

The project will document and provide commands equivalent to:

```bash
cd sql-mcp-server
uv sync
uv run ruff check .
uv run ruff format --check .
uv run pytest
uv run sql-mcp-server --transport stdio
uv run sql-mcp-server --transport http
```

PostgreSQL integration tests run only when the explicit test database
configuration is present.

## Security Constraints

- No credentials or personal filesystem paths are committed.
- The database account is read-only and scoped to `mcp_demo`.
- SQL writes, DDL, multi-statements, and transaction control are rejected.
- Results, time, and request size are bounded.
- MRTR state uses SDK protection and expiration.
- The App uses a restrictive CSP and no third-party runtime assets.
- Tool descriptions, UI resources, SQL text, and database errors are treated as
  potentially sensitive.
- TLS verification is never disabled.

## Rollout

1. Scaffold the independent uv project and configuration.
2. Add deterministic PostgreSQL demo schema and Wren project.
3. Implement the read-only Wren service and tool handlers.
4. Add MRTR approval to `run_sql`.
5. Add the SQL Explorer MCP App.
6. Add unit, protocol-era, transport, and PostgreSQL integration tests.
7. Document setup and verify with an MCP client or Inspector.

Tasks, SQL writes, charts, OAuth deployment, and production tenancy remain
future extensions rather than partially implemented demo features.
