package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/kernel"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/modref"
)

// The controller's instance renders under its fixed name and namespace,
// from the merged values only, against the module's own pins.
func TestModuleRenderOpts(t *testing.T) {
	cfg := &config.GlobalConfig{}
	k8s := &config.ResolvedKubernetesConfig{}
	res := &modref.Resolution{Path: ControllerModulePath, Version: "v0.1.0"}
	values := kernel.Source{Origin: "merged values"}

	opts := ModuleRenderOpts(cfg, k8s, res, values)

	assert.Same(t, res, opts.Published)
	assert.Equal(t, ControllerInstanceName, opts.Name)
	assert.Equal(t, ControllerNamespace, opts.Namespace)
	assert.True(t, opts.DepsOnly, "the render never reads a Platform")
	require.Len(t, opts.Values, 1)
	assert.Equal(t, "merged values", opts.Values[0].Origin)
	assert.Empty(t, opts.ValuesFiles)
	assert.Empty(t, opts.PlatformFlag)
	assert.Nil(t, opts.ClusterPlatform)
	assert.Same(t, cfg, opts.Config)
	assert.Same(t, k8s, opts.K8sConfig)
}
