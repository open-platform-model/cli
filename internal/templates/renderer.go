package templates

import (
	"bytes"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"
)

// Renderer handles template rendering with data substitution.
type Renderer struct {
	data TemplateData
}

// NewRenderer creates a new renderer with the given template data.
func NewRenderer(data TemplateData) *Renderer {
	return &Renderer{data: data}
}

// RenderFile renders a single template file and returns the content.
func (r *Renderer) RenderFile(content []byte) ([]byte, error) {
	tmpl, err := template.New("file").Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, r.data); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	return buf.Bytes(), nil
}

// RenderString renders a template string and returns the result.
func (r *Renderer) RenderString(content string) (string, error) {
	result, err := r.RenderFile([]byte(content))
	if err != nil {
		return "", err
	}
	return string(result), nil
}

// TemplateFile represents a file to be generated from a template.
type TemplateFile struct {
	// SourcePath is the path within the embedded filesystem.
	SourcePath string

	// TargetPath is the output path (with .tmpl suffix removed).
	TargetPath string

	// Content is the rendered content.
	Content []byte
}

// RenderTemplate renders all files from a template and returns them.
func (r *Renderer) RenderTemplate(templateName string) ([]TemplateFile, error) {
	var files []TemplateFile

	err := fs.WalkDir(TemplateFS, templateName, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Only process .tmpl files
		if !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		// Read the template content
		content, err := fs.ReadFile(TemplateFS, path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		// Render the content
		rendered, err := r.RenderFile(content)
		if err != nil {
			return fmt.Errorf("rendering %s: %w", path, err)
		}

		// Calculate target path (remove template name prefix and .tmpl suffix)
		relPath := strings.TrimPrefix(path, templateName+"/")
		targetPath := strings.TrimSuffix(relPath, ".tmpl")

		files = append(files, TemplateFile{
			SourcePath: path,
			TargetPath: targetPath,
			Content:    rendered,
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking template %s: %w", templateName, err)
	}

	return files, nil
}

// ListTemplateFiles returns the list of files in a template without rendering.
func ListTemplateFiles(templateName string) ([]string, error) {
	var files []string

	err := fs.WalkDir(TemplateFS, templateName, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory
		if path == templateName {
			return nil
		}

		// Skip directories but continue walking
		if d.IsDir() {
			return nil
		}

		// Only include .tmpl files
		if !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		// Calculate relative path and remove .tmpl suffix
		relPath := strings.TrimPrefix(path, templateName+"/")
		targetPath := strings.TrimSuffix(relPath, ".tmpl")

		files = append(files, targetPath)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("listing template %s: %w", templateName, err)
	}

	return files, nil
}

// GetTemplateDirectories returns the subdirectories in a template.
func GetTemplateDirectories(templateName string) ([]string, error) {
	var dirs []string
	seen := make(map[string]bool)

	err := fs.WalkDir(TemplateFS, templateName, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory
		if path == templateName {
			return nil
		}

		// Get relative path
		relPath := strings.TrimPrefix(path, templateName+"/")

		// Track directories
		if d.IsDir() {
			if !seen[relPath] {
				dirs = append(dirs, relPath)
				seen[relPath] = true
			}
		} else {
			// Also track parent directories of files
			dir := filepath.Dir(relPath)
			if dir != "." && !seen[dir] {
				dirs = append(dirs, dir)
				seen[dir] = true
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return dirs, nil
}
