package handoff

import (
	"context"
	"fmt"
	"os"

	"github.com/open-platform-model/library/opm/helper/synth"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/materialize"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/platform"
	workflowrender "github.com/open-platform-model/cli/internal/workflow/render"
	pkgcore "github.com/open-platform-model/cli/pkg/core"
)

// cueCacheDirEnv is the CUE module cache location. The verification render
// overrides it per invocation so a previously cached artifact cannot answer for
// what the registry currently serves.
const cueCacheDirEnv = "CUE_CACHE_DIR"

// VerificationInput describes the render to reproduce: the CR's own module
// reference and values, rendered against the cluster Platform.
type VerificationInput struct {
	Client *kubernetes.Client
	Config *config.GlobalConfig

	Name      string
	Namespace string

	ModulePath    string
	ModuleVersion string

	// SpecValues is the CR's spec.values — the values the deployed render
	// consumed, replayed here verbatim.
	SpecValues map[string]any
}

// VerificationDigest renders the instance strictly from the registry and
// returns the render digest, computed with the same algorithm that produced the
// CR's status.lastAppliedRenderDigest.
//
// "Strictly from the registry" is the whole point of this function
// (enhancement 0006 D38), and it means two specific things:
//
//   - Acquisition goes through the registry loader, never the working
//     directory's module context. No cue.mod/local-module.cue can inject a
//     replaceWith, because no local module is discovered at all.
//   - A fresh, per-invocation CUE cache directory is used. The shared cache is
//     keyed by module@version and is registry-blind, so a republished
//     same-version artifact would otherwise be satisfied by stale bytes — the
//     exact trap the verification exists to catch.
//
// The render passes the CLI's own runtime name, so the digest is comparable
// with the CLI-recorded one. It is never compared against an operator-written
// digest (D40).
func VerificationDigest(ctx context.Context, in VerificationInput) (string, error) {
	restore, err := isolateCUECache()
	if err != nil {
		return "", err
	}
	defer restore()

	k := workflowrender.NewKernel(in.Config)

	mod, err := k.AcquireModuleFromRegistry(ctx, in.ModulePath, in.ModuleVersion)
	if err != nil {
		return "", fmt.Errorf(
			"cannot resolve module %s@%s from the registry — the operator would fail the same way; publish the module (or correct spec.module) before handing off: %w",
			in.ModulePath, in.ModuleVersion, err)
	}

	// The CR's spec.values replayed as the synthesis input. An instance with no
	// values yields an empty struct rather than a missing one, matching what
	// the recorded render consumed.
	values := k.CueContext().Encode(map[string]any{})
	if len(in.SpecValues) > 0 {
		values = k.CueContext().Encode(in.SpecValues)
	}
	if err := values.Err(); err != nil {
		return "", fmt.Errorf("encoding spec.values for the verification render: %w", err)
	}

	inst, err := k.SynthesizeInstance(ctx, synth.InstanceInput{
		Module:    mod,
		Name:      in.Name,
		Namespace: in.Namespace,
		Values:    values,
	})
	if err != nil {
		return "", fmt.Errorf("verification render: synthesizing instance %q from the published module: %w", in.Name, err)
	}

	mp, err := materializeClusterPlatform(ctx, k, in)
	if err != nil {
		return "", err
	}

	out, err := k.Compile(ctx, kernel.CompileInput{
		ModuleInstance: inst,
		Platform:       mp,
		RuntimeName:    workflowrender.RuntimeName,
	})
	if err != nil {
		return "", fmt.Errorf("verification render: compiling instance %q: %w", in.Name, err)
	}

	resources := make([]*pkgcore.Resource, 0, len(out.Compiled))
	for _, c := range out.Compiled {
		resources = append(resources, &pkgcore.Resource{
			Value:       c.Value,
			Instance:    c.Instance,
			Component:   c.Component,
			Transformer: c.Transformer,
		})
	}

	digest, err := inventory.ComputeRenderDigest(resources)
	if err != nil {
		return "", fmt.Errorf("computing the verification render digest: %w", err)
	}
	return digest, nil
}

// materializeClusterPlatform resolves the platform for a verification render.
// The cluster Platform is the only permitted source (0006 D11): there is no
// --platform flag on handoff, and unlike every other render path a failure to
// read the cluster Platform is fatal rather than a fall back to the local
// default — handoff is the one admin-adjacent operation (D17), and verifying
// against a platform the operator will not use proves nothing.
func materializeClusterPlatform(ctx context.Context, k *kernel.Kernel, in VerificationInput) (*materialize.MaterializedPlatform, error) {
	spec, name, unavailable, err := platform.ClusterSpecGetterFor(in.Client.Dynamic)(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading the cluster Platform for verification: %w", err)
	}
	if unavailable != "" {
		return nil, fmt.Errorf(
			"handoff requires the cluster Platform, which is unavailable (%s) — the operator renders against it, so verifying against anything else proves nothing; install the operator's Platform (or grant read access) before handing off",
			unavailable)
	}

	input, err := platform.DecodeCRSpec(spec, name)
	if err != nil {
		return nil, fmt.Errorf("decoding the cluster Platform %q: %w", name, err)
	}
	output.Debug("verification render platform", "source", "cluster", "name", name)

	mp, err := platform.Materialize(ctx, k, input)
	if err != nil {
		return nil, fmt.Errorf("materializing the cluster Platform %q for verification: %w", name, err)
	}
	return mp, nil
}

// isolateCUECache points the CUE module cache at a fresh temporary directory
// for the duration of the verification render, and returns a function that
// restores the previous setting and removes the directory.
func isolateCUECache() (restore func(), err error) {
	dir, err := os.MkdirTemp("", "opm-handoff-verify-*")
	if err != nil {
		return nil, fmt.Errorf("creating an isolated CUE cache directory: %w", err)
	}

	previous, had := os.LookupEnv(cueCacheDirEnv)
	if err := os.Setenv(cueCacheDirEnv, dir); err != nil {
		os.RemoveAll(dir)
		return nil, fmt.Errorf("isolating the CUE cache directory: %w", err)
	}

	return func() {
		if had {
			os.Setenv(cueCacheDirEnv, previous)
		} else {
			os.Unsetenv(cueCacheDirEnv)
		}
		if err := os.RemoveAll(dir); err != nil {
			output.Debug("could not remove the isolated CUE cache directory", "dir", dir, "error", err)
		}
	}, nil
}
