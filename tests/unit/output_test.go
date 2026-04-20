package unit_test

import (
	"encoding/json"
	"testing"

	"one-cli/internal/output"
)

func TestJSONSuccessEnvelope(t *testing.T) {
	out, err := output.JSONSuccess("one-leave get", "查询成功", map[string]string{"k": "v"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var envelope output.SuccessEnvelope
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("expected valid JSON, got %v", err)
	}

	if !envelope.OK {
		t.Fatal("expected ok=true")
	}
	if envelope.Command != "one-leave get" {
		t.Fatalf("unexpected command: %q", envelope.Command)
	}
	if envelope.Message != "查询成功" {
		t.Fatalf("unexpected message: %q", envelope.Message)
	}

	data, ok := envelope.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected data to decode as object, got %#v", envelope.Data)
	}
	if got := data["k"]; got != "v" {
		t.Fatalf("unexpected data value: %#v", got)
	}
}

func TestJSONErrorEnvelope(t *testing.T) {
	out, err := output.JSONError("one-leave request", "validation_error", "bad input")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var envelope output.ErrorEnvelope
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("expected valid JSON, got %v", err)
	}

	if envelope.OK {
		t.Fatal("expected ok=false")
	}
	if envelope.Command != "one-leave request" {
		t.Fatalf("unexpected command: %q", envelope.Command)
	}
	if envelope.Error.Code != "validation_error" {
		t.Fatalf("unexpected error code: %q", envelope.Error.Code)
	}
	if envelope.Error.Message != "bad input" {
		t.Fatalf("unexpected error message: %q", envelope.Error.Message)
	}
}

func TestPrettyTextNormalizesWhitespace(t *testing.T) {
	got := output.PrettyText("  hello world  \r\n\r\n  second line  \n\n\n")
	want := "hello world\n\nsecond line"

	if got != want {
		t.Fatalf("unexpected pretty text:\nwant: %q\ngot:  %q", want, got)
	}
}

func TestTableRendersAlignedColumns(t *testing.T) {
	got := output.Table(
		[]string{"Name", "Score"},
		[][]string{
			{"Alice", "10"},
			{"Bob", "7"},
		},
	)

	want := "Name  | Score\n----- | -----\nAlice | 10   \nBob   | 7    "
	if got != want {
		t.Fatalf("unexpected table output:\nwant:\n%s\ngot:\n%s", want, got)
	}
}
