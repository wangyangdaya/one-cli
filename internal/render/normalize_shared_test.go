package render

import (
	"testing"

	"one-cli/internal/model"
)

func TestNormalizeTargetsUseSharedKebabCaseFlags(t *testing.T) {
	app := model.App{Groups: []model.Group{{Operations: []model.Operation{
		{
			BodyMode: model.BodyModeSimpleJSON,
			Parameters: []model.Parameter{
				{Name: "petId", In: "path"},
				{Name: "URLValue", In: "query"},
				{Name: "customValue", PreferredFlagName: "CustomFlag", In: "query"},
			},
			BodyFields: []model.BodyField{{Name: "pageSize"}},
		},
		{
			BodyMode:   model.BodyModeFormURLEncoded,
			BodyFields: []model.BodyField{{Name: "URLValue"}},
		},
	}}}}

	for target, normalized := range map[string]model.App{
		"go":   normalizeGoApp(app),
		"rust": normalizeRustApp(app),
	} {
		operation := normalized.Groups[0].Operations[0]
		for index, want := range []string{"pet-id", "url-value", "CustomFlag"} {
			if got := operation.Parameters[index].FlagName; got != want {
				t.Fatalf("%s parameter %d flag = %q, want %q", target, index, got, want)
			}
		}
		if got := operation.BodyFields[0].FlagName; got != "page-size" {
			t.Fatalf("%s body flag = %q, want page-size", target, got)
		}
		if got := normalized.Groups[0].Operations[1].BodyFields[0].FlagName; got != "url-value" {
			t.Fatalf("%s form body flag = %q, want url-value", target, got)
		}
	}
}
