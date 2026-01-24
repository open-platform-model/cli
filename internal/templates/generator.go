package templates

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/opmodel/cli/internal/output"
)

// Generator handles module generation from templates.
type Generator struct {
	opts GenerateOptions
}

// NewGenerator creates a new generator with the given options.
func NewGenerator(opts GenerateOptions) *Generator {
	return &Generator{opts: opts}
}

// Generate creates a new module from a template.
func (g *Generator) Generate() (*GenerateResult, error) {
	// Validate template exists
	tmpl, err := Get(g.opts.TemplateName)
	if err != nil {
		return nil, err
	}

	// Determine module name and path
	dirname := filepath.Base(g.opts.TargetDir)
	moduleName := g.opts.ModuleName
	if moduleName == "" {
		moduleName = dirname
	}

	modulePath := g.opts.ModulePath
	if modulePath == "" {
		modulePath = DeriveModulePath(dirname)
	}

	// Validate module name
	if err := ValidateModuleName(moduleName); err != nil {
		return nil, err
	}

	// Check target directory
	if err := g.checkTargetDir(); err != nil {
		return nil, err
	}

	// Prepare template data
	data := TemplateData{
		ModuleName:  moduleName,
		ModulePath:  modulePath,
		Version:     "0.1.0",
		PackageName: SanitizeName(moduleName),
	}

	output.Debug("generating module",
		"template", tmpl.Name,
		"name", moduleName,
		"path", modulePath,
		"target", g.opts.TargetDir)

	// Render template files
	renderer := NewRenderer(data)
	files, err := renderer.RenderTemplate(g.opts.TemplateName)
	if err != nil {
		return nil, fmt.Errorf("rendering template: %w", err)
	}

	// Create directories and write files
	createdFiles := make([]string, 0, len(files))
	for _, f := range files {
		targetPath := filepath.Join(g.opts.TargetDir, f.TargetPath)

		// Create parent directory
		parentDir := filepath.Dir(targetPath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return nil, fmt.Errorf("creating directory %s: %w", parentDir, err)
		}

		// Check if file exists and --force is not set
		if !g.opts.Force {
			if _, err := os.Stat(targetPath); err == nil {
				return nil, fmt.Errorf("file %s already exists; use --force to overwrite", targetPath)
			}
		}

		// Write file
		if err := os.WriteFile(targetPath, f.Content, 0644); err != nil {
			return nil, fmt.Errorf("writing %s: %w", targetPath, err)
		}

		output.Debug("created file", "path", f.TargetPath)
		createdFiles = append(createdFiles, f.TargetPath)
	}

	return &GenerateResult{
		Files:        createdFiles,
		TemplateName: tmpl.Name,
		TargetDir:    g.opts.TargetDir,
	}, nil
}

// checkTargetDir validates the target directory.
func (g *Generator) checkTargetDir() error {
	info, err := os.Stat(g.opts.TargetDir)
	if os.IsNotExist(err) {
		// Directory doesn't exist, will be created
		return nil
	}
	if err != nil {
		return fmt.Errorf("checking target directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", g.opts.TargetDir)
	}

	// Check if directory is empty
	entries, err := os.ReadDir(g.opts.TargetDir)
	if err != nil {
		return fmt.Errorf("reading target directory: %w", err)
	}

	if len(entries) > 0 && !g.opts.Force {
		return fmt.Errorf("directory %s is not empty; use --force to overwrite existing files", g.opts.TargetDir)
	}

	return nil
}
