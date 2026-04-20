# OpenAPI Skills Verifier Design

## Goal

Create a self-contained Python verification project at the repository root that uses `uv` for dependency management and validates that generated skill documents under `tmp/openapi/skills` can be consumed by a model agent created with `create_deep_agent(...)`.

The verifier is not part of the generated Go project. It is a standalone test harness for validating the generated `SKILL.md` files and the generated CLI together.

## Scope

The verifier project will:

- live in a new root-level directory
- be implemented as an isolated Python project managed by `uv`
- define its own backend implementation in local source files
- point the agent at `tmp/openapi/skills`
- execute the generated CLI from `tmp/openapi`
- provide a single runnable entrypoint for prompt-based validation

The verifier project will not:

- import or depend on Python files from `examples/`
- rename or restructure generated Go output
- attempt to generalize into a production agent framework
- add unrelated features such as persistence, tracing pipelines, or web APIs

## Directory Layout

The new project will be created as a root-level folder named `skills-verify`.

Expected structure:

```text
skills-verify/
├── README.md
├── main.py
├── clibackend.py
└── pyproject.toml
```

## Agent Design

`main.py` will build a `create_deep_agent(...)` instance that is configured specifically for validating the generated OpenAPI skills.

The agent configuration will:

- use a locally configured model import
- use a locally defined backend class from `clibackend.py`
- set `skills` to the generated `tmp/openapi/skills` directory
- accept a prompt from the command line
- execute one verification run and print the result to stdout

The system prompt will instruct the agent to:

- prefer the provided skills for API-related tasks
- use the available CLI execution backend rather than invent commands
- stay within the generated CLI capabilities

## Backend Design

`clibackend.py` will define a class named `CliBackend`.

Responsibilities:

- expose the backend interface expected by `create_deep_agent(...)`
- restrict command execution to the generated `openapi-cli` binary
- run commands with the working directory set to `tmp/openapi`
- provide controlled stdout and stderr capture
- reject unsupported commands early with clear error messages

Behavioral constraints:

- only `openapi-cli ...` commands are allowed
- shell control operators such as `;`, `&&`, `||`, and `|` are rejected
- command execution uses subprocess APIs directly rather than a shell
- the backend adds the generated CLI location to `PATH` so the executable can be resolved consistently

This keeps the verification loop narrow: the agent can consume skills and call the generated CLI, but cannot execute arbitrary shell commands.

## Skills Integration

The verifier treats `tmp/openapi/skills/*/SKILL.md` as the source of truth for model guidance.

The expected validation path is:

1. `opencli generate` produces `tmp/openapi/skills`
2. the Python verifier points `create_deep_agent(...)` at that skills directory
3. the model reads the generated skill descriptions
4. the model chooses generated CLI commands based on those skills
5. `CliBackend` executes the generated CLI and returns structured output

This validates the intended contract: generated skills are meant for model consumption, not direct execution on their own.

## Configuration and Dependencies

The project will use `uv` with a `pyproject.toml`.

The Python dependency set should stay minimal and only include packages required to:

- run `create_deep_agent(...)`
- support the backend protocol used by the agent framework

Any model-specific import will be left explicit in `main.py` so the user can wire it to their local environment if needed.

## Verification Flow

The expected run sequence is:

1. generate the Go project into `tmp/openapi`
2. ensure the generated CLI is built or runnable in that directory
3. install Python dependencies with `uv`
4. run the verifier with a prompt such as asking which auth commands are available
5. confirm that the agent uses the generated skills and calls the generated CLI through `CliBackend`

## Error Handling

The verifier should fail clearly when:

- `tmp/openapi/skills` does not exist
- the generated CLI executable is missing
- the agent framework packages are unavailable
- the prompt is empty
- the agent attempts to execute a command outside the allowed CLI

Errors should be direct and operational, so the verifier is useful for troubleshooting generation output.

## Testing Strategy

This work is a runnable verification harness, not a new product subsystem.

Validation evidence should come from:

- successful dependency resolution for the Python project
- running the verifier entrypoint
- confirming the agent can see generated skills under `tmp/openapi/skills`
- confirming the backend only allows generated CLI commands

## Implementation Notes

- keep the code ASCII-only
- keep the project self-contained under `skills-verify`
- use a Pythonic class name `CliBackend`
- do not import the historical backend implementation from `examples/`
- use that file only as a behavioral reference while implementing the new local backend
