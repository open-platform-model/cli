// Package templates provides embedded module templates and rendering.
package templates

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed simple/*
var simpleFS embed.FS

//go:embed standard/*
var standardFS embed.FS

//go:embed advanced/*
var advancedFS embed.FS

// TemplateName represents a template type.
type TemplateName string

const (
	// Simple is a minimal single-file template.
	Simple TemplateName = "simple"

	// Standard is the default template with separated concerns.
	Standard TemplateName = "standard"

	// Advanced is a multi-package template for complex modules.
	Advanced TemplateName = "advanced"
)

// ValidTemplates returns all valid template names.
func ValidTemplates() []string {
	return []string{
		string(Simple),
		string(Standard),
		string(Advanced),
	}
}

// IsValidTemplate checks if a template name is valid.
func IsValidTemplate(name string) bool {
	switch TemplateName(name) {
	case Simple, Standard, Advanced:
		return true
	default:
		return false
	}
}

// TemplateData contains data for template rendering.
type TemplateData struct {
	// ModuleName is the name of the module in kebab-case (e.g., "my-app").
	ModuleName string

	// ModuleNamePascal is the PascalCase version of the module name (e.g., "MyApp").
	// Used for the FQN name field which requires uppercase first letter.
	ModuleNamePascal string

	// ModulePath is the module path without version (e.g., "example.com/my-app").
	ModulePath string

	// Version is the initial module version (e.g., "0.1.0").
	Version string
}

// getFS returns the embedded filesystem for a template.
func getFS(name TemplateName) (embed.FS, string, error) {
	switch name {
	case Simple:
		return simpleFS, "simple", nil
	case Standard:
		return standardFS, "standard", nil
	case Advanced:
		return advancedFS, "advanced", nil
	default:
		return embed.FS{}, "", fmt.Errorf("unknown template: %s", name)
	}
}

// Render renders a template to the specified directory.
func Render(templateName TemplateName, targetDir string, data TemplateData) ([]string, error) {
	fsys, rootDir, err := getFS(templateName)
	if err != nil {
		return nil, err
	}

	var createdFiles []string

	err = fs.WalkDir(fsys, rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Get the relative path from the template root
		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		// Target path in the output directory
		targetPath := filepath.Join(targetDir, relPath)

		// If it's a directory, create it
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}

		// Read the template file
		content, err := fs.ReadFile(fsys, path)
		if err != nil {
			return fmt.Errorf("reading template %s: %w", path, err)
		}

		// Remove .tmpl extension from target filename (TrimSuffix is a no-op if suffix not present)
		targetPath = strings.TrimSuffix(targetPath, ".tmpl")

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", targetPath, err)
		}

		// Parse and execute the template
		tmpl, err := template.New(filepath.Base(path)).Parse(string(content))
		if err != nil {
			return fmt.Errorf("parsing template %s: %w", path, err)
		}

		f, err := os.Create(targetPath)
		if err != nil {
			return fmt.Errorf("creating file %s: %w", targetPath, err)
		}
		defer f.Close()

		if err := tmpl.Execute(f, data); err != nil {
			return fmt.Errorf("executing template %s: %w", path, err)
		}

		// Add the cleaned path (without .tmpl) to createdFiles
		cleanPath := strings.TrimSuffix(relPath, ".tmpl")
		createdFiles = append(createdFiles, cleanPath)
		return nil
	})

	return createdFiles, err
}

// ListTemplateFiles returns all files in a template.
func ListTemplateFiles(templateName TemplateName) ([]string, error) {
	fsys, rootDir, err := getFS(templateName)
	if err != nil {
		return nil, err
	}

	var files []string

	err = fs.WalkDir(fsys, rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}

		// Remove .tmpl extension (TrimSuffix is a no-op if suffix not present)
		relPath = strings.TrimSuffix(relPath, ".tmpl")

		files = append(files, relPath)
		return nil
	})

	return files, err
}
