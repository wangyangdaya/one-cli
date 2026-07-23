package bundle

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func copyTree(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to package symbolic link %s", path)
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		return copyFileAtomic(path, target, info.Mode().Perm())
	})
}

func copyGroup(source, destination, appName string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to package symbolic link %s", path)
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		refresh := filepath.ToSlash(rel) == "generation-report.md"
		if !refresh {
			if info, err := os.Stat(target); err == nil && info.Mode().IsRegular() {
				return nil
			}
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			return copyAdaptedFile(path, target, info.Mode().Perm(), appName, "../bin/")
		}
		return copyFileAtomic(path, target, info.Mode().Perm())
	})
}

func copyAdaptedFile(source, destination string, mode os.FileMode, appName, prefix string) error {
	content, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	adapted := removeBundledBinaryRequirement(string(content), appName)
	adapted = adaptCommandPaths(adapted, appName, prefix)
	configPrefix := "./config/runtime.yaml"
	if prefix == "../bin/" {
		configPrefix = "../config/runtime.yaml"
		adapted = strings.ReplaceAll(adapted, "../bin/"+appName+" skills list", "../bin/"+appName+" skills --skills-dir .. list")
		adapted = strings.ReplaceAll(adapted, "../bin/"+appName+" skills read", "../bin/"+appName+" skills --skills-dir .. read")
	}
	adapted = replaceBareReference(adapted, "config/runtime.yaml", configPrefix)
	content = []byte(adapted)
	return writeFileAtomic(destination, content, mode)
}

func copyFileAtomic(source, destination string, mode os.FileMode) error {
	content, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	return writeFileAtomic(destination, content, mode)
}

func writeFileAtomic(destination string, content []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(destination), ".opencli-package-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err := temp.Write(content); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Chmod(mode); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, destination)
}

func adaptCommandPaths(content, appName, prefix string) string {
	needle := appName + " "
	if strings.TrimSpace(appName) == "" || !strings.Contains(content, needle) {
		return content
	}
	var result strings.Builder
	for {
		index := strings.Index(content, needle)
		if index < 0 {
			result.WriteString(content)
			break
		}
		result.WriteString(content[:index])
		previousIsPath := index > 0 && (content[index-1] == '/' || content[index-1] == '\\' || isCommandTokenCharacter(content[index-1]))
		if previousIsPath {
			result.WriteString(appName)
		} else {
			result.WriteString(prefix)
			result.WriteString(appName)
		}
		result.WriteByte(' ')
		content = content[index+len(needle):]
	}
	return result.String()
}

func isCommandTokenCharacter(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9' || value == '_' || value == '-'
}

func removeBundledBinaryRequirement(content, appName string) string {
	lines := strings.Split(content, "\n")
	wantBins := fmt.Sprintf(`bins: ["%s"]`, appName)
	result := make([]string, 0, len(lines))
	for index := 0; index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) == "requires:" && index+1 < len(lines) && strings.TrimSpace(lines[index+1]) == wantBins {
			index++
			continue
		}
		result = append(result, lines[index])
	}
	return strings.Join(result, "\n")
}

func replaceBareReference(content, needle, replacement string) string {
	var result strings.Builder
	for {
		index := strings.Index(content, needle)
		if index < 0 {
			result.WriteString(content)
			break
		}
		result.WriteString(content[:index])
		if index > 0 && (content[index-1] == '/' || content[index-1] == '\\') {
			result.WriteString(needle)
		} else {
			result.WriteString(replacement)
		}
		content = content[index+len(needle):]
	}
	return result.String()
}
