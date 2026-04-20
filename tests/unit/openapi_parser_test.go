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
