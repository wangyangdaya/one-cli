package render

import "path/filepath"

func skillPackageFiles(groupDir string, data templateData) []generatedFile {
	base := filepath.Join("skills", groupDir)
	prefix := "skill"
	if data.SkillLang == "zh" {
		prefix = "skill_zh"
	}
	return []generatedFile{
		{Path: filepath.Join(base, "SKILL.md"), Template: prefix + ".md.tmpl", Data: data},
		{Path: filepath.Join(base, "README.md"), Template: prefix + "/readme.md.tmpl", Data: data},
		{Path: filepath.Join(base, "assets", "demo-request.json"), Template: prefix + "/assets/demo-request.json.tmpl", Data: data},
		{Path: filepath.Join(base, "references", "command-routing.md"), Template: prefix + "/references/command-routing.md.tmpl", Data: data},
		{Path: filepath.Join(base, "references", "workflows.md"), Template: prefix + "/references/workflows.md.tmpl", Data: data},
		{Path: filepath.Join(base, "references", "production-checklist.md"), Template: prefix + "/references/production-checklist.md.tmpl", Data: data},
		{Path: filepath.Join(base, "generation-report.md"), Template: prefix + "/generation-report.md.tmpl", Data: data},
	}
}
