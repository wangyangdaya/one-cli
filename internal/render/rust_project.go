package render

import (
	"path/filepath"

	"one-cli/internal/model"
	"one-cli/internal/runtimeconfig"
)

func writeRustProject(outputDir, module string, app model.App, skillLang string, runtimeBundle *runtimeconfig.Bundle) error {
	app = normalizeRustApp(app)
	data := templateData{Module: module, Target: "rust", SkillLang: skillLang, App: app}
	runtimeData := data
	runtimeData.RuntimeBundle = runtimeBundle
	files := []generatedFile{
		{Path: "Cargo.toml", Template: "rust/Cargo.toml.tmpl", Data: data},
		{Path: "README.md", Template: "rust/readme.md.tmpl", Data: data},
		{Path: filepath.Join("skills", "README.md"), Template: skillIndexTemplate(skillLang), Data: data},
		{Path: filepath.Join("src", "main.rs"), Template: "rust/main.rs.tmpl", Data: data},
		{Path: filepath.Join("src", "cli.rs"), Template: "rust/cli.rs.tmpl", Data: data},
		{Path: filepath.Join("src", "client.rs"), Template: "rust/client.rs.tmpl", Data: data},
		{Path: filepath.Join("src", "output.rs"), Template: "rust/output.rs.tmpl", Data: data},
		{Path: filepath.Join("src", "trace.rs"), Template: "rust/trace.rs.tmpl", Data: data},
		{Path: filepath.Join("src", "types.rs"), Template: "rust/types.rs.tmpl", Data: data},
		{Path: filepath.Join("src", "runtime_config.rs"), Template: "rust/runtime_config.rs.tmpl", Data: runtimeData},
		{Path: filepath.Join("src", "commands", "mod.rs"), Template: "rust/commands_mod.rs.tmpl", Data: data},
		{Path: filepath.Join("bin", app.Name), Template: "rust/bin_launcher.sh.tmpl", Data: data, Mode: 0o755},
		{Path: filepath.Join("bin", app.Name+".cmd"), Template: "rust/bin_launcher.cmd.tmpl", Data: data, Mode: 0o644},
	}
	if appUsesAKSK(app) {
		files = append(files, generatedFile{Path: filepath.Join("src", "auth.rs"), Template: "rust/auth.rs.tmpl", Data: data})
	}
	if len(app.Groups) > 1 {
		files = append(files, generatedFile{Path: filepath.Join("skills", "SKILL.md"), Template: skillRouterTemplate(skillLang), Data: data})
	}

	for _, group := range app.Groups {
		groupData := data
		groupData.Group = group
		files = append(files, generatedFile{
			Path:     filepath.Join("src", "commands", rustModuleName(group)+".rs"),
			Template: "rust/group_command.rs.tmpl",
			Data:     groupData,
		})
		files = append(files, skillPackageFiles(skillName(group), groupData)...)
	}

	if err := writeTemplates(outputDir, files); err != nil {
		return err
	}
	return writeRuntimeConfig(outputDir, runtimeBundle)
}
