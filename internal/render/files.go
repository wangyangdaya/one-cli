package render

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"

	"one-cli/internal/model"
	"one-cli/internal/runtimeconfig"
)

type generatedFile struct {
	Path     string
	Template string
	Data     any
	Mode     os.FileMode
}

type templateData struct {
	Module        string
	Target        string
	SkillLang     string
	App           model.App
	Group         model.Group
	RuntimeBundle *runtimeconfig.Bundle
}

func readTemplate(name string) ([]byte, error) {
	return embeddedFS.ReadFile("templates/" + name)
}

func listEmbedDir(fsys embed.FS, dir string) ([]string, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, err
	}

	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		fullPath := dir + "/" + entry.Name()
		if entry.IsDir() {
			children, err := listEmbedDir(fsys, fullPath)
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

func writeRuntime(outputDir string) error {
	paths, err := listEmbedDir(embeddedFS, "runtime")
	if err != nil {
		return err
	}

	for _, path := range paths {
		content, err := embeddedFS.ReadFile(path)
		if err != nil {
			return err
		}

		// Strip the "runtime/" prefix to get the relative path
		relative := path[len("runtime/"):]
		if err := writeFile(filepath.Join(outputDir, "internal", relative), content, 0); err != nil {
			return err
		}
	}

	return nil
}

func writeFile(path string, content []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if mode == 0 {
		mode = 0o644
	}
	return os.WriteFile(path, content, mode)
}

func writeRuntimeConfig(outputDir string, bundle *runtimeconfig.Bundle) error {
	if bundle == nil {
		return nil
	}
	return writeFile(filepath.Join(outputDir, "config", "runtime.yaml"), bundle.YAML, 0)
}
