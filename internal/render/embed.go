package render

import "embed"

//go:embed templates runtime gomod.tmpl gosum.tmpl
var embeddedFS embed.FS
