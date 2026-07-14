# Generate Options API Design

## Problem

`internal/app/generate_command.go` exposes a chain of five generation functions. Each wrapper adds one parameter and delegates to the next function. Adding a generation setting therefore extends function names, positional argument lists, and the wrapper chain. Because most arguments are strings, call sites are also vulnerable to passing values in the wrong position.

## Decision

Replace the wrapper chain with one request type and one entry point:

```go
type GenerateOptions struct {
	Input      string
	MCPConfig  string
	Output     string
	Module     string
	AppName    string
	AppVersion string
	ConfigPath string
	SkillLang  string
	Auth       string
	Signer     string
	Target     string
}

func RunGenerate(opts GenerateOptions) error
```

All repository call sites will use named fields. The four intermediate wrapper functions and the old positional `RunGenerate` signature will be removed. No compatibility adapter is needed because `internal/app` cannot be imported outside this repository and all callers can be migrated atomically.

## Behavior

Generation behavior remains unchanged:

- Exactly one of `Input` and `MCPConfig` is required.
- Empty `Target` selects Go generation.
- Empty `SkillLang` selects English through the renderer's existing normalization.
- Configuration, app version, authentication, signer, parsing, planning, and rendering keep their current order and validation.
- The Cobra command maps flag values directly into `GenerateOptions`.

## Alternatives Rejected

- A single long positional function removes delegation but keeps ambiguous same-type arguments.
- Functional options add constructors and option functions without a current need for staged configuration or extensibility outside the repository.
- Deprecated wrappers preserve the maintenance problem and serve no external compatibility requirement.

## Testing

Use a compile-failing migration as the red test: convert one existing call to `GenerateOptions` before defining the type. Then add the type and canonical function, migrate every caller, and run formatting plus the complete Go test suite. Existing integration and command tests continue to verify generation defaults, targets, language, version, authentication, and signer behavior.
