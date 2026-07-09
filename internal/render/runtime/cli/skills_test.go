package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSkillDescriptionParsesYAMLFrontmatter(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			"double quoted",
			"---\nname: leave\ndescription: \"leave commands for one\"\n---\n# Leave",
			"leave commands for one",
		},
		{
			"single quoted",
			"---\nname: leave\ndescription: 'leave commands for one'\n---\n# Leave",
			"leave commands for one",
		},
		{
			"plain scalar",
			"---\nname: leave\ndescription: leave commands for one\n---\n# Leave",
			"leave commands for one",
		},
		{
			"folded multiline",
			"---\nname: leave\ndescription: >\n  leave commands\n  for one\n---\n# Leave",
			"leave commands for one",
		},
		{
			"no frontmatter",
			"# Leave\nNo frontmatter here.",
			"",
		},
		{
			"empty description",
			"---\nname: leave\n---\n# Leave",
			"",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, tc.name+".md")
			if err := os.WriteFile(path, []byte(tc.body), 0o644); err != nil {
				t.Fatalf("write: %v", err)
			}
			if got := skillDescription(path); got != tc.want {
				t.Fatalf("skillDescription(%q) = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}
