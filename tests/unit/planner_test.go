package unit_test

import (
	"testing"

	"one-cli/internal/configgen"
	"one-cli/internal/openapi"
	"one-cli/internal/planner"
)

func TestBuildGroupsOperationsAndNamesCommands(t *testing.T) {
	doc := openapi.Document{
		Operations: []openapi.Operation{
			{Method: "GET", Path: "/leaves", Tag: "leave", OperationID: "listLeaves"},
			{Method: "POST", Path: "/leaves/check", Tag: "leave", OperationID: "checkLeave"},
		},
	}

	plan, err := planner.Build(doc, configgen.Config{})
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}

	if len(plan.Groups) != 1 {
		t.Fatalf("groups = %d want 1", len(plan.Groups))
	}
	group := plan.Groups[0]
	if group.Name != "leave" {
		t.Fatalf("group name = %q want %q", group.Name, "leave")
	}
	if len(group.Operations) != 2 {
		t.Fatalf("operations = %d want 2", len(group.Operations))
	}
	if group.Operations[0].CommandName != "list" {
		t.Fatalf("first command = %q want %q", group.Operations[0].CommandName, "list")
	}
	if group.Operations[1].CommandName != "check" {
		t.Fatalf("second command = %q want %q", group.Operations[1].CommandName, "check")
	}
}

func TestBuildUsesTagAliasAndPathFallback(t *testing.T) {
	doc := openapi.Document{
		Operations: []openapi.Operation{
			{Method: "GET", Path: "/pets", Tag: "pet-store", OperationID: "getPets"},
			{Method: "POST", Path: "/billing/invoices", OperationID: ""},
		},
	}

	plan, err := planner.Build(doc, configgen.Config{
		Naming: configgen.NamingConfig{
			TagAlias: map[string]string{"pet-store": "pets"},
		},
	})
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}

	if len(plan.Groups) != 2 {
		t.Fatalf("groups = %d want 2", len(plan.Groups))
	}
	if plan.Groups[0].Name != "pets" {
		t.Fatalf("alias group = %q want %q", plan.Groups[0].Name, "pets")
	}
	if plan.Groups[0].Operations[0].CommandName != "get" {
		t.Fatalf("alias command = %q want %q", plan.Groups[0].Operations[0].CommandName, "get")
	}
	if plan.Groups[1].Name != "billing" {
		t.Fatalf("fallback group = %q want %q", plan.Groups[1].Name, "billing")
	}
	if plan.Groups[1].Operations[0].CommandName != "post-billing-invoices" {
		t.Fatalf("fallback command = %q want %q", plan.Groups[1].Operations[0].CommandName, "post-billing-invoices")
	}
}

func TestBuildChoosesBodyModeConservativelyAndHonorsOverrides(t *testing.T) {
	doc := openapi.Document{
		Operations: []openapi.Operation{
			{
				Method:      "POST",
				Path:        "/drafts",
				Tag:         "draft",
				OperationID: "createDraft",
				RequestBody: openapi.RequestBody{
					Required:     true,
					ContentTypes: []string{"application/json"},
				},
			},
			{
				Method:      "POST",
				Path:        "/uploads",
				Tag:         "upload",
				OperationID: "createUpload",
				RequestBody: openapi.RequestBody{
					Required:     true,
					ContentTypes: []string{"application/octet-stream"},
				},
			},
			{
				Method:      "POST",
				Path:        "/custom",
				Tag:         "custom",
				OperationID: "createCustom",
				RequestBody: openapi.RequestBody{
					Required:     true,
					ContentTypes: []string{"application/json"},
				},
			},
		},
	}

	plan, err := planner.Build(doc, configgen.Config{
		Overrides: configgen.OverrideConfig{
			BodyMode: map[string]string{"custom.create": "flags"},
		},
	})
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}

	if got := plan.Groups[0].Operations[0].BodyMode; got != "file-or-data" {
		t.Fatalf("default body mode = %q want %q", got, "file-or-data")
	}
	if got := plan.Groups[1].Operations[0].BodyMode; got != "file-or-data" {
		t.Fatalf("binary body mode = %q want %q", got, "file-or-data")
	}
	if got := plan.Groups[2].Operations[0].BodyMode; got != "flags" {
		t.Fatalf("override body mode = %q want %q", got, "flags")
	}
}

func TestBuildPropagatesSimpleJSONBodyFields(t *testing.T) {
	doc := openapi.Document{
		Operations: []openapi.Operation{
			{
				Method:      "POST",
				Path:        "/login",
				Tag:         "auth",
				OperationID: "login",
				RequestBody: openapi.RequestBody{
					Required:      true,
					ContentTypes:  []string{"application/json"},
					HasJSONSchema: true,
					IsSimpleJSON:  true,
					JSONFields: []openapi.BodyField{
						{Name: "email", Required: true, Type: "string"},
						{Name: "password", Required: true, Type: "string"},
						{Name: "remember", Required: false, Type: "boolean"},
					},
				},
			},
			{
				Method:      "POST",
				Path:        "/orders",
				Tag:         "order",
				OperationID: "createOrder",
				RequestBody: openapi.RequestBody{
					Required:      true,
					ContentTypes:  []string{"application/json"},
					HasJSONSchema: true,
					IsSimpleJSON:  false,
				},
			},
		},
	}

	plan, err := planner.Build(doc, configgen.Config{})
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}

	login := plan.Groups[0].Operations[0]
	if login.BodyMode != "simple-json" {
		t.Fatalf("login body mode = %q want %q", login.BodyMode, "simple-json")
	}
	if !login.BodyRequired {
		t.Fatal("expected login body to remain required")
	}
	if len(login.BodyFields) != 3 {
		t.Fatalf("simple body fields = %d want 3", len(login.BodyFields))
	}

	order := plan.Groups[1].Operations[0]
	if order.BodyMode != "file-or-data" {
		t.Fatalf("order body mode = %q want %q", order.BodyMode, "file-or-data")
	}
	if !order.BodyRequired {
		t.Fatal("expected order body to remain required")
	}
	if len(order.BodyFields) != 0 {
		t.Fatalf("complex body should not expose fields: %+v", order.BodyFields)
	}
}

func TestBuildUsesMCPToolNameForCLICommand(t *testing.T) {
	doc := openapi.Document{
		Operations: []openapi.Operation{
			{Method: "MCP", Backend: "mcp-streamable-http", Path: "/quark_web_search", Tag: "tool-quark-web-search", OperationID: "quark_web_search"},
			{Method: "MCP", Backend: "mcp-streamable-http", Path: "/search_tool", Tag: "tool-search", OperationID: "search_tool"},
		},
	}

	plan, err := planner.Build(doc, configgen.Config{})
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}

	if got := plan.Groups[0].Operations[0].CommandName; got != "quark-web-search" {
		t.Fatalf("first mcp command = %q want %q", got, "quark-web-search")
	}
	if got := plan.Groups[1].Operations[0].CommandName; got != "search-tool" {
		t.Fatalf("second mcp command = %q want %q", got, "search-tool")
	}
}
