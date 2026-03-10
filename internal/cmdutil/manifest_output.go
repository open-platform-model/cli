package cmdutil

import (
	"fmt"
	"os"
	"strings"

	opmexit "github.com/opmodel/cli/internal/exit"

	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ParseManifestOutputFormat validates output formats supported by manifest writers.
func ParseManifestOutputFormat(outputFmt string) (output.Format, error) {
	outputFormat, valid := output.ParseFormat(outputFmt)
	if !valid || !output.IsManifestFormat(outputFormat) {
		return "", &opmexit.ExitError{
			Code: opmexit.ExitGeneralError,
			Err:  fmt.Errorf("invalid output format %q (valid: yaml, json)", outputFmt),
		}
	}
	return outputFormat, nil
}

// WriteManifestOutput writes manifests either to stdout or split files.
func WriteManifestOutput(resources []*unstructured.Unstructured, outputFormat output.Format, split bool, outDir string, releaseName string) error {
	releaseLog := output.ReleaseLogger(releaseName)
	if split {
		splitOpts := output.SplitOptions{OutDir: outDir, Format: outputFormat}
		if err := output.WriteSplitManifests(resources, splitOpts); err != nil {
			return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("writing split manifests: %w", err)}
		}
		releaseLog.Info(fmt.Sprintf("wrote %d resources to %s", len(resources), outDir))
		return nil
	}

	manifestOpts := output.ManifestOptions{Format: outputFormat, Writer: os.Stdout}
	if err := output.WriteManifests(resources, manifestOpts); err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("writing manifests: %w", err)}
	}
	return nil
}

// FormatApplySummary builds a human-readable summary of apply results.
func FormatApplySummary(r *kubernetes.ApplyResult) string {
	var parts []string
	if r.Created > 0 {
		parts = append(parts, fmt.Sprintf("%d created", r.Created))
	}
	if r.Configured > 0 {
		parts = append(parts, fmt.Sprintf("%d configured", r.Configured))
	}
	if r.Unchanged > 0 {
		parts = append(parts, fmt.Sprintf("%d unchanged", r.Unchanged))
	}
	summary := fmt.Sprintf("applied %d resources successfully", r.Applied)
	if len(parts) > 0 {
		summary += fmt.Sprintf(" (%s)", strings.Join(parts, ", "))
	}
	return summary
}
