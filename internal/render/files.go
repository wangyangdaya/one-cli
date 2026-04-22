package render

import (
	"embed"
	"io/fs"
	"os"
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
