package mod

import (
	"fmt"
	"sort"
	"strings"

	"github.com/opmodel/cli/internal/cmd"
	"github.com/opmodel/cli/internal/templates"
	"github.com/spf13/cobra"
)

// NewTemplateCmd creates the template command group.
func NewTemplateCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "template",
		Short: "Manage module templates",
		Long: `Commands for discovering and inspecting module templates.

Templates define the initial structure and files for new OPM modules.
Use 'opm mod init --template <name>' to create a module from a template.`,
	}

	c.AddCommand(
		newTemplateListCmd(),
		newTemplateShowCmd(),
	)

	return c
}

// newTemplateListCmd creates the template list subcommand.
func newTemplateListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available templates",
		Long: `Lists all available module templates with their descriptions.

The default template is marked with "(default)".`,
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			return runTemplateList()
		},
	}
}

// newTemplateShowCmd creates the template show subcommand.
func newTemplateShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show template details",
		Long: `Shows detailed information about a template including:
- Description and use case
- List of files that will be created
- Directory structure`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runTemplateShow(args[0])
		},
	}
}

// runTemplateList displays all available templates.
func runTemplateList() error {
	tmplList := templates.List()

	// Print header
	fmt.Println("Available templates:")
	fmt.Println()

	// Calculate column widths
	maxNameLen := 8 // minimum width
	for _, t := range tmplList {
		name := t.Name
		if t.Default {
			name += " (default)"
		}
		if len(name) > maxNameLen {
			maxNameLen = len(name)
		}
	}

	// Print table header
	fmt.Printf("  %-*s  %s\n", maxNameLen, "NAME", "DESCRIPTION")
	fmt.Printf("  %-*s  %s\n", maxNameLen, strings.Repeat("-", maxNameLen), strings.Repeat("-", 50))

	// Print templates
	for _, t := range tmplList {
		name := t.Name
		if t.Default {
			name += " (default)"
		}
		fmt.Printf("  %-*s  %s\n", maxNameLen, name, t.Description)
	}

	fmt.Println()
	fmt.Println("Use 'opm mod template show <name>' for details about a template.")
	fmt.Println("Use 'opm mod init <directory> --template <name>' to create a module.")

	return nil
}

// runTemplateShow displays details about a specific template.
func runTemplateShow(name string) error {
	// Get template
	tmpl, err := templates.Get(name)
	if err != nil {
		fmt.Printf("Unknown template %q.\n\n", name)
		fmt.Println("Available templates:")
		for _, n := range templates.Names() {
			fmt.Printf("  - %s\n", n)
		}
		return cmd.NewExitError(err, cmd.ExitValidationError)
	}

	// Print template info
	fmt.Printf("Template: %s\n", tmpl.Name)
	if tmpl.Default {
		fmt.Println("Default: yes")
	}
	fmt.Println()
	fmt.Printf("Description: %s\n", tmpl.Description)
	fmt.Printf("Use case: %s\n", tmpl.UseCase)
	fmt.Println()

	// Get file list
	files, err := templates.ListTemplateFiles(name)
	if err != nil {
		return fmt.Errorf("listing template files: %w", err)
	}

	// Get directories
	dirs, err := templates.GetTemplateDirectories(name)
	if err != nil {
		return fmt.Errorf("listing template directories: %w", err)
	}

	// Sort for consistent output
	sort.Strings(files)
	sort.Strings(dirs)

	// Print directory structure
	fmt.Printf("Files (%d):\n", len(files))

	// Group files by directory for better display
	filesByDir := make(map[string][]string)
	rootFiles := []string{}

	for _, f := range files {
		dir := "."
		if idx := strings.LastIndex(f, "/"); idx != -1 {
			dir = f[:idx]
		}
		if dir == "." {
			rootFiles = append(rootFiles, f)
		} else {
			filesByDir[dir] = append(filesByDir[dir], f)
		}
	}

	// Print root files first
	for _, f := range rootFiles {
		fmt.Printf("  %s\n", f)
	}

	// Print subdirectory files
	sortedDirs := make([]string, 0, len(filesByDir))
	for d := range filesByDir {
		sortedDirs = append(sortedDirs, d)
	}
	sort.Strings(sortedDirs)

	for _, d := range sortedDirs {
		for _, f := range filesByDir[d] {
			fmt.Printf("  %s\n", f)
		}
	}

	fmt.Println()
	fmt.Printf("Create a module with: opm mod init <directory> --template %s\n", name)

	return nil
}
