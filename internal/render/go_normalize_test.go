package render

import (
	"testing"

	"one-cli/internal/model"
)

func TestNormalizeGoAppAllocatesUniqueOperationNames(t *testing.T) {
	app := model.App{Groups: []model.Group{{
		Name: "imports",
		Operations: []model.Operation{{
			BodyMode: "simple-json",
			Parameters: []model.Parameter{
				{Name: "data", In: "query", Type: "string"},
				{Name: "user.id", In: "query", Type: "string"},
				{Name: "user-id", In: "query", Type: "string"},
			},
			BodyFields: []model.BodyField{
				{Name: "data", Type: "string"},
				{Name: "description", Type: "string"},
			},
			FileFields: []model.BodyField{{Name: "file", Type: "string", Format: "binary"}},
		}},
	}}}

	normalized := normalizeGoApp(app)
	op := normalized.Groups[0].Operations[0]
	if got := op.Parameters[0]; got.FieldName != "Data" || got.FlagName != "query-data" {
		t.Fatalf("reserved query parameter names = %+v", got)
	}
	if op.Parameters[1].FieldName == op.Parameters[2].FieldName || op.Parameters[1].FlagName == op.Parameters[2].FlagName {
		t.Fatalf("normalized parameter names must be unique: %+v", op.Parameters)
	}
	if got := op.BodyFields[0]; got.FieldName != "BodyData2" || got.FlagName != "" {
		t.Fatalf("colliding body field should fall back to --data: %+v", got)
	}
	if got := op.BodyFields[1]; got.FieldName != "Description" || got.FlagName != "description" {
		t.Fatalf("ordinary body field names = %+v", got)
	}
	if got := op.FileFields[0]; got.FieldName != "FileFieldFile" {
		t.Fatalf("file field names = %+v", got)
	}

	if app.Groups[0].Operations[0].Parameters[0].FieldName != "" {
		t.Fatal("normalization must not modify the planner model")
	}
}

func TestNormalizeGoAppReservesFileWithoutBinaryFields(t *testing.T) {
	app := model.App{Groups: []model.Group{{Operations: []model.Operation{{
		BodyMode:   model.BodyModeSimpleJSON,
		Parameters: []model.Parameter{{Name: "file", In: "query"}},
		BodyFields: []model.BodyField{{Name: "file", Type: "string"}},
	}}}}}

	op := normalizeGoApp(app).Groups[0].Operations[0]
	if got := op.Parameters[0].FlagName; got != "query-file" {
		t.Fatalf("query file flag = %q, want query-file", got)
	}
	if got := op.BodyFields[0].FlagName; got != "" {
		t.Fatalf("JSON body file flag = %q, want fallback to --data", got)
	}
}
