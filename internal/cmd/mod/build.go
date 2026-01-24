package mod

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/opmodel/cli/internal/cmd"
	opmcue "github.com/opmodel/cli/internal/cue"
	"github.com/opmodel/cli/internal/output"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// buildOptions holds the flags for the build command.
type buildOptions struct {
	dir        string
	values     []string
	outputFmt  string
	outputDir  string
	outputFile string
}

// NewBuildCmd creates the mod build command.
func NewBuildCmd() *cobra.Command {
	opts := &buildOptions{}

	c := &cobra.Command{
		Use:   "build",
		Short: "Build module manifests",
		Long:  `Renders the module to Kubernetes manifests.`,
		RunE: func(c *cobra.Command, args []string) error {
			return runBuild(c.Context(), opts)
		},
	}

	c.Flags().StringVar(&opts.dir, "dir", ".", "Module directory")
	c.Flags().StringSliceVarP(&opts.values, "values", "f", nil, "Values files to unify")
	c.Flags().StringVarP(&opts.outputFmt, "output", "o", "yaml", "Output format (yaml, json, dir)")
	c.Flags().StringVar(&opts.outputDir, "output-dir", "", "Directory to write manifests (for -o dir)")
	c.Flags().StringVar(&opts.outputFile, "output-file", "", "File to write output (stdout if not specified)")

	return c
}

// runBuild renders the module.
func runBuild(ctx context.Context, opts *buildOptions) error {
	// Validate options
	if opts.outputFmt != "yaml" && opts.outputFmt != "json" && opts.outputFmt != "dir" {
		return fmt.Errorf("%w: invalid output format %q, use yaml, json, or dir", cmd.ErrValidation, opts.outputFmt)
	}

	if opts.outputFmt == "dir" && opts.outputDir == "" {
		return fmt.Errorf("%w: --output-dir required with -o dir", cmd.ErrValidation)
	}

	// Check directory exists
	if _, err := os.Stat(opts.dir); os.IsNotExist(err) {
		return fmt.Errorf("%w: directory %s", cmd.ErrNotFound, opts.dir)
	}

	// Load the module
	loader := opmcue.NewLoader()
	module, err := loader.LoadModule(ctx, opts.dir, opts.values)
	if err != nil {
		if errors.Is(err, opmcue.ErrModuleNotFound) {
			return fmt.Errorf("%w: %v", cmd.ErrNotFound, err)
		}
		if errors.Is(err, opmcue.ErrInvalidModule) {
			return fmt.Errorf("%w: %v", cmd.ErrValidation, err)
		}
		return fmt.Errorf("loading module: %w", err)
	}

	// Render manifests
	renderer := opmcue.NewRenderer()
	manifestSet, err := renderer.RenderModule(ctx, module)
	if err != nil {
		if errors.Is(err, opmcue.ErrNoManifests) {
			return fmt.Errorf("%w: no manifests found in module", cmd.ErrValidation)
		}
		if errors.Is(err, opmcue.ErrRenderFailed) {
			return fmt.Errorf("%w: %v", cmd.ErrValidation, err)
		}
		return fmt.Errorf("rendering manifests: %w", err)
	}

	// Sort for apply order
	manifestSet.SortForApply()

	// Output based on format
	switch opts.outputFmt {
	case "yaml":
		return outputYAML(manifestSet, opts.outputFile)
	case "json":
		return outputJSON(manifestSet, opts.outputFile)
	case "dir":
		return outputDir(manifestSet, opts.outputDir)
	}

	return nil
}

// outputYAML outputs manifests as YAML.
func outputYAML(ms *opmcue.ManifestSet, outputFile string) error {
	var out *os.File
	var err error

	if outputFile != "" {
		out, err = os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("creating output file: %w", err)
		}
		defer out.Close()
	} else {
		out = os.Stdout
	}

	for i, m := range ms.Manifests {
		if i > 0 {
			fmt.Fprintln(out, "---")
		}
		data, err := yaml.Marshal(m.Object.Object)
		if err != nil {
			return fmt.Errorf("marshaling manifest: %w", err)
		}
		out.Write(data)
	}

	return nil
}

// outputJSON outputs manifests as JSON.
func outputJSON(ms *opmcue.ManifestSet, outputFile string) error {
	var out *os.File
	var err error

	if outputFile != "" {
		out, err = os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("creating output file: %w", err)
		}
		defer out.Close()
	} else {
		out = os.Stdout
	}

	// Output as JSON array
	objects := make([]map[string]interface{}, len(ms.Manifests))
	for i, m := range ms.Manifests {
		objects[i] = m.Object.Object
	}

	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(objects)
}

// outputDir writes manifests to individual files in a directory.
func outputDir(ms *opmcue.ManifestSet, dir string) error {
	// Create directory
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	for i, m := range ms.Manifests {
		// Generate filename
		kind := m.Object.GetKind()
		name := m.Object.GetName()
		ns := m.Object.GetNamespace()

		filename := fmt.Sprintf("%03d-%s-%s", i, kind, name)
		if ns != "" {
			filename = fmt.Sprintf("%03d-%s-%s-%s", i, ns, kind, name)
		}
		filename += ".yaml"

		filepath := filepath.Join(dir, filename)

		data, err := yaml.Marshal(m.Object.Object)
		if err != nil {
			return fmt.Errorf("marshaling manifest: %w", err)
		}

		if err := os.WriteFile(filepath, data, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", filepath, err)
		}
	}

	fmt.Printf("Wrote %d manifests to %s\n", len(ms.Manifests), dir)
	return nil
}

func init() {
	// Reference output package to ensure it's imported
	_ = output.FormatYAML
}
