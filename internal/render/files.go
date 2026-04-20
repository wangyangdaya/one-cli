package render

import (
	"os"
	"path/filepath"
	"runtime"
)

type generatedFile struct {
	Path     string
	Template string
	Data     any
	Mode     os.FileMode
}

type templateData struct {
	Module string
	App    any
	Group  any
}

func packageRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		// Fallback to current working directory if Caller fails
		// This should never happen in normal operation
		return "."
	}
	return filepath.Dir(file)
}

func templatePath(name string) string {
	return filepath.Join(packageRoot(), "..", "templates", name)
}

func runtimeRoot() string {
	return filepath.Join(packageRoot(), "..", "runtime")
}

func readRuntimeDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		fullPath := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			children, err := readRuntimeDir(fullPath)
			if err != nil {
				return nil, err
			}
			paths = append(paths, children...)
			continue
		}
		paths = append(paths, fullPath)
	}

	return paths, nil
}
