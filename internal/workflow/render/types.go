package render

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/modref"
	"github.com/open-platform-model/cli/internal/platform"
)

// Result is the output of the shared render workflow.
type Result struct {
	Resources []*unstructured.Unstructured
	// Instance is the acquired instance's metadata as the kernel decoded it,
	// the library's type; only Namespace may differ, when the --namespace
	// flag or env override applied. Was: Release (0002:D8/D9).
	Instance module.InstanceMetadata
	// Module is the embedded module's metadata, the library's type, decoded
	// from the instance package (the canonical spec.module reference is
	// derived from it by CanonicalModuleRef).
	Module module.ModuleMetadata

	// Pairs are the matched (component, transformer) pairs the render
	// evaluated, in build order — the kernel's diagnostics, shown by the
	// transformer-match output.
	Pairs []kernel.RenderPair

	// Warnings are the render's advisory facts, worded by the CLI from the
	// kernel's diagnostics rows (formatAdvisories): unhandled optional traits
	// and, under the warn skew policy, catalog version skew. Non-empty is
	// not failure; every entry is shown to the user.
	Warnings []string

	// Platform is the resolved platform-source provenance (0006:D21).
	Platform platform.Resolution

	// RenderDigest is the operator-parity render digest computed over the
	// kernel-rendered resources (CUE-value serialization, operator sort
	// order — see inventory.ComputeRenderDigest). Written verbatim to
	// status.lastAppliedRenderDigest so a future ownership transfer has a
	// recorded value to verify against (0006:D9/D30).
	RenderDigest string

	// Values is the single unified values blob the render consumed, decoded to
	// a JSON-shaped map. The apply workflow writes it verbatim to the
	// ModuleInstance CR's spec.values (0006:D19). Nil when the
	// instance carries no values or they could not be decoded.
	Values map[string]any

	// SourceLocal is the render-provenance signal (0006:D7): true
	// when the module bytes did not come from pure registry resolution — the
	// main module is a local directory, or its cue.mod/local-module.cue carries
	// a replaceWith. The apply workflow stamps
	// module-instance.opmodel.dev/source: local on the CR accordingly.
	SourceLocal bool

	// Skipped is every demand the kernel skipped under --skip-unprovided,
	// in build order (the kernel's rows, taken whole). Empty without the
	// flag. The apply workflow records them on the ModuleInstance.
	Skipped []kernel.SkippedDemand
}

func (r *Result) HasWarnings() bool {
	return len(r.Warnings) > 0
}

func (r *Result) ResourceCount() int {
	return len(r.Resources)
}

type InstanceFileOpts struct {
	// InstanceFilePath names the instance: a .cue file or its package
	// directory.
	InstanceFilePath string

	// ModuleCommand is the module command a module package is pointed at
	// when one is given in place of an instance ("opm module build" for
	// instance build). Empty points at the module command group.
	ModuleCommand string

	// ValuesFiles are -f files layered onto the instance package as kernel
	// values sources, in order.
	ValuesFiles []string

	// PlatformFlag is the --platform platform module directory (0006:D21;
	// highest platform-source precedence).
	PlatformFlag string
	// ClusterPlatform reads the cluster Platform CR. nil skips the cluster:
	// the render resolves --platform, else the instance package's own deps.
	ClusterPlatform platform.ClusterPlatformGetter
	// ClusterOptional makes any cluster read failure warn and fall back to
	// the deps instead of failing the render (instance build and vet).
	ClusterOptional bool

	// SkipUnprovided is --skip-unprovided: the kernel's switch to skip
	// provider-fulfilled demands nothing on the platform provides.
	SkipUnprovided bool

	K8sConfig *config.ResolvedKubernetesConfig
	Config    *config.GlobalConfig
}

// ModuleOpts configures rendering a module through the synthesis path (no
// instance.cue on disk): a local module-package directory (ModulePath) or a
// published module resolved from the registry (Published).
type ModuleOpts struct {
	// ModulePath is the directory containing the user's module CUE package.
	// Unused when Published is set.
	ModulePath string

	// Published, when set, is the resolved published module to acquire from
	// the registry instead of a local directory.
	Published *modref.Resolution

	// ValuesFiles, when non-empty, override the module's debugValues.
	ValuesFiles []string

	// Name overrides the synthetic metadata.name. Empty falls back to
	// "<module.metadata.name>-debug".
	Name string

	// PlatformFlag is the --platform platform module directory (0006:D21;
	// highest platform-source precedence).
	PlatformFlag string
	// ClusterPlatform reads the cluster Platform CR. nil (module build and
	// vet) skips the cluster: the render resolves --platform, else the
	// module's own deps. module apply sets it, and falls back to the deps
	// when the cluster has no readable Platform.
	ClusterPlatform platform.ClusterPlatformGetter

	// SkipUnprovided is --skip-unprovided: the kernel's switch to skip
	// provider-fulfilled demands nothing on the platform provides.
	SkipUnprovided bool

	K8sConfig *config.ResolvedKubernetesConfig
	Config    *config.GlobalConfig
}

type ShowOutputOpts struct {
	Verbose bool
}
