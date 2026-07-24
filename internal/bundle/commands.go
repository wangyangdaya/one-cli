package bundle

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type documentedCommand struct {
	Group   string
	Command string
}

func validateDocumentedCommands(project projectInfo) error {
	for _, groupDir := range project.Groups {
		source := filepath.Join(project.SkillsDir, groupDir)
		referencePath := filepath.Join(source, "references", "commands.md")
		reference, err := os.ReadFile(referencePath)
		if err != nil {
			return fmt.Errorf("group %q is missing references/commands.md: %w", groupDir, err)
		}

		canonical := extractDocumentedCommands(string(reference), project.AppName)
		allowed := make(map[documentedCommand]bool, len(canonical))
		for _, command := range canonical {
			allowed[command] = true
		}
		if len(allowed) == 0 {
			return fmt.Errorf("group %q references/commands.md contains no CLI commands", groupDir)
		}

		err = filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("refusing to package symbolic link %s", path)
			}
			if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
				return nil
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, command := range extractDocumentedCommands(string(content), project.AppName) {
				if allowed[command] {
					continue
				}
				return fmt.Errorf(
					"documented command %q is not present in references/commands.md for group %q",
					project.AppName+" "+command.Group+" "+command.Command,
					groupDir,
				)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func extractDocumentedCommands(content, appName string) []documentedCommand {
	if strings.TrimSpace(appName) == "" {
		return nil
	}
	pattern := `(?:\.\.?[/\\]bin[/\\])?` + regexp.QuoteMeta(appName) + `(?:\.exe)?\s+([A-Za-z0-9_-]+)\s+([A-Za-z0-9_-]+)`
	matches := regexp.MustCompile(pattern).FindAllStringSubmatch(content, -1)
	commands := make([]documentedCommand, 0, len(matches))
	for _, match := range matches {
		group := match[1]
		command := match[2]
		if group == "skills" || strings.HasPrefix(command, "-") {
			continue
		}
		commands = append(commands, documentedCommand{Group: group, Command: command})
	}
	return commands
}
