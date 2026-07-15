package app

import "testing"

func TestJSONEnabledIsScopedToRootCommand(t *testing.T) {
	jsonRoot := NewRootCommand()
	if err := jsonRoot.PersistentFlags().Set("json", "true"); err != nil {
		t.Fatal(err)
	}
	plainRoot := NewRootCommand()

	if !JSONEnabled(jsonRoot) {
		t.Fatal("JSON should be enabled for the root configured with --json")
	}
	if JSONEnabled(plainRoot) {
		t.Fatal("JSON should not leak between root commands")
	}
}
