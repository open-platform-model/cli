package modulecmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/templates"
	oerrors "github.com/open-platform-model/cli/pkg/errors"
)

// moduleNameRegex enforces a strict subset of the module-path segment grammar:
// lowercase letters, digits, and hyphens; must start with a letter; must not
// end with a hyphen. Matches the convention used by every module in modules/.
var moduleNameRegex = regexp.MustCompile(`^[a-z]([a-z0-9-]*[a-z0-9])?$`)

const validationFailedType = "validation failed"

// NewModuleInitCmd creates the module init command.
func NewModuleInitCmd(_ *config.GlobalConfig) *cobra.Command {
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
  opm module init my-app

  # Create a module with a specific template
  opm module init my-app --template simple

  # Create a module in a specific directory
  opm module init my-app --dir ./modules`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runModuleInit(args, templateFlag, dirFlag)
		},
	}

	c.Flags().StringVarP(&templateFlag, "template", "t", "standard",
		fmt.Sprintf("Template to use (%s)", strings.Join(templates.ValidTemplates(), ", ")))
	c.Flags().StringVarP(&dirFlag, "dir", "d", "",
		"Directory to create module in (defaults to module name)")

	return c
}

func runModuleInit(args []string, templateName, dir string) error {
	moduleName := args[0]

	// Validate module name
	if !moduleNameRegex.MatchString(moduleName) {
		return &opmexit.ExitError{
			Code: opmexit.ExitValidationError,
			Err: &oerrors.DetailError{
				Type:    validationFailedType,
				Message: fmt.Sprintf("invalid module name: %q", moduleName),
				Hint:    `Module names must be lowercase letters, digits, and hyphens; must start with a letter and not end with a hyphen (e.g. "my-app", "cert-manager").`,
				Cause:   oerrors.ErrValidation,
			},
		}
	}

	// Validate template name
	if !templates.IsValidTemplate(templateName) {
		return &opmexit.ExitError{
			Code: opmexit.ExitValidationError,
			Err: &oerrors.DetailError{
				Type:    validationFailedType,
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
		return &opmexit.ExitError{
			Code: opmexit.ExitValidationError,
			Err: &oerrors.DetailError{
				Type:     validationFailedType,
				Message:  fmt.Sprintf("directory already exists: %s", targetDir),
				Location: targetDir,
				Hint:     "Choose a different directory or remove the existing one.",
				Cause:    oerrors.ErrValidation,
			},
		}
	}

	// Create the target directory
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return &opmexit.ExitError{
			Code: opmexit.ExitGeneralError,
			Err:  fmt.Errorf("creating directory %s: %w", targetDir, err),
		}
	}

	// Prepare template data
	absDir, err := filepath.Abs(targetDir)
	if err != nil {
		return &opmexit.ExitError{
			Code: opmexit.ExitGeneralError,
			Err:  fmt.Errorf("getting absolute path: %w", err),
		}
	}

	data := templates.TemplateData{
		ModuleName:  moduleName,
		PackageName: toPackageName(moduleName),
		ModulePath:  "example.com/modules",
		Version:     "0.1.0",
	}

	// Render the template
	createdFiles, err := templates.Render(templates.TemplateName(templateName), targetDir, data)
	if err != nil {
		// Clean up on failure
		_ = os.RemoveAll(targetDir)
		return &opmexit.ExitError{
			Code: opmexit.ExitGeneralError,
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

// toPackageName converts a module name to a valid CUE package name.
func toPackageName(s string) string {
	var result strings.Builder
	lastUnderscore := false

	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			result.WriteRune(unicode.ToLower(r))
			lastUnderscore = false
		case !lastUnderscore:
			result.WriteRune('_')
			lastUnderscore = true
		}
	}

	name := strings.Trim(result.String(), "_")
	if name == "" {
		return "module"
	}

	if name[0] >= '0' && name[0] <= '9' {
		return "module_" + name
	}

	return name
}
