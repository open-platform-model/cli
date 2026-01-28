package mod

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/opmodel/cli/internal/cmd"
	"github.com/opmodel/cli/internal/output"
	"github.com/opmodel/cli/internal/render"
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
	verbose    bool
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
	c.Flags().BoolVarP(&opts.verbose, "verbose", "v", false, "Verbose output (show pipeline phases)")

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

	// Create render pipeline
	pipeline := render.NewPipeline(&render.Options{
		Dir:         opts.dir,
		ValuesFiles: opts.values,
		Verbose:     opts.verbose,
	})

	// Execute render pipeline
	result, err := pipeline.Render(ctx)
	if err != nil {
		// Check if we have partial results even with errors
		if result != nil && len(result.Manifests) > 0 {
			// We have some manifests, continue to output
			if !opts.verbose {
				fmt.Fprintf(os.Stderr, "Warning: render completed with errors: %v\n", err)
			}
		} else {
			return fmt.Errorf("%w: %v", cmd.ErrValidation, err)
		}
	}

	if len(result.Manifests) == 0 {
		return fmt.Errorf("%w: no manifests generated", cmd.ErrValidation)
	}

	// Output based on format
	switch opts.outputFmt {
	case "yaml":
		return outputYAMLFromManifests(result.Manifests, opts.outputFile)
	case "json":
		return outputJSONFromManifests(result.Manifests, opts.outputFile)
	case "dir":
		return outputDirFromManifests(result.Manifests, opts.outputDir)
	}

	return nil
}

// outputYAMLFromManifests outputs manifests as YAML.
func outputYAMLFromManifests(manifests []render.Manifest, outputFile string) error {
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

	for i, m := range manifests {
		if i > 0 {
			fmt.Fprintln(out, "---")
		}
		// Add source comment
		fmt.Fprintf(out, "# Source: %s/%s\n", m.TransformerID, m.ComponentName)
		data, err := yaml.Marshal(m.Object)
		if err != nil {
			return fmt.Errorf("marshaling manifest: %w", err)
		}
		out.Write(data)
	}

	return nil
}

// outputJSONFromManifests outputs manifests as JSON.
func outputJSONFromManifests(manifests []render.Manifest, outputFile string) error {
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
	objects := make([]map[string]interface{}, len(manifests))
	for i, m := range manifests {
		objects[i] = m.Object
	}

	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(objects)
}

// outputDirFromManifests writes manifests to individual files in a directory.
func outputDirFromManifests(manifests []render.Manifest, dir string) error {
	// Create directory
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	for i, m := range manifests {
		// Generate filename from object
		kind, _ := m.Object["kind"].(string)
		metadata, _ := m.Object["metadata"].(map[string]interface{})
		name, _ := metadata["name"].(string)
		ns, _ := metadata["namespace"].(string)

		filename := fmt.Sprintf("%03d-%s-%s", i, kind, name)
		if ns != "" {
			filename = fmt.Sprintf("%03d-%s-%s-%s", i, ns, kind, name)
		}
		filename += ".yaml"

		filePath := filepath.Join(dir, filename)

		// Add source comment
		comment := fmt.Sprintf("# Source: %s/%s\n", m.TransformerID, m.ComponentName)
		data, err := yaml.Marshal(m.Object)
		if err != nil {
			return fmt.Errorf("marshaling manifest: %w", err)
		}

		fullData := append([]byte(comment), data...)
		if err := os.WriteFile(filePath, fullData, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", filePath, err)
		}
	}

	fmt.Printf("Wrote %d manifests to %s\n", len(manifests), dir)
	return nil
}

func init() {
	// Reference output package to ensure it's imported
	_ = output.FormatYAML
}
