package ainormalize

import (
	"strings"
	"testing"

	"one-cli/internal/openapi"
)

func TestValidateSuggestionRendersConfigYAML(t *testing.T) {
	doc := openapi.Document{
		Title: "supplier",
		Operations: []openapi.Operation{
			{Method: "POST", Path: "/api-apply/v2/get/supplierMrpMonth", Tag: "计划物流.", Summary: "M+6月物料需求计划."},
			{Method: "POST", Path: "/api-apply/v2/get/supplierPo", Tag: "计划物流.", Summary: "采购订单."},
		},
	}
	inventory := BuildInventory(doc)
	suggestion := Suggestion{
		TagAlias: map[string]string{
			"计划物流.":       "logistics",
			"unknown-tag": "bad",
		},
		OperationAlias: map[string]string{
			"POST /api-apply/v2/get/supplierMrpMonth": "mrp-month",
			"POST /api-apply/v2/get/supplierPo":       "采购订单",
			"POST /missing":                           "missing",
		},
	}

	cfg, diagnostics := ValidateSuggestion(inventory, suggestion)
	out, err := RenderConfigYAML(cfg)
	if err != nil {
		t.Fatalf("render config yaml: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"naming:",
		"tag_alias:",
		`计划物流.: logistics`,
		"operation_alias:",
		`POST /api-apply/v2/get/supplierMrpMonth: mrp-month`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered YAML missing %q:\n%s", want, text)
		}
	}
	for _, unwanted := range []string{"unknown-tag", "采购订单", "POST /missing"} {
		if strings.Contains(text, unwanted) {
			t.Fatalf("rendered YAML should omit %q:\n%s", unwanted, text)
		}
	}
	if len(diagnostics.Rejected) != 3 {
		t.Fatalf("expected 3 rejected entries, got %+v", diagnostics.Rejected)
	}
}

func TestValidateSuggestionRejectsDuplicateOperationAliasesInGroup(t *testing.T) {
	doc := openapi.Document{
		Operations: []openapi.Operation{
			{Method: "POST", Path: "/api-apply/v2/get/supplierMrpMonth", Tag: "计划物流."},
			{Method: "POST", Path: "/api-apply/v2/get/supplierMrpDate", Tag: "计划物流."},
		},
	}
	inventory := BuildInventory(doc)
	suggestion := Suggestion{
		TagAlias: map[string]string{"计划物流.": "logistics"},
		OperationAlias: map[string]string{
			"POST /api-apply/v2/get/supplierMrpMonth": "mrp",
			"POST /api-apply/v2/get/supplierMrpDate":  "mrp",
		},
	}

	cfg, diagnostics := ValidateSuggestion(inventory, suggestion)
	if len(cfg.Naming.OperationAlias) != 1 {
		t.Fatalf("expected one operation alias after duplicate rejection, got %+v", cfg.Naming.OperationAlias)
	}
	if len(diagnostics.Rejected) != 1 || !strings.Contains(diagnostics.Rejected[0].Reason, "duplicate") {
		t.Fatalf("expected duplicate rejection, got %+v", diagnostics.Rejected)
	}
}
