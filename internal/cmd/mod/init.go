package mod

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/config"
	oerrors "github.com/opmodel/cli/internal/errors"
	"github.com/opmodel/cli/internal/output"
	"github.com/opmodel/cli/internal/templates"
)

// NewModInitCmd creates the mod init command.
func NewModInitCmd(_ *config.GlobalConfig) *cobra.Command {
	var templateFlag string
	var dirFlag string

	c := &cobra.Command{
		Use:   "init <module-name>",
		Short: "Create a new module from template",
		Long: `Create a new OPM module from a template.

Templates:
  simple    Minimal single-file module for learning and prototypes
  standard  Separated concerns for team collaboration (default)
  advanced  Multi-package architecture for complex platforms

Examples:
  # Create a module with the standard template (default)
  opm mod init my-app

  # Create a module with a specific template
  opm mod init my-app --template simple

  # Create a module in a specific directory
  opm mod init my-app --dir ./modules`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runModInit(args, templateFlag, dirFlag)
		},
	}

	c.Flags().StringVarP(&templateFlag, "template", "t", "standard",
		fmt.Sprintf("Template to use (%s)", strings.Join(templates.ValidTemplates(), ", ")))
	c.Flags().StringVarP(&dirFlag, "dir", "d", "",
		"Directory to create module in (defaults to module name)")

	return c
}

func runModInit(args []string, templateName, dir string) error {
	moduleName := args[0]

	// Validate template name
	if !templates.IsValidTemplate(templateName) {
		return &oerrors.ExitError{
			Code: oerrors.ExitValidationError,
			Err: &oerrors.DetailError{
				Type:    "validation failed",
				Message: fmt.Sprintf("unknown template: %s", templateName),
				Hint:    fmt.Sprintf("Valid templates: %s", strings.Join(templates.ValidTemplates(), ", ")),
				Cause:   oerrors.ErrValidation,
			},
		}
	}

	// Determine target directory
	targetDir := dir
	if targetDir == "" {
		targetDir = moduleName
	}

	// Check if directory already exists
	if _, err := os.Stat(targetDir); err == nil {
		return &oerrors.ExitError{
			Code: oerrors.ExitValidationError,
			Err: &oerrors.DetailError{
				Type:     "validation failed",
				Message:  fmt.Sprintf("directory already exists: %s", targetDir),
				Location: targetDir,
				Hint:     "Choose a different directory or remove the existing one.",
				Cause:    oerrors.ErrValidation,
			},
		}
	}

	// Create the target directory
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return &oerrors.ExitError{
			Code: oerrors.ExitGeneralError,
			Err:  fmt.Errorf("creating directory %s: %w", targetDir, err),
		}
	}

	// Prepare template data
	absDir, err := filepath.Abs(targetDir)
	if err != nil {
		return &oerrors.ExitError{
			Code: oerrors.ExitGeneralError,
			Err:  fmt.Errorf("getting absolute path: %w", err),
		}
	}

	data := templates.TemplateData{
		ModuleName:       moduleName,
		ModuleNamePascal: toPascalCase(moduleName),
		ModulePath:       fmt.Sprintf("example.com/%s", moduleName),
		Version:          "0.1.0",
	}

	// Render the template
	createdFiles, err := templates.Render(templates.TemplateName(templateName), targetDir, data)
	if err != nil {
		// Clean up on failure
		_ = os.RemoveAll(targetDir)
		return &oerrors.ExitError{
			Code: oerrors.ExitGeneralError,
			Err:  fmt.Errorf("rendering template: %w", err),
		}
	}

	// Print success output with aligned file tree
	output.Println(fmt.Sprintf("Created module '%s' in %s\n", moduleName, absDir))

	// Build file entries for aligned output
	entries := make([]output.FileEntry, 0, len(createdFiles)+1)
	entries = append(entries, output.FileEntry{
		Path:        targetDir + "/",
		Description: "Module directory",
	})

	for _, f := range createdFiles {
		desc := getFileDescription(f)
		entries = append(entries, output.FileEntry{
			Path:        "  " + f,
			Description: desc,
		})
	}

	// Render with column 30 alignment per spec
	output.Print(output.RenderFileTree(entries, 30))

	return nil
}

// getFileDescription returns a description for a template file.
func getFileDescription(filename string) string {
	// Remove .tmpl suffix if present
	filename = strings.TrimSuffix(filename, ".tmpl")

	descriptions := map[string]string{
		"cue.mod/module.cue": "CUE module metadata",
		"module.cue":         "Module definition",
		"values.cue":         "Default values",
		"components.cue":     "Component definitions",
		"scopes.cue":         "Scope definitions",
		"policies.cue":       "Policy definitions",
		"debug_values.cue":   "Debug-specific values",
	}

	if desc, ok := descriptions[filename]; ok {
		return desc
	}

	// Handle subdirectory files
	if strings.HasPrefix(filename, "components/") {
		return "Component template"
	}
	if strings.HasPrefix(filename, "scopes/") {
		return "Scope template"
	}

	return ""
}

// toPascalCase converts a kebab-case or snake_case string to PascalCase.
// Examples: "my-app" -> "MyApp", "my_service" -> "MyService"
func toPascalCase(s string) string {
	var result strings.Builder
	capitalizeNext := true

	for _, r := range s {
		if r == '-' || r == '_' {
			capitalizeNext = true
			continue
		}
		if capitalizeNext {
			result.WriteRune(unicode.ToUpper(r))
			capitalizeNext = false
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}
