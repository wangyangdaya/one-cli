package render

import (
	"path/filepath"

	"one-cli/internal/model"
)

func writeRustProject(outputDir, module string, app model.App, skillLang string) error {
	app = normalizeRustApp(app)
	files := []generatedFile{
		{Path: "Cargo.toml", Template: "rust/Cargo.toml.tmpl", Data: templateData{Module: module, Target: "rust", SkillLang: skillLang, App: app}},
		{Path: "README.md", Template: "rust/readme.md.tmpl", Data: templateData{Module: module, Target: "rust", SkillLang: skillLang, App: app}},
		{Path: filepath.Join("src", "main.rs"), Template: "rust/main.rs.tmpl", Data: templateData{Module: module, Target: "rust", SkillLang: skillLang, App: app}},
		{Path: filepath.Join("src", "cli.rs"), Template: "rust/cli.rs.tmpl", Data: templateData{Module: module, Target: "rust", SkillLang: skillLang, App: app}},
		{Path: filepath.Join("src", "client.rs"), Template: "rust/client.rs.tmpl", Data: templateData{Module: module, Target: "rust", SkillLang: skillLang, App: app}},
		{Path: filepath.Join("src", "output.rs"), Template: "rust/output.rs.tmpl", Data: templateData{Module: module, Target: "rust", SkillLang: skillLang, App: app}},
		{Path: filepath.Join("src", "trace.rs"), Template: "rust/trace.rs.tmpl", Data: templateData{Module: module, Target: "rust", SkillLang: skillLang, App: app}},
		{Path: filepath.Join("src", "types.rs"), Template: "rust/types.rs.tmpl", Data: templateData{Module: module, Target: "rust", SkillLang: skillLang, App: app}},
		{Path: filepath.Join("src", "commands", "mod.rs"), Template: "rust/commands_mod.rs.tmpl", Data: templateData{Module: module, Target: "rust", SkillLang: skillLang, App: app}},
	}
	if appUsesAKSK(app) {
		files = append(files, generatedFile{Path: filepath.Join("src", "auth.rs"), Template: "rust/auth.rs.tmpl", Data: templateData{Module: module, Target: "rust", SkillLang: skillLang, App: app}})
	}

	for _, group := range app.Groups {
		files = append(files, generatedFile{
			Path:     filepath.Join("src", "commands", rustModuleName(group)+".rs"),
			Template: "rust/group_command.rs.tmpl",
			Data:     templateData{Module: module, Target: "rust", SkillLang: skillLang, App: app, Group: group},
		})
		groupDir := rustModuleName(group)
		files = append(files, skillPackageFiles(groupDir, templateData{Module: module, Target: "rust", SkillLang: skillLang, App: app, Group: group})...)
	}

	if app.SingleSkill {
		prefix := "skill"
		if skillLang == "zh" {
			prefix = "skill_zh"
		}
		skillDir := filepath.Join("skills", skillPackageName(app))
		files = append(files, generatedFile{
			Path:     filepath.Join(skillDir, "SKILL.md"),
			Template: prefix + "/unified_skill.md.tmpl",
			Data:     templateData{Module: module, Target: "rust", SkillLang: skillLang, App: app},
		})
		files = append(files,
			generatedFile{Path: filepath.Join(skillDir, "scripts", app.Name), Template: "skill/scripts/single_launcher.sh.tmpl", Data: templateData{Module: module, Target: "rust", SkillLang: skillLang, App: app}, Mode: 0o755},
			generatedFile{Path: filepath.Join(skillDir, "scripts", app.Name+".cmd"), Template: "skill/scripts/single_launcher.cmd.tmpl", Data: templateData{Module: module, Target: "rust", SkillLang: skillLang, App: app}, Mode: 0o644},
		)
	}

	return writeTemplates(outputDir, files)
}
