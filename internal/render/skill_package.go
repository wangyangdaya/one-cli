package render

import "path/filepath"

func skillPackageFiles(groupDir string, data templateData) []generatedFile {
	base := filepath.Join("skills", groupDir)
	return []generatedFile{
		{Path: filepath.Join(base, "SKILL.md"), Template: "skill.md.tmpl", Data: data},
		{Path: filepath.Join(base, "README.md"), Template: "skill/README.md.tmpl", Data: data},
		{Path: filepath.Join(base, "assets", "demo-request.json"), Template: "skill/assets/demo-request.json.tmpl", Data: data},
		{Path: filepath.Join(base, "references", "command-routing.md"), Template: "skill/references/command-routing.md.tmpl", Data: data},
		{Path: filepath.Join(base, "references", "workflows.md"), Template: "skill/references/workflows.md.tmpl", Data: data},
		{Path: filepath.Join(base, "references", "production-checklist.md"), Template: "skill/references/production-checklist.md.tmpl", Data: data},
	}
}
