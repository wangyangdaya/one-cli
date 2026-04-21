# OpenCLI Skills Verifier

An interactive AI agent for validating generated CLI skills from OpenAPI specifications.

## Features

- **Multi-executable support**: Configure multiple CLI commands to be allowed
- **Skill validation**: Automatically loads and validates SKILL.md files
- **Interactive REPL**: Chat with the AI agent to test CLI commands
- **Sandboxed execution**: Only allows configured CLI commands, no shell control tokens

## Setup

1. Install dependencies:
```bash
cd skills-verify
uv sync
```

2. Copy and configure environment variables:
```bash
cp .env.example .env
# Edit .env with your settings
```

3. Generate a CLI project to test:
```bash
cd ..
go run ./cmd/opencli generate \
  --input ./examples/openapi.json \
  --output ./tmp/openapi \
  --module github.com/acme/openapi-cli \
  --app openapi-cli
```

## Configuration

Edit `skills-verify/.env`:

### Required Settings

- `LLM_BASE_URL`: Your LLM endpoint URL (OpenAI-compatible)
- `LLM_API_KEY`: API key for the LLM endpoint
- `LLM_MODEL_NAME`: Model name (default: `gpt-4o-mini`)

### Optional Settings

- `LOG_LEVEL`: Logging level (default: `INFO`)
- `SKILLS_VERIFY_APP_DIR`: Generated app directory (default: `tmp/openapi`)
- `OPENCLI_BASE_URL`: Base URL for API calls (required by generated CLI)
- `ALLOWED_EXECUTABLES`: Comma-separated list of allowed CLI executables

### Multi-Executable Configuration

To allow multiple CLI commands, set `ALLOWED_EXECUTABLES`:

```bash
# Single executable (default)
ALLOWED_EXECUTABLES=openapi-cli

# Multiple executables
ALLOWED_EXECUTABLES=openapi-cli,petcli,usercli

# With spaces (will be trimmed)
ALLOWED_EXECUTABLES=openapi-cli, petcli, usercli
```

## Usage

Run the interactive verifier:

```bash
cd skills-verify
uv run python main.py
```

### Example Session

```
正在初始化 openapi skills verifier（请稍候）...
skills 目录: /path/to/tmp/openapi/skills
skills 已加载: 1 个
  - /path/to/tmp/openapi/skills/auth/SKILL.md
已就绪（thread_id=skills-verify）。q / quit / exit / 退出 结束。

用户: List all available auth commands

助手: Based on the auth skill, here are the available commands:

1. `openapi-cli auth login` - Login with email and password
2. `openapi-cli auth logout` - Logout current session
3. `openapi-cli auth me` - Get current user information
4. `openapi-cli auth refresh` - Refresh authentication token
5. `openapi-cli auth register` - Register a new user account

Would you like me to demonstrate any of these commands?

用户: Show me how to register a new user

助手: I'll demonstrate the register command:

[Agent executes: openapi-cli auth register --email "test@example.com" --password "secret123" --username "Test User"]

The register command requires:
- `email` (string, required): User email address
- `password` (string, required): User password
- `username` (string, required): User name

Example:
```bash
openapi-cli auth register \
  --email "user@example.com" \
  --password "secret123" \
  --username "John Doe"
```

用户: quit
再见。
```

## Architecture

### Components

1. **CliBackend** (`cli_backend.py`)
   - Extends `FilesystemBackend` from deepagents
   - Parses and validates CLI commands
   - Executes commands in a sandboxed environment
   - Supports multiple allowed executables

2. **Main Agent** (`main.py`)
   - Loads SKILL.md files from generated CLI project
   - Creates AI agent with CLI backend
   - Provides interactive REPL interface

### Security Features

- **Command whitelist**: Only configured executables are allowed
- **No shell control**: Blocks `;`, `&&`, `||`, `|`, `>`, etc.
- **Timeout protection**: Commands timeout after 120 seconds (configurable)
- **Path isolation**: Commands run in the app directory

## Development

### Running Tests

```bash
cd skills-verify
uv run pytest
```

### Code Style

```bash
cd skills-verify
uv run ruff check .
uv run ruff format .
```

## Troubleshooting

### "Skills directory not found"

Make sure you've generated a CLI project first:
```bash
go run ./cmd/opencli generate --input ./examples/openapi.json --output ./tmp/openapi --module github.com/acme/openapi-cli --app openapi-cli
```

### "Only these commands are allowed"

Check your `ALLOWED_EXECUTABLES` setting in `.env`. The command you're trying to run must be in the allowed list.

### "Command timed out"

Increase the timeout in `CliBackend.__init__()` or check if the command is hanging.

### LLM Connection Issues

- Verify `LLM_BASE_URL` is correct and accessible
- Check `LLM_API_KEY` is valid
- Ensure the model name in `LLM_MODEL_NAME` is supported by your endpoint

## License

MIT
