package render

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/library/opm/compile"
	"github.com/open-platform-model/library/opm/kernel"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/platform"
	pkgmodule "github.com/open-platform-model/cli/pkg/module"
)

// Result is the output of the shared render workflow.
type Result struct {
	Resources  []*unstructured.Unstructured
	Instance   pkgmodule.InstanceMetadata // Was: Release (enhancement 0002 D8/D9)
	Module     pkgmodule.ModuleMetadata
	Components []compile.ComponentSummary
	MatchPlan  *kernel.MatchPlan
	Warnings   []string

	// Platform is the resolved platform-source provenance (0006 D21). The
	// apply workflow uses it for the D12 write-if-absent decision.
	Platform platform.Resolution

	// RenderDigest is the operator-parity render digest computed over the
	// kernel-compiled resources (CUE-value serialization, operator sort
	// order — see inventory.ComputeRenderDigest). Written verbatim to
	// status.lastAppliedRenderDigest; the D7.4 handoff verification
	// compares against it (0006 D9/D30).
	RenderDigest string

	// Values is the single unified values blob the render consumed, decoded to
	// a JSON-shaped map. The apply workflow writes it verbatim to the
	// ModuleInstance CR's spec.values (enhancement 0006 D19). Nil when the
	// instance carries no values or they could not be decoded.
	Values map[string]any

	// SourceLocal is the render-provenance signal (enhancement 0006 D7): true
	// when the module bytes did not come from pure registry resolution — the
	// main module is a local directory, or its cue.mod/local-module.cue carries
	// a replaceWith. The apply workflow stamps
	// module-instance.opmodel.dev/source: local on the CR accordingly.
	SourceLocal bool
}

func (r *Result) HasWarnings() bool {
	return len(r.Warnings) > 0
}

func (r *Result) ResourceCount() int {
	return len(r.Resources)
}

type InstanceFileOpts struct {
	InstanceFilePath string
	ValuesFiles      []string

	// PlatformFlag is the --platform local override file (0006 D21).
	PlatformFlag string
	// ClusterPlatform reads the cluster Platform CR spec. nil marks the
	// command offline: the cluster is never consulted (D17/D21).
	ClusterPlatform platform.ClusterSpecGetter

	K8sConfig *config.ResolvedKubernetesConfig
	Config    *config.GlobalConfig
}

// ModuleOpts configures rendering from a module-package directory through the
// synthesis path (no instance.cue on disk).
type ModuleOpts struct {
	// ModulePath is the directory containing the user's module CUE package.
	ModulePath string

	// ValuesFiles, when non-empty, override the module's debugValues.
	ValuesFiles []string

	// Name overrides the synthetic metadata.name. Empty falls back to
	// "<module.metadata.name>-debug".
	Name string

	// PlatformFlag is the --platform local override file (0006 D21).
	PlatformFlag string
	// ClusterPlatform reads the cluster Platform CR spec. nil marks the
	// command offline: the cluster is never consulted (D17/D21).
	ClusterPlatform platform.ClusterSpecGetter

	K8sConfig *config.ResolvedKubernetesConfig
	Config    *config.GlobalConfig
}

type ShowOutputOpts struct {
	Verbose bool
}
