package unit_test

import (
	"testing"

	"one-cli/internal/configgen"
	"one-cli/internal/model"
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

	plan := planner.Build(doc, configgen.Config{})

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

func TestBuildUsesConfiguredAppVersion(t *testing.T) {
	doc := openapi.Document{
		Title: "Pet Store",
		Operations: []openapi.Operation{
			{Method: "GET", Path: "/pets", Tag: "pet", OperationID: "listPets"},
		},
	}

	plan := planner.Build(doc, configgen.Config{
		App: configgen.AppConfig{Version: "0.0.1"},
	})

	if plan.Version != "0.0.1" {
		t.Fatalf("app version = %q want 0.0.1", plan.Version)
	}
}

func TestBuildUsesTagAliasAndPathFallback(t *testing.T) {
	doc := openapi.Document{
		Operations: []openapi.Operation{
			{Method: "GET", Path: "/pets", Tag: "pet-store", OperationID: "getPets"},
			{Method: "POST", Path: "/billing/invoices", OperationID: ""},
		},
	}

	plan := planner.Build(doc, configgen.Config{
		Naming: configgen.NamingConfig{
			TagAlias: map[string]string{"pet-store": "pets"},
		},
	})

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

func TestBuildUsesOperationAliasByMethodPathAndPath(t *testing.T) {
	doc := openapi.Document{
		Operations: []openapi.Operation{
			{Method: "POST", Path: "/api-apply/v2/get/supplierMrpMonth", Tag: "计划物流."},
			{Method: "POST", Path: "/api-apply/v2/get/supplierPo", Tag: "计划物流."},
		},
	}

	plan := planner.Build(doc, configgen.Config{
		Naming: configgen.NamingConfig{
			TagAlias: map[string]string{
				"计划物流.": "supplier",
			},
			OperationAlias: map[string]string{
				"POST /api-apply/v2/get/supplierMrpMonth": "mrp-month",
				"/api-apply/v2/get/supplierPo":            "purchase-order",
			},
		},
	})

	if len(plan.Groups) != 1 {
		t.Fatalf("groups = %d want 1", len(plan.Groups))
	}
	if plan.Groups[0].Name != "supplier" {
		t.Fatalf("group = %q want supplier", plan.Groups[0].Name)
	}
	if got := plan.Groups[0].Operations[0].CommandName; got != "mrp-month" {
		t.Fatalf("method path alias command = %q want mrp-month", got)
	}
	if got := plan.Groups[0].Operations[1].CommandName; got != "purchase-order" {
		t.Fatalf("path alias command = %q want purchase-order", got)
	}
}

func TestBuildDerivesGroupNameFromMixedLanguageControllerTag(t *testing.T) {
	doc := openapi.Document{
		Operations: []openapi.Operation{
			{
				Method:      "POST",
				Path:        "/mri/current/clear",
				Tag:         "物料需求信息业务临时中间-TeMmMriCurrentController",
				OperationID: "clearUsingPOST",
			},
		},
	}

	plan := planner.Build(doc, configgen.Config{})

	if len(plan.Groups) != 1 {
		t.Fatalf("groups = %d want 1", len(plan.Groups))
	}
	if got := plan.Groups[0].Name; got != "te-mm-mri-current" {
		t.Fatalf("group name = %q want %q", got, "te-mm-mri-current")
	}
}

func TestBuildDerivesASCIIGroupNameFromNonASCIIBusinessTag(t *testing.T) {
	doc := openapi.Document{
		Tags: []openapi.Tag{
			{Name: "TMS-物料需求", Description: "Tms Mri Current Controller"},
			{Name: "TMS-配送单", Description: "Tms Sheet Controller"},
			{Name: "包装存储关系Api", Description: "Tr Bas Package Storage Controller"},
			{Name: "告警定时任务", Description: "Warning Schedule Controller"},
		},
		Operations: []openapi.Operation{
			{
				Method:      "GET",
				Path:        "/les/api/tms/mriCurrent/getMriCurrentlist",
				Tag:         "TMS-物料需求",
				OperationID: "getMriCurrentlistUsingGET",
			},
			{
				Method:      "POST",
				Path:        "/les/api/tms/sheet/updateTmsSheetList",
				Tag:         "TMS-配送单",
				OperationID: "updateTmsSheetListUsingPOST",
			},
			{
				Method:      "DELETE",
				Path:        "/les/api/trBasPackageStorage/deleteBatchIds",
				Tag:         "包装存储关系Api",
				OperationID: "deleteBatchIdsUsingDELETE_46",
			},
			{
				Method:      "POST",
				Path:        "/les/api/warningSchedule/doSheetWarningJob",
				Tag:         "告警定时任务",
				OperationID: "doSheetWarningJobUsingPOST",
			},
		},
	}

	plan := planner.Build(doc, configgen.Config{})

	if len(plan.Groups) != 4 {
		t.Fatalf("groups = %d want 4", len(plan.Groups))
	}
	for i, want := range []string{"tms-mri-current", "tms-sheet", "tr-bas-package-storage", "warning-schedule"} {
		if got := plan.Groups[i].Name; got != want {
			t.Fatalf("group[%d] name = %q want %q", i, got, want)
		}
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

	plan := planner.Build(doc, configgen.Config{
		Overrides: configgen.OverrideConfig{
			BodyMode: map[string]string{"custom.create": "flags"},
		},
	})

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
					JSONSchemaFields: []openapi.BodyField{
						{Name: "lineItems", Type: "array", Description: "order lines"},
						{Name: "note", Type: "string", Description: "order note"},
					},
				},
			},
		},
	}

	plan := planner.Build(doc, configgen.Config{})

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
	if len(order.BodySchemaFields) != 2 {
		t.Fatalf("complex body schema fields = %d want 2", len(order.BodySchemaFields))
	}
	if order.BodySchemaFields[0].Name != "lineItems" || order.BodySchemaFields[0].Description != "order lines" {
		t.Fatalf("unexpected body schema fields: %+v", order.BodySchemaFields)
	}
}

func TestBuildAppliesBodyFieldOverrides(t *testing.T) {
	required := true
	doc := openapi.Document{
		Operations: []openapi.Operation{
			{
				Method:      "POST",
				Path:        "/api-apply/v2/get/supplierDelState",
				Tag:         "计划物流.",
				OperationID: "POST /api-apply/v2/get/supplierDelState",
				RequestBody: openapi.RequestBody{
					ContentTypes:  []string{"application/json"},
					HasJSONSchema: true,
					IsSimpleJSON:  true,
					JSONFields: []openapi.BodyField{
						{Name: "date", RequiredUnknown: true, Type: "string"},
					},
					JSONSchemaFields: []openapi.BodyField{
						{Name: "date", RequiredUnknown: true, Type: "string"},
					},
				},
			},
		},
	}

	plan := planner.Build(doc, configgen.Config{
		Naming: configgen.NamingConfig{
			TagAlias:       map[string]string{"计划物流.": "supplier"},
			OperationAlias: map[string]string{"POST /api-apply/v2/get/supplierDelState": "kanban-delivery"},
		},
		Overrides: configgen.OverrideConfig{
			BodyFields: map[string][]configgen.BodyField{
				"supplier.kanban-delivery": {
					{Name: "date", Required: &required, Description: "拉取日期", Type: "string"},
				},
			},
		},
	})

	field := plan.Groups[0].Operations[0].BodyFields[0]
	if field.Name != "date" || !field.Required || field.RequiredUnknown || field.Description != "拉取日期" {
		t.Fatalf("unexpected body field override: %+v", field)
	}
}

func TestBuildAppliesBodyFieldOverridesByOriginalTagAfterReservedRename(t *testing.T) {
	required := true
	doc := openapi.Document{
		Operations: []openapi.Operation{
			{
				Method:      "POST",
				Path:        "/api-apply/v2/get/supplierDelState",
				Tag:         "skills",
				OperationID: "POST /api-apply/v2/get/supplierDelState",
				RequestBody: openapi.RequestBody{
					ContentTypes:  []string{"application/json"},
					HasJSONSchema: true,
					IsSimpleJSON:  true,
					JSONFields: []openapi.BodyField{
						{Name: "date", RequiredUnknown: true, Type: "string"},
					},
					JSONSchemaFields: []openapi.BodyField{
						{Name: "date", RequiredUnknown: true, Type: "string"},
					},
				},
			},
		},
	}

	plan := planner.Build(doc, configgen.Config{
		Naming: configgen.NamingConfig{
			OperationAlias: map[string]string{
				"POST /api-apply/v2/get/supplierDelState": "kanban-delivery",
			},
		},
		Overrides: configgen.OverrideConfig{
			BodyFields: map[string][]configgen.BodyField{
				"skills.kanban-delivery": {
					{Name: "date", Required: &required, Description: "拉取日期", Type: "string"},
				},
			},
		},
	})

	if len(plan.Groups) != 1 {
		t.Fatalf("groups = %d want 1", len(plan.Groups))
	}
	if plan.Groups[0].Name != "skills-api" {
		t.Fatalf("group name = %q want %q", plan.Groups[0].Name, "skills-api")
	}
	if got := plan.Groups[0].Operations[0].CommandName; got != "kanban-delivery" {
		t.Fatalf("command name = %q want %q", got, "kanban-delivery")
	}
	field := plan.Groups[0].Operations[0].BodyFields[0]
	if field.Name != "date" || !field.Required || field.RequiredUnknown || field.Description != "拉取日期" {
		t.Fatalf("body field override by original tag should still match after reserved rename: %+v", field)
	}
}

func TestBuildUsesMCPToolNameForCLICommand(t *testing.T) {
	doc := openapi.Document{
		Operations: []openapi.Operation{
			{Method: "MCP", Backend: "mcp-streamable-http", Path: "/quark_web_search", Tag: "tool-quark-web-search", OperationID: "quark_web_search"},
			{Method: "MCP", Backend: "mcp-streamable-http", Path: "/search_tool", Tag: "tool-search", OperationID: "search_tool"},
		},
	}

	plan := planner.Build(doc, configgen.Config{})

	if got := plan.Groups[0].Operations[0].CommandName; got != "quark-web-search" {
		t.Fatalf("first mcp command = %q want %q", got, "quark-web-search")
	}
	if got := plan.Groups[1].Operations[0].CommandName; got != "search-tool" {
		t.Fatalf("second mcp command = %q want %q", got, "search-tool")
	}
}

func TestBuildMakesCommandNamesIdentifierSafeAndUniquePerGroup(t *testing.T) {
	doc := openapi.Document{
		Operations: []openapi.Operation{
			{Method: "POST", Path: "/jobs/do-a", Tag: "jobs", OperationID: "doNormalOrgSheetUsingPOST"},
			{Method: "POST", Path: "/jobs/do-b", Tag: "jobs", OperationID: "doEmergeOrgSheetUsingPOST"},
			{Method: "GET", Path: "/jobs/{id}", Tag: "jobs", OperationID: "getInfoByIdUsingGET_3"},
			{Method: "GET", Path: "/jobs/pull", Tag: "jobs", OperationID: "getPartSupplPullUsingGET"},
			{Method: "GET", Path: "/jobs/storage", Tag: "jobs", OperationID: "getPartSupplStorageUsingGET"},
			{Method: "GET", Path: "/jobs/time", Tag: "jobs", OperationID: "getRecRequrieTimeUsingGET"},
			{Method: "GET", Path: "/jobs/download-kd", Tag: "jobs", OperationID: "downloadKDMriResultUsingGET"},
			{Method: "GET", Path: "/jobs/download-sp", Tag: "jobs", OperationID: "downloadSPMriResultUsingGET"},
			{Method: "POST", Path: "/jobs/import-kd", Tag: "jobs", OperationID: "importKDUsingPOST"},
		},
	}

	plan := planner.Build(doc, configgen.Config{})

	group := plan.Groups[0]
	wantCommands := []string{
		"normal",
		"emerge",
		"info-by-id",
		"pull",
		"storage",
		"requrie-time",
		"kd-result",
		"sp-result",
		"import-kd",
	}
	for i, want := range wantCommands {
		if got := group.Operations[i].CommandName; got != want {
			t.Fatalf("command %d = %q want %q", i, got, want)
		}
	}
}

func TestBuildRenamesReservedGroupName(t *testing.T) {
	doc := openapi.Document{
		Operations: []openapi.Operation{
			{Method: "GET", Path: "/skills/items", Tag: "skills", OperationID: "listSkills"},
			{Method: "GET", Path: "/orders", Tag: "orders", OperationID: "listOrders"},
		},
	}

	plan := planner.Build(doc, configgen.Config{})
	if len(plan.Groups) != 2 {
		t.Fatalf("groups = %d want 2", len(plan.Groups))
	}
	var skillsGroup model.Group
	for _, g := range plan.Groups {
		if g.RenamedFrom != "" {
			skillsGroup = g
		}
	}
	if skillsGroup.Name != "skills-api" {
		t.Fatalf("reserved group name = %q want %q", skillsGroup.Name, "skills-api")
	}
	if skillsGroup.RenamedFrom != "skills" {
		t.Fatalf("renamed-from = %q want %q", skillsGroup.RenamedFrom, "skills")
	}
	if skillsGroup.PackageName != "skills_api" {
		t.Fatalf("package name = %q want %q", skillsGroup.PackageName, "skills_api")
	}
}
