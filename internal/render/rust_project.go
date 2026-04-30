package render

import (
	"path/filepath"

	"one-cli/internal/model"
)

func writeRustProject(outputDir, module string, app model.App) error {
	files := []generatedFile{
		{Path: "Cargo.toml", Template: "rust/Cargo.toml.tmpl", Data: templateData{Module: module, App: app}},
		{Path: "README.md", Template: "rust/readme.md.tmpl", Data: templateData{Module: module, App: app}},
		{Path: filepath.Join("src", "main.rs"), Template: "rust/main.rs.tmpl", Data: templateData{Module: module, App: app}},
		{Path: filepath.Join("src", "cli.rs"), Template: "rust/cli.rs.tmpl", Data: templateData{Module: module, App: app}},
		{Path: filepath.Join("src", "client.rs"), Template: "rust/client.rs.tmpl", Data: templateData{Module: module, App: app}},
		{Path: filepath.Join("src", "output.rs"), Template: "rust/output.rs.tmpl", Data: templateData{Module: module, App: app}},
		{Path: filepath.Join("src", "trace.rs"), Template: "rust/trace.rs.tmpl", Data: templateData{Module: module, App: app}},
		{Path: filepath.Join("src", "types.rs"), Template: "rust/types.rs.tmpl", Data: templateData{Module: module, App: app}},
		{Path: filepath.Join("src", "commands", "mod.rs"), Template: "rust/commands_mod.rs.tmpl", Data: templateData{Module: module, App: app}},
	}

	for _, group := range app.Groups {
		files = append(files, generatedFile{
			Path:     filepath.Join("src", "commands", rustModuleName(group)+".rs"),
			Template: "rust/group_command.rs.tmpl",
			Data:     templateData{Module: module, App: app, Group: group},
		})
		groupDir := rustModuleName(group)
		files = append(files, generatedFile{
			Path:     filepath.Join("skills", groupDir, "SKILL.md"),
			Template: "go/skill.md.tmpl",
			Data:     templateData{Module: module, App: app, Group: group},
		})
	}

	return writeTemplates(outputDir, files)
}
