package controller

import (
	"github.com/open-platform-model/library/opm/kernel"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/modref"
	workflowrender "github.com/open-platform-model/cli/internal/workflow/render"
)

// ModuleRenderOpts are the render options of the controller's instance: the
// resolved module, the one merged values source, the fixed instance name and
// namespace, and the module's own dependency pins (never a Platform). Install
// and the pin task both render through them, so the CRDs the pin embeds are
// the CRDs install applies.
func ModuleRenderOpts(cfg *config.GlobalConfig, k8sConfig *config.ResolvedKubernetesConfig, res *modref.Resolution, values kernel.Source) workflowrender.ModuleOpts {
	return workflowrender.ModuleOpts{
		Published: res,
		Values:    []kernel.Source{values},
		Name:      ControllerInstanceName,
		Namespace: ControllerNamespace,
		DepsOnly:  true,
		K8sConfig: k8sConfig,
		Config:    cfg,
	}
}
