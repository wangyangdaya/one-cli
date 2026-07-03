---
name: swagger-config
description: Create or improve OpenCLI config YAML from Swagger/OpenAPI documents before generating a CLI. Use when the user wants to inspect an API spec, shorten generated command names, add tag/operation aliases, choose body modes, document auth mode such as ak_sk, or prepare an opencli.yaml/supplier.opencli.yaml file for later opencli generate.
---

# Swagger Config

Use this skill to turn a Swagger/OpenAPI document into an `opencli.yaml` config that makes the generated CLI usable.

## Workflow

1. Inspect the source document.
   - Prefer `opencli inspect --input <swagger-or-openapi-file>` when available.
   - For JSON, use structured tools such as `jq` to list `path`, `method`, `operationId`, `tags`, `summary`, and `description`.
   - Do not rely on raw string search when a parser is available.

2. Identify naming problems.
   - Long generated names such as `post-api-apply-v2-get-suppliermrpmonth` usually mean the source document has no `operationId`.
   - Non-business tags such as `api-apply` should become a domain group through `naming.tag_alias`.
   - Use business terms from `summary` or `description`; keep names lowercase kebab-case.

3. Write the config YAML.
   - Use `naming.tag_alias` for command groups.
   - Use `naming.operation_alias` for command names.
   - `operation_alias` keys may be `operationId`, `METHOD /path`, or `/path`.
   - Prefer `METHOD /path` for path-based aliases because it is unambiguous.
   - Use `overrides.body_mode` only when the default body mode is wrong.
   - If the Swagger export only contains a raw JSON example but the source UI shows body field requiredness or descriptions, preserve that information in `overrides.body_fields`.
   - Do not infer requiredness from examples alone. Examples prove shape and order, not required semantics.
   - For repeated body shapes, define one YAML anchor and reuse it across aliases instead of copying fields.

4. Include auth guidance when relevant.
   - For token CLIs, leave auth mode unchanged unless the user asks otherwise.
   - For AK/SK APIs, set `auth.type: ak_sk` and choose a signer profile, for example `auth.signer.profile: supplier_edi`; do not put secrets in YAML.
   - The generate command can override YAML with `--auth ak_sk --signer supplier_edi`.
   - Runtime credentials belong in `OPENCLI_AK` and `OPENCLI_SK`.

5. Validate with generation.
   - Run `opencli generate --input <spec> --config <config> --output <tmp-dir> --module <module> --app <app>`.
   - Add `--auth ak_sk --signer <profile>` only when overriding config auth.
   - Inspect generated command help or command files to confirm aliases took effect.
   - Inspect generated `skills/<group>/SKILL.md` to confirm body fields show the expected required labels.
   - Build or test the generated project when feasible.

## Config Shape

```yaml
app:
  binary: supplier-cli
  root_command: supplier-cli

auth:
  type: ak_sk
  signer:
    profile: supplier_edi
    algorithm: sha512_hex
    headers:
      access_key: appKey
      signature: sign
      timestamp: timestamp
      nonce: nonce
    path:
      strip_prefix: /api-apply
    body:
      order: spec
    canonical:
      template: "method={method}&path={path}&appKey={access_key}&appSecret={secret_key}&timestamp={timestamp}&nonce={nonce}&jsonBody={json_body}"

naming:
  tag_alias:
    "计划物流.": supplier
  operation_alias:
    "POST /api-apply/v2/get/supplierMrpMonth": mrp-month
    "/api-apply/v2/get/supplierPo": purchase-order

overrides:
  body_mode:
    supplier.mrp-month: file-or-data
  body_fields:
    supplier.kanban-delivery: &supplier_pull_body_fields
      - name: date
        required: true
        type: string
        description: 拉取日期，格式 yyyy-MM-dd
      - name: pageSize
        required: true
        type: integer
        description: 每次数量，最大 1000
    supplier.purchase-order: *supplier_pull_body_fields
```

## Body Field Rules

- In Swagger/OpenAPI JSON, required body fields belong in `requestBody.content.application/json.schema.required`.
- If the JSON only has `schema: {type: object}` plus `example`, treat field requiredness as unknown until a source UI, API owner, or config confirms it.
- Use `overrides.body_fields` to restore missing body metadata from Apipost/Swagger UI tables.
- Prefer keys in this order: `group.command` for readability, or `"METHOD /path"` when the command alias is not stable yet.
- For simple JSON bodies, `body_fields` controls generated flag docs such as `--date | 是`.
- For complex JSON bodies, `body_fields` documents top-level request body fields under Request Body Schema; the CLI still uses `--file` / `--data`.
- Mark only truly required fields with `required: true`. Do not write “nullable” unless the source schema explicitly allows `null`.

Repeated pull API example:

```yaml
overrides:
  body_fields:
    supplier.kanban-delivery: &supplier_pull_body_fields
      - name: date
        required: true
        type: string
        description: 拉取日期；格式 yyyy-MM-dd
      - name: pageSize
        required: true
        type: integer
        description: 每次数量，最大 1000
      - name: pageNum
        required: true
        type: integer
        description: 页码，从 1 开始
      - name: isForce
        required: true
        type: boolean
        description: 拉取状态
    supplier.purchase-order: *supplier_pull_body_fields
```

Complex push API example:

```yaml
overrides:
  body_fields:
    supplier.mrp-daily-risk-confirm: &supplier_push_body_fields
      - name: batchNo
        required: true
        type: string
        description: 批次号
      - name: total
        required: true
        type: integer
        description: 总数量
      - name: pageSize
        required: true
        type: integer
        description: 每页数量
      - name: pageNum
        required: true
        type: integer
        description: 页码，从 1 开始
      - name: list
        required: true
        type: array
        description: 明细列表
    supplier.purchase-order-risk-confirm: *supplier_push_body_fields
```

## Naming Rules

- Prefer nouns for read/query commands: `mrp-month`, `purchase-order`, `rdc-inventory`.
- Use verbs only when the API changes server state: `confirm`, `push`, `create`, `update`, `delete`.
- Keep groups stable and few: `supplier`, `mrp`, `inventory`, `purchase-order` are usually better than path segments.
- Avoid leaking transport details into user commands: skip `post`, `api`, `v2`, `get`, `apply` unless they carry business meaning.
- Preserve source paths in config keys so aliases remain traceable.

## Output Checklist

When done, report:

- The config file path.
- The command used to generate the CLI.
- A short before/after example of command names.
- Any unresolved naming or auth assumptions.
- Verification evidence, such as generated aliases found or build/test result.
