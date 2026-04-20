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

## Model Wiring

Set `SKILLS_VERIFY_MODEL` to an import path in `module:attribute` form before running:

```bash
export SKILLS_VERIFY_MODEL="your_package.your_module:your_model"
```

## Verification Prompt

Example:

```bash
cd skills-verify
uv run python main.py --prompt "Use the auth skill to inspect available auth commands"
```
