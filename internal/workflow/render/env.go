package render

import (
	"context"
	"errors"
	"fmt"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	opmlabels "github.com/open-platform-model/library/opm/k8s/labels"
	"github.com/open-platform-model/library/opm/kernel"
	libplatform "github.com/open-platform-model/library/opm/platform"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/platform"
)

// RuntimeName is the runtime identity the CLI injects into every kernel
// render (#context.#runtimeName): the library's managed-by value for the cli,
// the peer of the operator's opmlabels.ManagedByController.
const RuntimeName = opmlabels.ManagedByCLI

// The per-invocation kernel itself is constructed by config.NewKernel; this
// file holds the render environment built on top of it.

// renderEnv is the prepared per-invocation render environment: the kernel,
// the acquired (source-carrying) platform with its provenance, the skew
// policy the render runs under, and the caller's --skip-unprovided switch.
type renderEnv struct {
	kernel         *kernel.Kernel
	platform       *libplatform.Platform
	resolution     platform.Resolution
	skew           kernel.SkewPolicy
	skipUnprovided bool
}

// resolvePlatformEnv resolves the platform by precedence (--platform, the
// cluster Platform when sel.Cluster is set, then the render's own deps when
// sel.Deps is set), acquires the resolved module directory on the given
// kernel and reports provenance. It runs AFTER the instance is loaded and
// its values validated, so cheap validation failures surface before any
// platform/registry work. sel carries the command's sources; the config path
// and registry come from cfg. An unparseable committed module file is the
// render's validation failure, every other generation failure a general one.
//
// Acquisition is the build: a bad pin, a key-to-import mismatch or an
// unpublished catalog fails here naming the entry or dependency, identically
// for every source (0019:D5).
func resolvePlatformEnv(ctx context.Context, k *kernel.Kernel, cfg *config.GlobalConfig, sel platform.ResolveOptions) (*renderEnv, error) {
	sel.ConfigPath = cfg.ConfigPath
	sel.Registry = cfg.Registry
	dir, res, err := platform.Resolve(ctx, sel)
	if err != nil {
		if errors.Is(err, platform.ErrModuleDepsFile) {
			return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err}
		}
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err}
	}
	skew, skewNote := skewPolicyFor(res, cfg)
	output.Info(res.Describe() + skewNote)

	p, err := k.AcquirePlatformFromDir(ctx, dir)
	if err != nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("building platform module %s (source %s): %w", dir, res.Source, err)}
	}

	return &renderEnv{kernel: k, platform: p, resolution: res, skew: skew}, nil
}

// clusterSkewRefuse is the Platform CR's spec.skewPolicy value that refuses
// (the operator's SkewPolicyRefuse); anything else, including unset, warns.
const clusterSkewRefuse = "Refuse"

// skewPolicyFor chooses the kernel's skew policy (0019:D7/D18): when the
// cluster Platform CR is the source its spec.skewPolicy wins, so CLI and
// operator judge the same platform the same way; otherwise the config file's
// skewPolicy applies. Absent means warn on both. The returned note is
// appended to the provenance line: it names a refuse policy and its source,
// and names the CR as the policy's source when it overrode a config key
// that would have refused. A platform generated from the module's own deps
// pins every path at least at the render's own version, so skew cannot
// arise: the policy is not applied and not named.
func skewPolicyFor(res platform.Resolution, cfg *config.GlobalConfig) (policy kernel.SkewPolicy, note string) {
	if res.Source == platform.SourceModuleDeps {
		return kernel.SkewWarn, ""
	}
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
