# AI CLI Normalization Design

## Context

Some OpenAPI documents are technically valid but poor inputs for generated CLIs. The supplier document is a good example: operations are grouped under a Chinese business tag such as `计划物流.`, paths use platform-oriented names such as `/api-apply/v2/get/supplierMrpMonth`, and operation naming does not consistently express the task a CLI user wants to perform.

The generator already supports deterministic overrides through `opencli.yaml`, especially `naming.tag_alias` and `naming.operation_alias`. That makes a safe first AI integration possible: ask a model to suggest a config file that improves CLI names, then keep generation deterministic.

## Goals

- Add an AI-assisted path that turns an irregular OpenAPI document into an `opencli.yaml` naming suggestion.
- Keep the model output reviewable and editable before generation.
- Avoid silently rewriting source OpenAPI documents.
- Reuse the existing config surface instead of adding a parallel naming mechanism.
- Make generated command names short, stable, ASCII, and suitable for shells.

## Non-Goals

- Do not directly mutate the input OpenAPI file in the first version.
- Do not let model output bypass normal config parsing or planner validation.
- Do not require AI for normal generation.
- Do not infer schemas, response models, auth, or signing rules in this feature.
- Do not build a provider-specific SDK abstraction before one real model integration is working.

## User Experience

First version command:

```bash
opencli inspect --input ./examples/supplier.json --ai-suggest-config
```

Default behavior writes YAML to stdout:

```yaml
naming:
  tag_alias:
    "计划物流.": logistics
  operation_alias:
    "POST /api-apply/v2/get/supplierMrpMonth": mrp-month
    "POST /api-apply/v2/get/supplierPo": po
```

Optional output file:

```bash
opencli inspect --input ./examples/supplier.json --ai-suggest-config --output opencli.ai.yaml
opencli generate --input ./examples/supplier.json --config opencli.ai.yaml --output ./tmp/supplier --module github.com/acme/supplier --app supplier
```

The AI command should print diagnostics to stderr, including the number of operations analyzed, the model used, and any entries rejected during validation.

## AI Input

Send the model a compact operation inventory instead of the whole OpenAPI document:

```json
{
  "title": "supplier",
  "operations": [
    {
      "method": "POST",
      "path": "/api-apply/v2/get/supplierMrpMonth",
      "tag": "计划物流.",
      "operationId": "",
      "summary": "M+6月物料需求计划."
    }
  ]
}
```

Include only fields useful for naming:

- document title
- method
- path
- tag
- operationId
- summary
- description, when non-empty

Do not include request examples, response examples, credentials, headers, or environment values.

## AI Output Contract

The model must return JSON, not YAML, so the CLI can validate it before rendering:

```json
{
  "tag_alias": {
    "计划物流.": "logistics"
  },
  "operation_alias": {
    "POST /api-apply/v2/get/supplierMrpMonth": "mrp-month"
  }
}
```

Validation rules:

- every `tag_alias` key must match an observed tag
- every `operation_alias` key must match an observed operation alias key, preferably `METHOD path`
- aliases must be lowercase ASCII command identifiers using letters, numbers, and hyphens
- aliases must not be empty
- aliases within the same generated group must not collide
- rejected aliases are omitted and reported

After validation, render the result as normal `opencli.yaml`.

## Model Provider

Use explicit environment configuration for the first version:

```text
OPENCLI_AI_BASE_URL
OPENCLI_AI_API_KEY
OPENCLI_AI_MODEL
```

`OPENCLI_AI_BASE_URL` should support OpenAI-compatible chat completions. If any required value is missing, fail with a clear message explaining which environment variable is needed.

Keep the provider package small and internal. The rest of the app should depend on an interface such as:

```text
SuggestConfig(context) -> Suggestion
```

## Prompting

The system instruction should ask for CLI-friendly names, not translated prose:

- prefer concise English command identifiers
- preserve business distinctions from summaries
- remove transport noise such as `api`, version segments, `supplier`, `get`, and `push` when redundant
- keep verbs only when they clarify the action, such as `confirm-po`
- return only JSON matching the output schema

The command should use deterministic model settings where the provider supports them.

## Error Handling

- Missing AI configuration: fail before loading model request data.
- Provider request failure: return the provider status and a short sanitized error.
- Invalid JSON: show that the model response could not be parsed.
- Invalid aliases: keep valid entries, report rejected entries, and exit successfully if at least one valid suggestion remains.
- Empty valid result: fail and tell the user to inspect the document manually or retry with a different model.

Never call `generate` automatically from the AI suggestion command.

## Testing

Add focused tests for:

- operation inventory creation from a parsed OpenAPI document
- validation rejecting unknown operation keys
- validation rejecting non-CLI-safe aliases
- duplicate alias detection within a group
- rendering validated suggestions into `opencli.yaml`
- command behavior with a fake AI client

Do not add network tests. Provider integration should be covered by a fake HTTP server or fake client.

## Future Extensions

- `opencli generate --ai-suggest-config` could run the suggestion step into a temporary config and require `--yes` before using it.
- A later `--ai-normalize-openapi` mode may produce a patched OpenAPI document, but only after the config-suggestion flow proves useful.
- Provider presets can be added after the first integration demonstrates the needed compatibility surface.
