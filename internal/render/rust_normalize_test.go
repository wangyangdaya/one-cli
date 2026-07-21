package render

import (
	"testing"

	"one-cli/internal/model"
)

func TestNormalizeRustAppDoesNotModifyInput(t *testing.T) {
	app := model.App{
		Groups: []model.Group{{
			Name:        "orders",
			PackageName: "orders.api",
			Operations: []model.Operation{{
				Parameters: []model.Parameter{{
					Name:      "user.id",
					In:        "query",
					FieldName: "GoField",
					FlagName:  "go.flag",
				}},
				BodyFields: []model.BodyField{{
					Name:      "user.id",
					FieldName: "GoBodyField",
					FlagName:  "go.body.flag",
				}},
			}},
		}},
	}

	normalized := normalizeRustApp(app)

	group := app.Groups[0]
	if group.PackageName != "orders.api" {
		t.Fatalf("input package name = %q, want unchanged %q", group.PackageName, "orders.api")
	}
	parameter := group.Operations[0].Parameters[0]
	if parameter.FieldName != "GoField" || parameter.FlagName != "go.flag" {
		t.Fatalf("input parameter names = %q/%q, want unchanged", parameter.FieldName, parameter.FlagName)
	}
	bodyField := group.Operations[0].BodyFields[0]
	if bodyField.FieldName != "GoBodyField" || bodyField.FlagName != "go.body.flag" {
		t.Fatalf("input body field names = %q/%q, want unchanged", bodyField.FieldName, bodyField.FlagName)
	}

	normalizedGroup := normalized.Groups[0]
	if normalizedGroup.PackageName != "orders_api" {
		t.Fatalf("normalized package name = %q, want %q", normalizedGroup.PackageName, "orders_api")
	}
	normalizedParameter := normalizedGroup.Operations[0].Parameters[0]
	if normalizedParameter.FieldName != "user_id" || normalizedParameter.FlagName != "user-id" {
		t.Fatalf("normalized parameter names = %q/%q", normalizedParameter.FieldName, normalizedParameter.FlagName)
	}
	normalizedBodyField := normalizedGroup.Operations[0].BodyFields[0]
	if normalizedBodyField.FieldName != "body_user_id" || normalizedBodyField.FlagName != "body-user-id" {
		t.Fatalf("normalized body field names = %q/%q", normalizedBodyField.FieldName, normalizedBodyField.FlagName)
	}
}

func TestNormalizeRustAppReservesDataAndFileForTheirInputRoles(t *testing.T) {
	app := model.App{Groups: []model.Group{{Operations: []model.Operation{{
		BodyMode: model.BodyModeSimpleJSON,
		Parameters: []model.Parameter{
			{Name: "data", In: "query"},
			{Name: "file", In: "query"},
		},
		BodyFields: []model.BodyField{{Name: "data"}, {Name: "description"}},
		FileFields: []model.BodyField{{Name: "asset", Format: "binary"}},
	}}}}}

	op := normalizeRustApp(app).Groups[0].Operations[0]
	if op.Parameters[0].FlagName != "query-data" {
		t.Fatalf("query data flag = %q, want query-data", op.Parameters[0].FlagName)
	}
	if op.Parameters[1].FlagName != "query-file" {
		t.Fatalf("query file flag = %q, want query-file", op.Parameters[1].FlagName)
	}
	if op.BodyFields[0].FlagName != "" {
		t.Fatalf("body data flag = %q, want fallback to --data", op.BodyFields[0].FlagName)
	}
	if op.FileFields[0].FlagName != "file" {
		t.Fatalf("upload flag = %q, want file", op.FileFields[0].FlagName)
	}
}

func TestNormalizeRustAppReservesFileWithoutBinaryFields(t *testing.T) {
	app := model.App{Groups: []model.Group{{Operations: []model.Operation{{
		BodyMode:   model.BodyModeSimpleJSON,
		Parameters: []model.Parameter{{Name: "file", In: "query"}},
		BodyFields: []model.BodyField{{Name: "file", Type: "string"}},
	}}}}}

	op := normalizeRustApp(app).Groups[0].Operations[0]
	if got := op.Parameters[0].FlagName; got != "query-file" {
		t.Fatalf("query file flag = %q, want query-file", got)
	}
	if got := op.BodyFields[0].FlagName; got != "" {
		t.Fatalf("JSON body file flag = %q, want fallback to --data", got)
	}
}
