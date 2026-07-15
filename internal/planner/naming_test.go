package planner

import (
	"reflect"
	"testing"
)

func TestFilterEmptySegmentsDoesNotModifyInput(t *testing.T) {
	values := []string{" first ", "", "second"}
	original := append([]string(nil), values...)

	got := filterEmptySegments(values)

	if !reflect.DeepEqual(values, original) {
		t.Fatalf("input = %#v, want unchanged %#v", values, original)
	}
	want := []string{"first", "second"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("filtered = %#v, want %#v", got, want)
	}
}
