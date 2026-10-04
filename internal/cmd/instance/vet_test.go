package instance

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
)

// skipUnprovidedRegistry routes core and the catalogs to GHCR; the
// skip-unprovided fixture pins nothing on the testing domain.
const skipUnprovidedRegistry = "opmodel.dev=ghcr.io/open-platform-model,registry.cue.works"

// An OPM_NAMESPACE that disagrees with the instance file is refused by
// opm instance vet: the environment reaches the render through
// config.ResolveKubernetes, not only through a hand-built resolved config.
// Not parallel: it sets the environment.
func TestInstanceVet_OPMNamespaceOverrideRefused(t *testing.T) {
	if os.Getenv("OPM_SKIP_REGISTRY_TESTS") != "" {
		t.Skip("skipping registry-backed tests")
	}
	if _, err := config.NewKernel(skipUnprovidedRegistry).SchemaCache().Get(); err != nil {
		t.Skipf("core v2 schema unavailable (registry/cache): %v", err)
	}
	instanceDir, err := filepath.Abs(filepath.Join("..", "..", "workflow", "render", "testdata", "skip-unprovided", "instance"))
	require.NoError(t, err)

	t.Setenv("OPM_NAMESPACE", "staging")
	cfg := &config.GlobalConfig{ConfigPath: filepath.Join(t.TempDir(), "config.cue"), Registry: skipUnprovidedRegistry}

	err = runInstanceVet(instanceDir, cfg, &cmdutil.InstanceFileFlags{}, clusterLookup{offline: true}, "")
	require.Error(t, err)
	var exitErr *opmexit.ExitError
	require.True(t, errors.As(err, &exitErr), "want an ExitError, got %T: %v", err, err)
	assert.Equal(t, opmexit.ExitValidationError, exitErr.Code)
	for _, want := range []string{"OPM_NAMESPACE", `"staging"`, `"opm-skip-unprovided-itest"`} {
		assert.Contains(t, err.Error(), want)
	}
}
