package bundle

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

type projectInfo struct {
	AppName     string
	SkillsDir   string
	Groups      []string
	RootSkillMD string
}

var executeCommand = func(dir, name string, args ...string) error {
	command := exec.Command(name, args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w\n%s", name, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}

func resolveBinary(projectDir, appName, providedPath string) (string, func(), error) {
	if path := strings.TrimSpace(providedPath); path != "" {
		if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
			return "", func() {}, fmt.Errorf("binary %q is not a regular file", path)
		}
		return path, func() {}, nil
	}

	goProject := regularFileExists(filepath.Join(projectDir, "go.mod"))
	rustProject := regularFileExists(filepath.Join(projectDir, "Cargo.toml"))
	if goProject && rustProject {
		return "", func() {}, fmt.Errorf("generated project contains both go.mod and Cargo.toml; pass --binary explicitly")
	}
	if !goProject && !rustProject {
		return "", func() {}, fmt.Errorf("generated project contains neither go.mod nor Cargo.toml; pass --binary explicitly")
	}

	if goProject {
		tempDir, err := os.MkdirTemp("", "opencli-package-*")
		if err != nil {
			return "", func() {}, err
		}
		cleanup := func() { _ = os.RemoveAll(tempDir) }
		output := filepath.Join(tempDir, executableName(appName))
		if err := executeCommand(projectDir, "go", "build", "-o", output, "./cmd/"+appName); err != nil {
			cleanup()
			return "", func() {}, err
		}
		return output, cleanup, nil
	}

	packageName, err := readCargoPackageName(filepath.Join(projectDir, "Cargo.toml"))
	if err != nil {
		return "", func() {}, err
	}
	if err := executeCommand(projectDir, "cargo", "build", "--release"); err != nil {
		return "", func() {}, err
	}
	output := filepath.Join(projectDir, "target", "release", executableName(packageName))
	if !regularFileExists(output) {
		return "", func() {}, fmt.Errorf("Rust build completed but binary %q was not found", output)
	}
	return output, func() {}, nil
}

func readCargoPackageName(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	inPackage := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			inPackage = line == "[package]"
			continue
		}
		if !inPackage {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found || strings.TrimSpace(key) != "name" {
			continue
		}
		name := strings.Trim(strings.TrimSpace(value), "\"'")
		if name != "" {
			return name, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("Cargo.toml %q is missing [package].name", path)
}

func executableName(appName string) string {
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(appName), ".exe") {
		return appName + ".exe"
	}
	return appName
}

func regularFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func discoverProject(projectDir string) (projectInfo, error) {
	binDir := filepath.Join(projectDir, "bin")
	entries, err := os.ReadDir(binDir)
	if err != nil {
		return projectInfo{}, fmt.Errorf("read generated bin directory: %w", err)
	}
	launchers := make([]string, 0, 1)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || strings.HasSuffix(strings.ToLower(name), ".cmd") {
			continue
		}
		launchers = append(launchers, name)
	}
	if len(launchers) != 1 {
		return projectInfo{}, fmt.Errorf("generated project must contain exactly one CLI launcher in %s; found %d", binDir, len(launchers))
	}

	skillsDir := filepath.Join(projectDir, "skills")
	entries, err = os.ReadDir(skillsDir)
	if err != nil {
		return projectInfo{}, fmt.Errorf("read generated skills directory: %w", err)
	}
	groups := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if info, err := os.Stat(filepath.Join(skillsDir, entry.Name(), "SKILL.md")); err == nil && info.Mode().IsRegular() {
			groups = append(groups, entry.Name())
		}
	}
	if len(groups) == 0 {
		return projectInfo{}, fmt.Errorf("generated project contains no group Skills in %s", skillsDir)
	}
	if info, err := os.Stat(filepath.Join(skillsDir, "README.md")); err != nil || !info.Mode().IsRegular() {
		return projectInfo{}, fmt.Errorf("generated skills root is missing README.md")
	}
	rootSkillMD := filepath.Join(skillsDir, "SKILL.md")
	if !regularFileExists(rootSkillMD) {
		rootSkillMD = ""
	}
	sort.Strings(groups)
	return projectInfo{AppName: launchers[0], SkillsDir: skillsDir, Groups: groups, RootSkillMD: rootSkillMD}, nil
}
