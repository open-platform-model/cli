package render

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/cli/internal/config"
	pkgmodule "github.com/open-platform-model/cli/pkg/module"
	pkgrender "github.com/open-platform-model/cli/pkg/render"
)

// Result is the output of the shared render workflow.
type Result struct {
	Resources  []*unstructured.Unstructured
	Instance   pkgmodule.InstanceMetadata // Was: Release (enhancement 0002 D8/D9)
	Module     pkgmodule.ModuleMetadata
	Components []pkgrender.ComponentSummary
	MatchPlan  *pkgrender.MatchPlan
	Warnings   []string
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
	K8sConfig        *config.ResolvedKubernetesConfig
	Config           *config.GlobalConfig
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

	K8sConfig *config.ResolvedKubernetesConfig
	Config    *config.GlobalConfig
}

type ShowOutputOpts struct {
	Verbose bool
}
