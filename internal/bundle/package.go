package bundle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Options struct {
	ProjectDir string
	OutputDir  string
	BinaryPath string
}

type Result struct {
	AppName   string
	OutputDir string
	Groups    []string
}

func Package(options Options) (Result, error) {
	projectDir, err := existingDirectory(options.ProjectDir, "project")
	if err != nil {
		return Result{}, err
	}
	outputDir := strings.TrimSpace(options.OutputDir)
	if outputDir == "" {
		return Result{}, fmt.Errorf("missing output directory")
	}

	project, err := discoverProject(projectDir)
	if err != nil {
		return Result{}, err
	}
	binaryPath, cleanupBinary, err := resolveBinary(projectDir, project.AppName, options.BinaryPath)
	if err != nil {
		return Result{}, err
	}
	defer cleanupBinary()

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return Result{}, err
	}
	if project.RootSkillMD != "" {
		if err := copyAdaptedFile(project.RootSkillMD, filepath.Join(outputDir, "SKILL.md"), 0o644, project.AppName, "./bin/"); err != nil {
			return Result{}, err
		}
	} else {
		content := singleGroupRouter(project.AppName, project.Groups[0])
		if err := writeFileAtomic(filepath.Join(outputDir, "SKILL.md"), []byte(content), 0o644); err != nil {
			return Result{}, err
		}
	}
	if err := copyAdaptedFile(filepath.Join(project.SkillsDir, "README.md"), filepath.Join(outputDir, "README.md"), 0o644, project.AppName, "./bin/"); err != nil {
		return Result{}, err
	}
	for _, group := range project.Groups {
		if err := copyGroup(filepath.Join(project.SkillsDir, group), filepath.Join(outputDir, group), project.AppName); err != nil {
			return Result{}, err
		}
	}
	if err := copyFileAtomic(binaryPath, filepath.Join(outputDir, "bin", executableName(project.AppName)), 0o755); err != nil {
		return Result{}, err
	}
	runtimeConfig := filepath.Join(projectDir, "config", "runtime.yaml")
	if info, err := os.Stat(runtimeConfig); err == nil && info.Mode().IsRegular() {
		if err := copyFileAtomic(runtimeConfig, filepath.Join(outputDir, "config", "runtime.yaml"), 0o644); err != nil {
			return Result{}, err
		}
	}
	for _, runtimeName := range []string{"python", "node"} {
		if err := os.MkdirAll(filepath.Join(outputDir, "libexec", runtimeName), 0o755); err != nil {
			return Result{}, err
		}
	}
	if sourceLibexec := filepath.Join(projectDir, "libexec"); directoryExists(sourceLibexec) {
		if err := copyTree(sourceLibexec, filepath.Join(outputDir, "libexec")); err != nil {
			return Result{}, err
		}
	}

	return Result{AppName: project.AppName, OutputDir: outputDir, Groups: project.Groups}, nil
}

func singleGroupRouter(appName, group string) string {
	return fmt.Sprintf(`---
name: %s-skills
description: Route requests for %s to its generated command-group Skill.
---

# %s Skills Router

Read [%s/SKILL.md](%s/SKILL.md) before invoking the shared CLI.

List installed groups with:

`+"```bash"+`
./bin/%s skills list
`+"```"+`
`, appName, appName, appName, group, group, appName)
}

func existingDirectory(value, label string) (string, error) {
	path := strings.TrimSpace(value)
	if path == "" {
		return "", fmt.Errorf("missing %s directory", label)
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("%s directory %q not found", label, path)
	}
	return path, nil
}

func directoryExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
