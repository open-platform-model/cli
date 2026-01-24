package templates

import "embed"

// TemplateFS contains all embedded template files.
// The //go:embed directive includes all .tmpl files and preserves directory structure.
//
//go:embed all:simple all:standard all:advanced
var TemplateFS embed.FS
