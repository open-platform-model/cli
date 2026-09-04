package render

import (
	"context"
	"fmt"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	loaderfile "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/kernel"
	libplatform "github.com/open-platform-model/library/opm/platform"
	"github.com/open-platform-model/library/opm/schema"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/platform"
)

// RuntimeName is the runtime identity the CLI injects into every kernel
// render (#context.#runtimeName) — the peer of the operator's "opm-controller".
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

// renderEnv is the prepared per-invocation render environment: the kernel,
// the acquired (source-carrying) platform with its provenance, the seed
// document decoded from it and the skew policy the render runs under.
type renderEnv struct {
	kernel     *kernel.Kernel
	platform   *libplatform.Platform
	resolution platform.Resolution
	// spec is decoded from the built platform the render consumes
	// (platform.SpecFromPlatform) — carried onto Result so the apply
	// workflow can seed the cluster Platform without re-reading the module
	// (no second I/O, no TOCTOU).
	spec platform.Spec
	skew kernel.SkewPolicy
}

// resolvePlatformEnv resolves the platform by precedence (D11/D21), acquires
// the resolved module directory on the given kernel and reports provenance.
// It runs AFTER the instance is loaded and its values validated, so cheap
// validation failures surface before any platform/registry work.
// clusterGetter is nil for offline commands (build/render — D17: they never
// read the cluster).
//
// Acquisition is the build: a bad pin, a key-to-import mismatch or an
// unpublished catalog fails here naming the entry or dependency, identically
// for the flag, cluster and local sources (0019 D5).
func resolvePlatformEnv(ctx context.Context, k *kernel.Kernel, cfg *config.GlobalConfig, platformFlag string, clusterGetter platform.ClusterSpecGetter) (*renderEnv, error) {
	dir, res, err := platform.Resolve(ctx, platform.ResolveOptions{
		PlatformFlag: platformFlag,
		ConfigPath:   cfg.ConfigPath,
		Cluster:      clusterGetter,
		Registry:     cfg.Registry,
	})
	if err != nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err}
	}
	skew, skewNote := skewPolicyFor(res, cfg)
	output.Info(res.Describe() + skewNote)

	p, err := k.AcquirePlatformFromDir(ctx, dir, loaderfile.LoadOptions{Registry: cfg.Registry})
	if err != nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("building platform module %s (source %s): %w", dir, res.Source, err)}
	}
	spec, err := platform.SpecFromPlatform(p)
	if err != nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("reading platform module %s (source %s): %w", dir, res.Source, err)}
	}

	return &renderEnv{kernel: k, platform: p, resolution: res, spec: spec, skew: skew}, nil
}

// clusterSkewRefuse is the Platform CR's spec.skewPolicy value that refuses
// (the operator's SkewPolicyRefuse); anything else, including unset, warns.
const clusterSkewRefuse = "Refuse"

// skewPolicyFor chooses the kernel's skew policy (0019 D7/D18): when the
// cluster Platform CR is the source its spec.skewPolicy wins, so CLI and
// operator judge the same platform the same way; otherwise the config file's
// skewPolicy applies. Absent means warn on both. The returned note is
// appended to the provenance line: it names a refuse policy and its source,
// and names the CR as the policy's source when it overrode a config key
// that would have refused.
func skewPolicyFor(res platform.Resolution, cfg *config.GlobalConfig) (policy kernel.SkewPolicy, note string) {
	if res.Source == platform.SourceClusterCR {
		if res.SkewPolicy == clusterSkewRefuse {
			return kernel.SkewRefuse, ", skew policy: refuse (cluster Platform)"
		}
		if cfg.SkewPolicy == config.SkewPolicyRefuse {
			return kernel.SkewWarn, ", skew policy: warn (cluster Platform overrides config skewPolicy)"
		}
		return kernel.SkewWarn, ""
	}
	if cfg.SkewPolicy == config.SkewPolicyRefuse {
		return kernel.SkewRefuse, ", skew policy: refuse (config)"
	}
	return kernel.SkewWarn, ""
}
