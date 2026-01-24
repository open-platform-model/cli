package mod

import (
	"fmt"
	"path/filepath"

	"github.com/opmodel/cli/internal/cmd"
	"github.com/opmodel/cli/internal/output"
	"github.com/opmodel/cli/internal/templates"
	"github.com/spf13/cobra"
)

// initOptions holds the flags for the init command.
type initOptions struct {
	template string
	name     string
	module   string
	force    bool
}

// NewInitCmd creates the mod init command.
func NewInitCmd() *cobra.Command {
	opts := &initOptions{}

	c := &cobra.Command{
		Use:   "init <directory>",
		Short: "Initialize a new OPM module",
		Long: `Creates a new OPM module with the specified template structure.

Templates available:
  simple    Single-file inline - Learning OPM, prototypes
  standard  Separated components - Team projects, production (default)
  advanced  Multi-package with subpackages - Complex platforms, enterprise

Examples:
  # Create a module with default (standard) template
  opm mod init my-app

  # Create a module with simple template
  opm mod init my-app --template simple

  # Create a module with custom name and path
  opm mod init my-app --name myapp --module github.com/org/myapp

  # Overwrite existing files
  opm mod init my-app --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runInit(args[0], opts)
		},
	}

	c.Flags().StringVarP(&opts.template, "template", "t", "", "Module template to use (simple, standard, advanced)")
	c.Flags().StringVar(&opts.name, "name", "", "Module name (defaults to directory name)")
	c.Flags().StringVar(&opts.module, "module", "", "CUE module path (defaults to example.com/<dirname>)")
	c.Flags().BoolVarP(&opts.force, "force", "f", false, "Overwrite existing files in non-empty directory")

	return c
}

// runInit creates a new module using the template system.
func runInit(dir string, opts *initOptions) error {
	// Determine target directory
	targetDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	// Determine template to use
	templateName := opts.template
	if templateName == "" {
		templateName = templates.DefaultTemplateName
		output.Info("Using template", "name", templateName)
	}

	// Validate template exists
	tmpl, err := templates.Get(templateName)
	if err != nil {
		return cmd.NewExitError(err, cmd.ExitValidationError)
	}

	// Determine module name
	moduleName := opts.name
	if moduleName == "" {
		moduleName = filepath.Base(targetDir)
	}

	// Validate module name
	if err := templates.ValidateModuleName(moduleName); err != nil {
		return cmd.NewExitError(err, cmd.ExitValidationError)
	}

	// Generate the module
	gen := templates.NewGenerator(templates.GenerateOptions{
		TargetDir:    targetDir,
		TemplateName: templateName,
		ModuleName:   moduleName,
		ModulePath:   opts.module,
		Force:        opts.force,
	})

	result, err := gen.Generate()
	if err != nil {
		// Check if it's a validation error (name/path issues)
		if err := templates.ValidateModuleName(moduleName); err != nil {
			return cmd.NewExitError(err, cmd.ExitValidationError)
		}
		return fmt.Errorf("generating module: %w", err)
	}

	// Print success message
	fmt.Printf("Created module %s in %s\n", moduleName, result.TargetDir)
	fmt.Printf("Template: %s (%s)\n", tmpl.Name, tmpl.Description)
	fmt.Printf("Files created: %d\n", len(result.Files))

	// Print next steps
	fmt.Println("\nNext steps:")
	relDir, _ := filepath.Rel(".", result.TargetDir)
	if relDir == "" || relDir == "." {
		relDir = filepath.Base(result.TargetDir)
	}
	fmt.Printf("  cd %s\n", relDir)
	fmt.Println("  opm mod vet    # Validate the module")
	fmt.Println("  opm mod build  # Build manifests")

	return nil
}

// sanitizeName converts a module name to a valid CUE package name.
// Deprecated: Use templates.SanitizeName instead.
func sanitizeName(name string) string {
	return templates.SanitizeName(name)
}

func init() {
	// Register exit codes
	_ = cmd.ExitSuccess
}
