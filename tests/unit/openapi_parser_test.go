package unit_test

import (
	"testing"

	"one-cli/internal/openapi"
)

func TestParseDocumentNormalizesTagsOperationsParametersAndRequestBody(t *testing.T) {
	doc, err := openapi.Parse([]byte(`
openapi: 3.0.0
info:
  title: Demo API
  version: "1.0"
tags:
  - name: pet
    description: Pet operations
paths:
  /pets:
    get:
      tags: [pet]
      operationId: listPets
      parameters:
        - in: query
          name: limit
          required: false
          schema:
            type: integer
      responses:
        "200":
          description: ok
    post:
      tags: [pet]
      operationId: createPet
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
      responses:
        "201":
          description: created
`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if doc.Title != "Demo API" {
		t.Fatalf("title = %q want %q", doc.Title, "Demo API")
	}
	if doc.Version != "1.0" {
		t.Fatalf("version = %q want %q", doc.Version, "1.0")
	}
	if len(doc.Tags) != 1 || doc.Tags[0].Name != "pet" {
		t.Fatalf("unexpected tags: %+v", doc.Tags)
	}
	if len(doc.Operations) != 2 {
		t.Fatalf("operations = %d want 2", len(doc.Operations))
	}

	getOp := doc.Operations[0]
	if getOp.Method != "GET" || getOp.Path != "/pets" {
		t.Fatalf("unexpected get operation location: %+v", getOp)
	}
	if getOp.Tag != "pet" || getOp.OperationID != "listPets" {
		t.Fatalf("unexpected get operation metadata: %+v", getOp)
	}
	if len(getOp.Parameters) != 1 {
		t.Fatalf("parameters = %d want 1", len(getOp.Parameters))
	}
	if getOp.Parameters[0].In != "query" || getOp.Parameters[0].Name != "limit" || getOp.Parameters[0].Type != "integer" {
		t.Fatalf("unexpected parameter: %+v", getOp.Parameters[0])
	}

	postOp := doc.Operations[1]
	if !postOp.RequestBody.Required {
		t.Fatal("expected request body to be required")
	}
	if len(postOp.RequestBody.ContentTypes) != 1 || postOp.RequestBody.ContentTypes[0] != "application/json" {
		t.Fatalf("unexpected request body content types: %+v", postOp.RequestBody.ContentTypes)
	}
}

func TestParseDocumentCapturesSimpleAndComplexJSONBodies(t *testing.T) {
	doc, err := openapi.Parse([]byte(`
openapi: 3.0.0
info:
  title: Demo API
  version: "1.0"
paths:
  /login:
    post:
      operationId: login
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [email, password]
              properties:
                email:
                  type: string
                password:
                  type: string
                remember:
                  type: boolean
      responses:
        "200":
          description: ok
  /orders:
    post:
      operationId: createOrder
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                items:
                  type: array
                  items:
                    type: string
      responses:
        "200":
          description: ok
`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	login := doc.Operations[0].RequestBody
	if !login.HasJSONSchema {
		t.Fatal("expected login body to expose JSON schema")
	}
	if !login.IsSimpleJSON {
		t.Fatal("expected login body to be classified as simple JSON")
	}
	if len(login.JSONFields) != 3 {
		t.Fatalf("fields = %d want 3", len(login.JSONFields))
	}
	if !login.JSONFields[0].Required || login.JSONFields[0].Name != "email" || login.JSONFields[0].Type != "string" {
		t.Fatalf("unexpected first field: %+v", login.JSONFields[0])
	}

	order := doc.Operations[1].RequestBody
	if !order.HasJSONSchema {
		t.Fatal("expected order body to expose JSON schema")
	}
	if order.IsSimpleJSON {
		t.Fatal("expected array body to be classified as complex JSON")
	}
	if len(order.JSONFields) != 0 {
		t.Fatalf("complex body should not expose generated fields: %+v", order.JSONFields)
	}
}

func TestParseDocumentResolvesReferencedJSONBodySchema(t *testing.T) {
	doc, err := openapi.Parse([]byte(`
openapi: 3.0.0
info:
  title: Demo API
  version: "1.0"
paths:
  /login:
    post:
      operationId: login
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/LoginRequest'
      responses:
        "200":
          description: ok
components:
  schemas:
    LoginRequest:
      type: object
      required: [email, password]
      properties:
        email:
          type: string
        password:
          type: string
        remember:
          type: boolean
`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	body := doc.Operations[0].RequestBody
	if !body.IsSimpleJSON {
		t.Fatal("expected referenced schema to be classified as simple JSON")
	}
	if len(body.JSONFields) != 3 {
		t.Fatalf("fields = %d want 3", len(body.JSONFields))
	}
}

func TestParseDocumentResolvesReferencedParameters(t *testing.T) {
	doc, err := openapi.Parse([]byte(`
openapi: 3.0.0
info:
  title: Demo API
  version: "1.0"
paths:
  /reports:
    get:
      operationId: listReports
      parameters:
        - $ref: '#/components/parameters/AuthorizationHeader'
        - name: limit
          in: query
          required: false
          schema:
            type: integer
      responses:
        "200":
          description: ok
components:
  parameters:
    AuthorizationHeader:
      name: Authorization
      in: header
      required: true
      description: Bearer token
      schema:
        type: string
`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(doc.Operations) != 1 {
		t.Fatalf("operations = %d want 1", len(doc.Operations))
	}
	if len(doc.Operations[0].Parameters) != 2 {
		t.Fatalf("parameters = %d want 2", len(doc.Operations[0].Parameters))
	}

	header := doc.Operations[0].Parameters[0]
	if header.Name != "Authorization" || header.In != "header" || !header.Required || header.Type != "string" {
		t.Fatalf("unexpected referenced header parameter: %+v", header)
	}
	if header.Description != "Bearer token" {
		t.Fatalf("description = %q want %q", header.Description, "Bearer token")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Task 6.1: allOf 合并测试
// ──────────────────────────────────────────────────────────────────────────────

func TestParseDocumentAllOfMergesPropertiesCorrectly(t *testing.T) {
	doc, err := openapi.Parse([]byte(`
openapi: 3.0.0
info:
  title: AllOf API
  version: "1.0"
paths:
  /users:
    post:
      operationId: createUser
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              allOf:
                - type: object
                  required: [name]
                  properties:
                    name:
                      type: string
                - type: object
                  required: [age]
                  properties:
                    age:
                      type: integer
      responses:
        "201":
          description: created
`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	body := doc.Operations[0].RequestBody
	if !body.HasJSONSchema {
		t.Fatal("expected HasJSONSchema to be true")
	}
	if !body.IsSimpleJSON {
		t.Fatal("expected allOf merged schema to be classified as simple JSON")
	}
	if len(body.JSONFields) != 2 {
		t.Fatalf("fields = %d want 2", len(body.JSONFields))
	}

	// Fields should be sorted alphabetically: age, name
	if body.JSONFields[0].Name != "age" || body.JSONFields[0].Type != "integer" || !body.JSONFields[0].Required {
		t.Fatalf("unexpected first field: %+v", body.JSONFields[0])
	}
	if body.JSONFields[1].Name != "name" || body.JSONFields[1].Type != "string" || !body.JSONFields[1].Required {
		t.Fatalf("unexpected second field: %+v", body.JSONFields[1])
	}
}

func TestParseDocumentAllOfMergedSatisfiesSimpleJSON(t *testing.T) {
	doc, err := openapi.Parse([]byte(`
openapi: 3.0.0
info:
  title: AllOf Simple API
  version: "1.0"
paths:
  /items:
    post:
      operationId: createItem
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              allOf:
                - type: object
                  properties:
                    color:
                      type: string
                - type: object
                  properties:
                    weight:
                      type: number
                - type: object
                  properties:
                    active:
                      type: boolean
      responses:
        "200":
          description: ok
`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	body := doc.Operations[0].RequestBody
	if !body.IsSimpleJSON {
		t.Fatal("expected allOf merged result to satisfy Simple JSON")
	}
	if len(body.JSONFields) != 3 {
		t.Fatalf("fields = %d want 3", len(body.JSONFields))
	}
	// Sorted: active, color, weight
	if body.JSONFields[0].Name != "active" || body.JSONFields[0].Type != "boolean" {
		t.Fatalf("unexpected field[0]: %+v", body.JSONFields[0])
	}
	if body.JSONFields[1].Name != "color" || body.JSONFields[1].Type != "string" {
		t.Fatalf("unexpected field[1]: %+v", body.JSONFields[1])
	}
	if body.JSONFields[2].Name != "weight" || body.JSONFields[2].Type != "number" {
		t.Fatalf("unexpected field[2]: %+v", body.JSONFields[2])
	}
}

func TestParseDocumentAllOfWithRefResolvesBeforeMerging(t *testing.T) {
	doc, err := openapi.Parse([]byte(`
openapi: 3.0.0
info:
  title: AllOf Ref API
  version: "1.0"
paths:
  /accounts:
    post:
      operationId: createAccount
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              allOf:
                - $ref: '#/components/schemas/BasePerson'
                - type: object
                  required: [role]
                  properties:
                    role:
                      type: string
      responses:
        "201":
          description: created
components:
  schemas:
    BasePerson:
      type: object
      required: [name]
      properties:
        name:
          type: string
        email:
          type: string
`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	body := doc.Operations[0].RequestBody
	if !body.IsSimpleJSON {
		t.Fatal("expected allOf with $ref to be classified as simple JSON after merge")
	}
	if len(body.JSONFields) != 3 {
		t.Fatalf("fields = %d want 3", len(body.JSONFields))
	}
	// Sorted: email, name, role
	if body.JSONFields[0].Name != "email" {
		t.Fatalf("expected email, got %q", body.JSONFields[0].Name)
	}
	if body.JSONFields[1].Name != "name" || !body.JSONFields[1].Required {
		t.Fatalf("expected name (required), got %+v", body.JSONFields[1])
	}
	if body.JSONFields[2].Name != "role" || !body.JSONFields[2].Required {
		t.Fatalf("expected role (required), got %+v", body.JSONFields[2])
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Task 6.2: oneOf/anyOf 测试
// ──────────────────────────────────────────────────────────────────────────────

func TestParseDocumentTopLevelOneOfIsNotSimpleJSON(t *testing.T) {
	doc, err := openapi.Parse([]byte(`
openapi: 3.0.0
info:
  title: OneOf API
  version: "1.0"
paths:
  /events:
    post:
      operationId: createEvent
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              oneOf:
                - type: object
                  properties:
                    kind:
                      type: string
                - type: object
                  properties:
                    category:
                      type: string
      responses:
        "200":
          description: ok
`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	body := doc.Operations[0].RequestBody
	if body.IsSimpleJSON {
		t.Fatal("expected top-level oneOf to be classified as complex JSON")
	}
}

func TestParseDocumentTopLevelAnyOfIsNotSimpleJSON(t *testing.T) {
	doc, err := openapi.Parse([]byte(`
openapi: 3.0.0
info:
  title: AnyOf API
  version: "1.0"
paths:
  /events:
    post:
      operationId: createEvent
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              anyOf:
                - type: object
                  properties:
                    kind:
                      type: string
                - type: object
                  properties:
                    category:
                      type: string
      responses:
        "200":
          description: ok
`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	body := doc.Operations[0].RequestBody
	if body.IsSimpleJSON {
		t.Fatal("expected top-level anyOf to be classified as complex JSON")
	}
}

func TestParseDocumentPropertyWithOneOfIsNotSimpleJSON(t *testing.T) {
	doc, err := openapi.Parse([]byte(`
openapi: 3.0.0
info:
  title: Property OneOf API
  version: "1.0"
paths:
  /items:
    post:
      operationId: createItem
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                value:
                  oneOf:
                    - type: string
                    - type: integer
      responses:
        "200":
          description: ok
`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	body := doc.Operations[0].RequestBody
	if body.IsSimpleJSON {
		t.Fatal("expected property with oneOf to be classified as complex JSON")
	}
}

func TestParseDocumentWithOneOfAnyOfParsesSuccessfully(t *testing.T) {
	_, err := openapi.Parse([]byte(`
openapi: 3.0.0
info:
  title: Mixed Schema API
  version: "1.0"
paths:
  /a:
    post:
      operationId: opA
      requestBody:
        content:
          application/json:
            schema:
              oneOf:
                - type: object
                  properties:
                    x:
                      type: string
                - type: object
                  properties:
                    y:
                      type: integer
      responses:
        "200":
          description: ok
  /b:
    post:
      operationId: opB
      requestBody:
        content:
          application/json:
            schema:
              anyOf:
                - type: object
                  properties:
                    a:
                      type: string
                - type: object
                  properties:
                    b:
                      type: boolean
      responses:
        "200":
          description: ok
`))
	if err != nil {
		t.Fatalf("expected document with oneOf/anyOf to parse without error, got: %v", err)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Task 6.3: Swagger 2.0 解析测试
// ──────────────────────────────────────────────────────────────────────────────

func TestParseDocumentSwagger20ParsesCorrectly(t *testing.T) {
	doc, err := openapi.Parse([]byte(`
swagger: "2.0"
info:
  title: Swagger Pet API
  version: "2.0"
basePath: /v1
paths:
  /pets:
    get:
      operationId: listPets
      tags:
        - pets
      responses:
        200:
          description: ok
`))
	if err != nil {
		t.Fatalf("parse swagger 2.0: %v", err)
	}

	if doc.Title != "Swagger Pet API" {
		t.Fatalf("title = %q want %q", doc.Title, "Swagger Pet API")
	}
	if doc.Version != "2.0" {
		t.Fatalf("version = %q want %q", doc.Version, "2.0")
	}
	if len(doc.Operations) == 0 {
		t.Fatal("expected at least one operation from swagger 2.0 document")
	}
	if doc.Operations[0].OperationID != "listPets" {
		t.Fatalf("operationId = %q want listPets", doc.Operations[0].OperationID)
	}
}

func TestParseDocumentSwagger20DefinitionsAndParametersResolve(t *testing.T) {
	doc, err := openapi.Parse([]byte(`
swagger: "2.0"
info:
  title: Swagger Ref API
  version: "1.0"
basePath: /api
paths:
  /users:
    get:
      operationId: listUsers
      tags:
        - users
      parameters:
        - $ref: '#/parameters/LimitParam'
      responses:
        200:
          description: ok
parameters:
  LimitParam:
    name: limit
    in: query
`))
	if err != nil {
		t.Fatalf("parse swagger 2.0 with parameters ref: %v", err)
	}

	if len(doc.Operations) != 1 {
		t.Fatalf("operations = %d want 1", len(doc.Operations))
	}

	op := doc.Operations[0]
	if op.OperationID != "listUsers" {
		t.Fatalf("operationId = %q want listUsers", op.OperationID)
	}
	// The referenced parameter should be resolved
	if len(op.Parameters) != 1 {
		t.Fatalf("parameters = %d want 1", len(op.Parameters))
	}
	if op.Parameters[0].Name != "limit" {
		t.Fatalf("param name = %q want limit", op.Parameters[0].Name)
	}
	if op.Parameters[0].In != "query" {
		t.Fatalf("param in = %q want query", op.Parameters[0].In)
	}
}

func TestParseDocumentSwagger20BasePathAndOperations(t *testing.T) {
	doc, err := openapi.Parse([]byte(`
swagger: "2.0"
info:
  title: Swagger Path API
  version: "1.0"
basePath: /v2
paths:
  /orders:
    get:
      operationId: listOrders
      tags:
        - orders
      responses:
        200:
          description: ok
  /health:
    get:
      operationId: healthCheck
      responses:
        200:
          description: ok
`))
	if err != nil {
		t.Fatalf("parse swagger 2.0 paths: %v", err)
	}

	if len(doc.Operations) != 2 {
		t.Fatalf("operations = %d want 2", len(doc.Operations))
	}

	// Operations should be sorted by path: /health before /orders
	firstOp := doc.Operations[0]
	if firstOp.Method != "GET" {
		t.Fatalf("first op method = %q want GET", firstOp.Method)
	}
	if firstOp.OperationID != "healthCheck" {
		t.Fatalf("first op id = %q want healthCheck", firstOp.OperationID)
	}

	secondOp := doc.Operations[1]
	if secondOp.Method != "GET" {
		t.Fatalf("second op method = %q want GET", secondOp.Method)
	}
	if secondOp.OperationID != "listOrders" {
		t.Fatalf("second op id = %q want listOrders", secondOp.OperationID)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Task 6.4: OpenAPI 3.1 解析测试
// ──────────────────────────────────────────────────────────────────────────────

func TestParseDocumentOpenAPI31ParsesCorrectly(t *testing.T) {
	doc, err := openapi.Parse([]byte(`
openapi: "3.1.0"
info:
  title: OpenAPI 3.1 API
  version: "3.1"
paths:
  /items:
    get:
      operationId: listItems
      tags: [items]
      responses:
        "200":
          description: ok
`))
	if err != nil {
		t.Fatalf("parse openapi 3.1: %v", err)
	}

	if doc.Title != "OpenAPI 3.1 API" {
		t.Fatalf("title = %q want %q", doc.Title, "OpenAPI 3.1 API")
	}
	if doc.Version != "3.1" {
		t.Fatalf("version = %q want %q", doc.Version, "3.1")
	}
	if len(doc.Operations) != 1 {
		t.Fatalf("operations = %d want 1", len(doc.Operations))
	}
	if doc.Operations[0].OperationID != "listItems" {
		t.Fatalf("operationId = %q want listItems", doc.Operations[0].OperationID)
	}
}

func TestParseDocumentOpenAPI31TypeArrayNoCrash(t *testing.T) {
	// OpenAPI 3.1 supports JSON Schema 2020-12 where "type" can be an array
	_, err := openapi.Parse([]byte(`
openapi: "3.1.0"
info:
  title: Type Array API
  version: "1.0"
paths:
  /data:
    post:
      operationId: postData
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                value:
                  type:
                    - string
                    - "null"
      responses:
        "200":
          description: ok
`))
	if err != nil {
		t.Fatalf("expected openapi 3.1 type array to parse without error, got: %v", err)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Task 6.5: 错误处理测试
// ──────────────────────────────────────────────────────────────────────────────

func TestParseDocumentEmptyInputReturnsEmptyDocument(t *testing.T) {
	doc, err := openapi.Parse([]byte{})
	if err != nil {
		t.Fatalf("expected no error for empty input, got: %v", err)
	}
	if doc.Title != "" || doc.Version != "" {
		t.Fatalf("expected empty document, got title=%q version=%q", doc.Title, doc.Version)
	}
	if len(doc.Tags) != 0 || len(doc.Operations) != 0 {
		t.Fatalf("expected no tags or operations, got tags=%d ops=%d", len(doc.Tags), len(doc.Operations))
	}
}

func TestParseDocumentInvalidYAMLReturnsError(t *testing.T) {
	_, err := openapi.Parse([]byte(`{{{not valid yaml or json!!!`))
	if err == nil {
		t.Fatal("expected error for invalid YAML/JSON input")
	}
}

func TestParseDocumentValidYAMLButNotOpenAPIReturnsError(t *testing.T) {
	_, err := openapi.Parse([]byte(`
name: just a regular yaml file
items:
  - one
  - two
`))
	if err == nil {
		t.Fatal("expected error for valid YAML that is not an OpenAPI document")
	}
}

func TestParseDocumentInvalidRefReturnsErrorWithPath(t *testing.T) {
	_, err := openapi.Parse([]byte(`
openapi: 3.0.0
info:
  title: Bad Ref API
  version: "1.0"
paths:
  /test:
    post:
      operationId: testOp
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/DoesNotExist'
      responses:
        "200":
          description: ok
`))
	if err == nil {
		t.Fatal("expected error for invalid $ref reference")
	}
	errMsg := err.Error()
	if !contains(errMsg, "DoesNotExist") && !contains(errMsg, "components/schemas") {
		t.Fatalf("expected error to contain reference path info, got: %v", errMsg)
	}
}

// contains is a simple helper to check substring presence.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
