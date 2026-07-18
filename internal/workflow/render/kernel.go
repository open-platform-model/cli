package render

import (
	"context"
	"fmt"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/materialize"
	"github.com/open-platform-model/library/opm/schema"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/platform"
)

// RuntimeName is the runtime identity the CLI injects into every kernel
// compile (#context.runtime) — the peer of the operator's "opm-controller".
const RuntimeName = "opm-cli"

// NewKernel constructs the per-invocation library kernel (design LD1): one
// Kernel per command, owning the CUE context and schema cache. The resolved
// registry threads into module acquisition AND the schema OCILoader — the
// latter otherwise reads only the process CUE_REGISTRY.
func NewKernel(cfg *config.GlobalConfig) *kernel.Kernel {
	return kernel.New(
		kernel.WithRegistry(cfg.Registry),
		kernel.WithSchemaLoader(schema.OCILoader{Registry: cfg.Registry}),
	)
}

// renderEnv is the prepared per-invocation render environment: the kernel and
// the materialized platform with its provenance.
type renderEnv struct {
	kernel     *kernel.Kernel
	platform   *materialize.MaterializedPlatform
	resolution platform.Resolution
}

// resolvePlatformEnv resolves the platform by precedence (D11/D21), reports
// provenance, and materializes it on the given kernel. It runs AFTER the
// instance is loaded and its values validated, so cheap validation failures
// surface before any platform/registry work. clusterGetter is nil for
// offline commands (build/render — D17: they never read the cluster).
func resolvePlatformEnv(ctx context.Context, k *kernel.Kernel, cfg *config.GlobalConfig, platformFlag string, clusterGetter platform.ClusterSpecGetter) (*renderEnv, error) {
	in, res, err := platform.Resolve(ctx, platform.ResolveOptions{
		PlatformFlag: platformFlag,
		ConfigPath:   cfg.ConfigPath,
		Cluster:      clusterGetter,
	})
	if err != nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err}
	}
	output.Info(res.Describe())

	mp, err := platform.Materialize(ctx, k, in)
	if err != nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("materializing platform (source %s): %w", res.Source, err)}
	}

	return &renderEnv{kernel: k, platform: mp, resolution: res}, nil
}
