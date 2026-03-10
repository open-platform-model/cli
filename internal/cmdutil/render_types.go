package cmdutil

import (
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/pkg/engine"
	pkgmodule "github.com/opmodel/cli/pkg/module"
	"github.com/opmodel/cli/pkg/modulerelease"
)

// RenderResult is the output of RenderRelease, combining engine output with
// release/module metadata for use across all cmd/mod commands.
type RenderResult struct {
	// Resources is the ordered list of Kubernetes resources, already converted
	// to *unstructured.Unstructured for direct use with inventory and k8s packages.
	Resources []*unstructured.Unstructured

	// Release contains release-level metadata (name, namespace, release UUID, labels).
	Release modulerelease.ReleaseMetadata

	// Module contains module-level metadata (canonical name, FQN, version, UUID).
	Module pkgmodule.ModuleMetadata

	// Components contains summary data for each component rendered in this release.
	// Sorted by component name for deterministic output.
	Components []engine.ComponentSummary

	// MatchPlan describes which transformers matched which components.
	// Used for verbose output and debugging.
	MatchPlan *engine.MatchPlan

	// Warnings contains non-fatal warnings (e.g. unhandled traits).
	Warnings []string
}

// HasWarnings returns true if there are warnings.
func (r *RenderResult) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// ResourceCount returns the number of rendered resources.
func (r *RenderResult) ResourceCount() int {
	return len(r.Resources)
}

// RenderReleaseOpts holds the inputs for RenderRelease.
type RenderReleaseOpts struct {
	// Args from the cobra command (first arg is module path).
	Args []string
	// Values files (-f flags).
	Values []string
	// ReleaseName overrides the default release name.
	ReleaseName string
	// K8sConfig is the pre-resolved Kubernetes configuration.
	// All fields (namespace, provider, kubeconfig, context) must already be resolved
	// via config.ResolveKubernetes before calling RenderRelease.
	K8sConfig *config.ResolvedKubernetesConfig
	// Config is the fully loaded global configuration.
	Config *config.GlobalConfig
	// DebugValues instructs RenderRelease to extract the module's debugValues field
	// and use it as the values source instead of a values file. Intended for
	// opm mod vet when no -f flag is provided.
	DebugValues bool
}

// RenderFromReleaseFileOpts holds inputs for RenderFromReleaseFile.
type RenderFromReleaseFileOpts struct {
	// ReleaseFilePath is the path to the .cue release file (required).
	// May be a directory, in which case release.cue inside it is used.
	ReleaseFilePath string
	// ValuesFiles are values CUE files (optional, from --values/-f).
	// When empty, values.cue next to the release file is used if it exists.
	// If neither is found and the release package has no concrete values field,
	// an error is returned.
	ValuesFiles []string
	// ModulePath is the path to a local module directory (optional, from --module).
	ModulePath string
	// K8sConfig is the pre-resolved Kubernetes configuration.
	K8sConfig *config.ResolvedKubernetesConfig
	// Config is the fully loaded global configuration.
	Config *config.GlobalConfig
}

// hasReleaseFile reports whether a release.cue file exists inside modulePath.
// modulePath may be a directory (checked directly) or a file path (its parent
// directory is checked). Returns false on any stat error.
func hasReleaseFile(modulePath string) bool {
	dir := modulePath
	info, err := os.Stat(modulePath)
	if err == nil && !info.IsDir() {
		dir = filepath.Dir(modulePath)
	}
	_, statErr := os.Stat(filepath.Join(dir, "release.cue"))
	return statErr == nil
}
