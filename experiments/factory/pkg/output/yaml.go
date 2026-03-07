// Package output provides shared rendering helpers for the factory pipeline.
//
// It is intentionally thin: callers own iteration order and error handling.
// The helpers here handle only the formatting and writing concerns.
package output

import (
	"fmt"
	"io"

	cueyaml "cuelang.org/go/encoding/yaml"

	"github.com/opmodel/cli/experiments/factory/pkg/core"
)

// PrintResources writes each resource as a YAML document to w.
//
// Each document is preceded by a provenance comment showing the release,
// component, and transformer that produced it, and followed by a "---"
// separator. The function returns the first encoding error encountered.
//
// The Release field on each resource is included in the comment only when
// non-empty, making the output clean for single-release (ModuleRelease)
// rendering while still being informative for multi-release (BundleRelease)
// rendering.
func PrintResources(w io.Writer, resources []*core.Resource) error {
	for _, res := range resources {
		if res.Release != "" {
			fmt.Fprintf(w, "# release: %s\n# component: %s\n# transformer: %s\n",
				res.Release, res.Component, res.Transformer)
		} else {
			fmt.Fprintf(w, "# component: %s\n# transformer: %s\n",
				res.Component, res.Transformer)
		}

		yamlBytes, err := cueyaml.Encode(res.Value)
		if err != nil {
			return fmt.Errorf("encoding resource as YAML (release=%q component=%q transformer=%q): %w",
				res.Release, res.Component, res.Transformer, err)
		}

		fmt.Fprintf(w, "%s---\n", yamlBytes)
	}
	return nil
}

// PrintWarnings writes each warning as a prefixed line to w.
func PrintWarnings(w io.Writer, warnings []string) {
	for _, warning := range warnings {
		fmt.Fprintf(w, "WARNING: %s\n", warning)
	}
}
