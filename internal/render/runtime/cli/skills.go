package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
)

func NewSkillsCommand() *cobra.Command {
	var skillsDir string
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "List or read generated skill files from disk",
	}
	cmd.PersistentFlags().StringVar(&skillsDir, "skills-dir", "skills", "directory containing generated skill files")
	cmd.AddCommand(newSkillsListCommand(&skillsDir), newSkillsReadCommand(&skillsDir))
	return cmd
}

func newSkillsListCommand(skillsDir *string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List generated skills",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := skillsRoot(*skillsDir)
			if err != nil {
				return err
			}
			entries, err := os.ReadDir(root)
			if err != nil {
				return err
			}
			names := make([]string, 0, len(entries))
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				if _, err := os.Stat(filepath.Join(root, entry.Name(), "SKILL.md")); err == nil {
					names = append(names, entry.Name())
				}
			}
			sort.Strings(names)
			for _, name := range names {
				desc := skillDescription(filepath.Join(root, name, "SKILL.md"))
				if desc == "" {
					_, err := fmt.Fprintln(cmd.OutOrStdout(), name)
					if err != nil {
						return err
					}
				} else {
					if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", name, desc); err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
}

func newSkillsReadCommand(skillsDir *string) *cobra.Command {
	return &cobra.Command{
		Use:   "read <skill> [path]",
		Short: "Print a generated skill file",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := skillsRoot(*skillsDir)
			if err != nil {
				return err
			}
			skill := args[0]
			rel := "SKILL.md"
			if len(args) == 2 {
				rel = args[1]
			}
			path, err := safeSkillPath(root, skill, rel)
			if err != nil {
				return err
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(content)
			return err
		},
	}
}

func skillsRoot(dir string) (string, error) {
	root := strings.TrimSpace(dir)
	if root == "" {
		root = "skills"
	}
	if info, err := os.Stat(root); err == nil && info.IsDir() {
		return root, nil
	}
	return "", fmt.Errorf("skills directory %q not found; run from the generated project root or pass --skills-dir", root)
}

func safeSkillPath(root, skill, rel string) (string, error) {
	if skill == "" || strings.ContainsAny(skill, `/\`) {
		return "", fmt.Errorf("invalid skill %q", skill)
	}
	clean := filepath.Clean(rel)
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid skill file path %q", rel)
	}
	base := filepath.Join(root, skill)
	if info, err := os.Stat(filepath.Join(base, "SKILL.md")); err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("unknown skill %q", skill)
	}
	return filepath.Join(base, clean), nil
}

func skillDescription(filePath string) string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}
	trimmed := strings.TrimSpace(string(content))
	if !strings.HasPrefix(trimmed, "---") {
		return ""
	}
	body := strings.TrimPrefix(trimmed, "---")
	if idx := strings.Index(body, "\n---"); idx >= 0 {
		body = body[:idx]
	} else if i := strings.Index(body, "\n..."); i >= 0 {
		body = body[:i]
	}
	var meta struct {
		Description string `yaml:"description"`
	}
	if err := yaml.Unmarshal([]byte(body), &meta); err != nil {
		return ""
	}
	return strings.TrimSpace(meta.Description)
}
