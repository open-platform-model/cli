package render

import (
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/engine"
	pkgmodule "github.com/opmodel/cli/pkg/module"
	pkgrender "github.com/opmodel/cli/pkg/render"
)

// Result is the output of the shared render workflow.
type Result struct {
	Resources  []*unstructured.Unstructured
	Release    pkgrender.ModuleReleaseMetadata
	Module     pkgmodule.ModuleMetadata
	Components []engine.ComponentSummary
	MatchPlan  *pkgrender.MatchPlan
	Warnings   []string
}

func (r *Result) HasWarnings() bool {
	return len(r.Warnings) > 0
}

func (r *Result) ResourceCount() int {
	return len(r.Resources)
}

type ReleaseOpts struct {
	Args        []string
	Values      []string
	ReleaseName string
	K8sConfig   *config.ResolvedKubernetesConfig
	Config      *config.GlobalConfig
	DebugValues bool
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

func hasReleaseFile(modulePath string) bool {
	dir := modulePath
	info, err := os.Stat(modulePath)
	if err == nil && !info.IsDir() {
		dir = filepath.Dir(modulePath)
	}
	_, statErr := os.Stat(filepath.Join(dir, "release.cue"))
	return statErr == nil
}
