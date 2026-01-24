// Package templates provides the module template system for opm mod init.
package templates

// Template represents a module template with its metadata.
type Template struct {
	// Name is the template identifier (simple, standard, advanced).
	Name string

	// Description explains the template's purpose and use case.
	Description string

	// Default indicates if this is the default template when --template is omitted.
	Default bool

	// UseCase describes when to use this template.
	UseCase string
}

// TemplateData holds the data passed to template rendering.
type TemplateData struct {
	// ModuleName is the name of the module (from --name or directory name).
	ModuleName string

	// ModulePath is the CUE module path (from --module or derived).
	ModulePath string

	// Version is the initial version (hardcoded to 0.1.0).
	Version string

	// PackageName is the CUE package name (sanitized from ModuleName).
	PackageName string
}

// GenerateOptions configures module generation behavior.
type GenerateOptions struct {
	// TargetDir is the directory to generate the module in.
	TargetDir string

	// TemplateName is the template to use.
	TemplateName string

	// ModuleName overrides the module metadata.name.
	ModuleName string

	// ModulePath overrides the CUE module path.
	ModulePath string

	// Force allows overwriting files in non-empty directories.
	Force bool
}

// GenerateResult contains the result of module generation.
type GenerateResult struct {
	// Files is the list of files created.
	Files []string

	// TemplateName is the template that was used.
	TemplateName string

	// TargetDir is the directory where files were created.
	TargetDir string
}
