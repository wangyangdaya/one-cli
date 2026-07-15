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
