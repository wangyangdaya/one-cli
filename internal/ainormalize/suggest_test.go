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

func TestStripCodeFences(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"plain json", `{"a":1}`, `{"a":1}`},
		{"json fence", "```json\n{\"a\":1}\n```", `{"a":1}`},
		{"bare fence", "```\n{\"a\":1}\n```", `{"a":1}`},
		{"json fence uppercase", "```JSON\n{\"a\":1}\n```", `{"a":1}`},
		{"leading whitespace", "  ```json\n{\"a\":1}\n```  ", `{"a":1}`},
		{"no closing fence", "```json\n{\"a\":1}", `{"a":1}`},
		{"preamble before fence", "Here is the JSON:\n```json\n{\"a\":1}\n```", `{"a":1}`},
		{"trailing commentary", "```json\n{\"a\":1}\n```\nNote: done", `{"a":1}`},
		{"jsonc variant", "```jsonc\n{\"a\":1}\n```", `{"a":1}`},
		{"json on fence line", "```json{\"a\":1}```", `{"a":1}`},
		{"preamble with brace", "Here is the {json}:\n```json\n{\"a\":1}\n```", `{"a":1}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := stripCodeFences(tc.input); got != tc.want {
				t.Fatalf("stripCodeFences(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestChatCompletionsEndpointAvoidsDoubleV1(t *testing.T) {
	cases := []struct {
		base string
		want string
	}{
		{"https://api.openai.com", "https://api.openai.com/v1/chat/completions"},
		{"https://api.openai.com/v1", "https://api.openai.com/v1/chat/completions"},
		{"https://gateway.example.com/v1/", "https://gateway.example.com/v1/chat/completions"},
		{"https://proxy.example.com/openai/v1/chat/completions", "https://proxy.example.com/openai/v1/chat/completions"},
	}
	for _, tc := range cases {
		t.Run(tc.base, func(t *testing.T) {
			got, err := chatCompletionsEndpoint(tc.base)
			if err != nil {
				t.Fatalf("chatCompletionsEndpoint(%q): %v", tc.base, err)
			}
			if got != tc.want {
				t.Fatalf("chatCompletionsEndpoint(%q) = %q, want %q", tc.base, got, tc.want)
			}
		})
	}
}

func TestValidateSuggestionRejectsSameGroupDuplicateViaPathOnlyKey(t *testing.T) {
	doc := openapi.Document{
		Operations: []openapi.Operation{
			{Method: "GET", Path: "/items", Tag: "catalog"},
			{Method: "POST", Path: "/items", Tag: "catalog"},
		},
	}
	inventory := BuildInventory(doc)
	suggestion := Suggestion{
		OperationAlias: map[string]string{
			"/items": "items",
		},
	}

	cfg, diagnostics := ValidateSuggestion(inventory, suggestion)
	if len(cfg.Naming.OperationAlias) != 0 {
		t.Fatalf("expected zero operation alias after same-group duplicate rejection, got %+v", cfg.Naming.OperationAlias)
	}
	if len(diagnostics.Rejected) != 1 || !strings.Contains(diagnostics.Rejected[0].Reason, "duplicate") {
		t.Fatalf("expected one duplicate rejection, got %+v", diagnostics.Rejected)
	}
}

func TestValidateSuggestionAllowsCrossGroupAliasViaPathOnlyKey(t *testing.T) {
	doc := openapi.Document{
		Operations: []openapi.Operation{
			{Method: "GET", Path: "/items", Tag: "catalog"},
			{Method: "POST", Path: "/items", Tag: "inventory"},
		},
	}
	inventory := BuildInventory(doc)
	suggestion := Suggestion{
		OperationAlias: map[string]string{
			"/items": "items",
		},
	}

	cfg, diagnostics := ValidateSuggestion(inventory, suggestion)
	if len(cfg.Naming.OperationAlias) != 1 {
		t.Fatalf("expected one operation alias, got %+v", cfg.Naming.OperationAlias)
	}
	if len(diagnostics.Rejected) != 0 {
		t.Fatalf("expected no rejections for cross-group alias, got %+v", diagnostics.Rejected)
	}
}

func TestSystemPromptIncludesDocumentCleaningGuidance(t *testing.T) {
	for _, want := range []string{
		"METHOD /path",
		"Chinese",
		"2-3 words",
		"operationId",
		"Do not infer schemas",
		"Do not infer auth",
		"api-apply",
		"securitySchemes",
	} {
		if !strings.Contains(systemPrompt, want) {
			t.Fatalf("system prompt missing %q:\n%s", want, systemPrompt)
		}
	}
}
