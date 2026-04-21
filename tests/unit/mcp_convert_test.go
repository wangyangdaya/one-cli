package unit_test

import (
	"testing"

	"one-cli/internal/configgen"
	"one-cli/internal/mcp"
	"one-cli/internal/planner"
)

func TestConvertToolExpandsSimpleObjectSchema(t *testing.T) {
	doc, err := mcp.ConvertTools("search", []mcp.Tool{
		{
			Name:        "web-search",
			Description: "Search the web",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string"},
					"limit": map[string]any{"type": "integer"},
				},
				"required": []any{"query"},
			},
		},
	})
	if err != nil {
		t.Fatalf("convert tools: %v", err)
	}

	plan, err := planner.Build(doc, configgen.Config{})
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}

	op := plan.Groups[0].Operations[0]
	if op.BodyMode != "simple-json" {
		t.Fatalf("body mode = %q", op.BodyMode)
	}
	if len(op.BodyFields) != 2 {
		t.Fatalf("body fields = %d", len(op.BodyFields))
	}
}

func TestConvertToolFallsBackForComplexSchema(t *testing.T) {
	doc, err := mcp.ConvertTools("search", []mcp.Tool{
		{
			Name: "advanced-search",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"filters": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "string"},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("convert tools: %v", err)
	}

	plan, err := planner.Build(doc, configgen.Config{})
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}

	if got := plan.Groups[0].Operations[0].BodyMode; got != "file-or-data" {
		t.Fatalf("body mode = %q", got)
	}
}
