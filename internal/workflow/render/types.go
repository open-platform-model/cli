package render

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/opmodel/cli/internal/config"
	pkgmodule "github.com/opmodel/cli/pkg/module"
	pkgrender "github.com/opmodel/cli/pkg/render"
)

// Result is the output of the shared render workflow.
type Result struct {
	Resources  []*unstructured.Unstructured
	Release    pkgmodule.ReleaseMetadata
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

type ReleaseFileOpts struct {
	ReleaseFilePath string
	ValuesFiles     []string
	ModulePath      string
	K8sConfig       *config.ResolvedKubernetesConfig
	Config          *config.GlobalConfig
}

type ShowOutputOpts struct {
	Verbose bool
}
