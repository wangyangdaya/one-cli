# skills-verify

This project validates generated skill documents under `../tmp/openapi/skills`
by connecting them to a local `create_deep_agent(...)` runner and a restricted
CLI backend for the generated `openapi-cli`.

## Prerequisites

- `tmp/openapi` has already been generated
- the generated CLI is available from `tmp/openapi`
- Python 3.11+
- `uv`

## Install

```bash
cd skills-verify
uv sync
```

## Run

```bash
cd skills-verify
uv run python main.py --prompt "Use the auth skill to inspect available auth commands"
```
