//go:build ignore

// Render-digest parity check (enhancement 0006 D30 gate, slice C2 task 4.2).
//
// Renders the SAME module two ways and requires byte-identical render digests:
//
//	Path A — the CLI workflow (FromModule): local module directory, staged as
//	         the build's main module, synthesized and compiled by the kernel.
//	Path B — the operator's call sequence: AcquireModuleFromRegistry +
//	         SynthesizeInstance + Compile (mirroring KernelModuleRenderer),
//	         digested with the same inventory.ComputeRenderDigest.
//
// Both paths run with RuntimeName "opm-cli": the runtime identity is stamped
// into rendered labels (app.kubernetes.io/managed-by), so a cross-actor
// comparison with different runtime names differs by construction — the
// per-actor label is the KNOWN delta, load-path equivalence is what this
// check proves (local staging ≡ registry acquisition; D37/D6).
//
// Requires: registry serving opmodel.dev/modules/test/podinfo@v0 at the
// fixture's version, the catalogs, and core — SKIPs otherwise unless
// OPM_ITEST_RENDER_PARITY=1 forces a hard failure.
//
// Run with: go run tests/integration/render-parity/main.go
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/helper/synth"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/schema"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/platform"
	workflowrender "github.com/open-platform-model/cli/internal/workflow/render"
	pkgcore "github.com/open-platform-model/cli/pkg/core"
)

const (
	// moduleDir is repo-local (vendored from opm-operator — see the fixture's
	// module.cue header) so render-parity runs in a standalone clone with no
	// sibling checkout. Resolved relative to the repo root, the program's cwd.
	moduleDir     = "tests/fixtures/modules/podinfo"
	modulePath    = "opmodel.dev/modules/test/podinfo@v0"
	moduleVersion = "v0.1.3"
	instName      = "podinfo-parity"
	instNamespace = "default"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
}

func skipOrFail(format string, args ...any) error {
	if os.Getenv("OPM_ITEST_RENDER_PARITY") == "1" {
		return fmt.Errorf(format, args...)
	}
	fmt.Printf("SKIP: "+format+"\n", args...)
	return nil
}

func run() error {
	registry := os.Getenv("OPM_REGISTRY")
	if registry == "" {
		registry = os.Getenv("CUE_REGISTRY")
	}
	if registry == "" {
		fmt.Println("SKIP: neither OPM_REGISTRY nor CUE_REGISTRY is set")
		return nil
	}

	absModuleDir, err := filepath.Abs(moduleDir)
	if err != nil {
		return err
	}
	if _, statErr := os.Stat(absModuleDir); statErr != nil {
		return skipOrFail("podinfo fixture not found at %s", absModuleDir)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	// Seed a temp ~/.opm (config + default platform) for the CLI path.
	dir, err := os.MkdirTemp("", "opm-render-parity-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	configPath := filepath.Join(dir, "config.cue")
	if err := os.WriteFile(configPath, []byte(config.DefaultConfigTemplate), 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "platform.cue"), []byte(config.DefaultPlatformTemplate), 0o600); err != nil {
		return err
	}

	cfg := &config.GlobalConfig{Registry: registry, ConfigPath: configPath}

	// ── Path A: CLI workflow (local module directory) ─────────────────────
	resultA, err := workflowrender.FromModule(ctx, workflowrender.ModuleOpts{
		ModulePath: absModuleDir,
		Name:       instName,
		K8sConfig:  &config.ResolvedKubernetesConfig{},
		Config:     cfg,
	})
	if err != nil {
		return skipOrFail("CLI render path failed (registry/catalogs unavailable?): %v", err)
	}
	digestA := resultA.RenderDigest
	fmt.Printf("path A (CLI workflow, local dir):        %s (%d resources)\n", digestA, len(resultA.Resources))

	// ── Path B: operator call sequence (registry acquisition) ─────────────
	k := kernel.New(
		kernel.WithRegistry(registry),
		kernel.WithSchemaLoader(schema.OCILoader{Registry: registry}),
	)

	// Resolve + materialize the same default platform through the shared path.
	in, _, err := platform.Resolve(ctx, platform.ResolveOptions{ConfigPath: configPath})
	if err != nil {
		return err
	}
	mp, err := platform.Materialize(ctx, k, in)
	if err != nil {
		return skipOrFail("platform materialize failed: %v", err)
	}

	mod, err := k.AcquireModuleFromRegistry(ctx, modulePath, moduleVersion)
	if err != nil {
		return skipOrFail("module %s@%s not acquirable from %s: %v", modulePath, moduleVersion, registry, err)
	}

	// Same values source the CLI path used: the module's debugValues.
	debugValues := mod.Package.LookupPath(cue.ParsePath("debugValues"))
	if !debugValues.Exists() {
		return fmt.Errorf("acquired module has no debugValues")
	}

	inst, err := k.SynthesizeInstance(ctx, synth.InstanceInput{
		Module:    mod,
		Name:      instName,
		Namespace: instNamespace,
		Values:    debugValues,
	})
	if err != nil {
		return fmt.Errorf("operator-path synthesis: %w", err)
	}

	out, err := k.Compile(ctx, kernel.CompileInput{
		ModuleInstance: inst,
		Platform:       mp,
		RuntimeName:    workflowrender.RuntimeName, // held constant — see file comment
	})
	if err != nil {
		return fmt.Errorf("operator-path compile: %w", err)
	}

	resources := make([]*pkgcore.Resource, 0, len(out.Compiled))
	for _, c := range out.Compiled {
		resources = append(resources, &pkgcore.Resource{
			Value: c.Value, Instance: c.Instance, Component: c.Component, Transformer: c.Transformer,
		})
	}
	digestB, err := inventory.ComputeRenderDigest(resources)
	if err != nil {
		return err
	}
	fmt.Printf("path B (operator sequence, registry):    %s (%d resources)\n", digestB, len(resources))

	if digestA != digestB {
		return fmt.Errorf("render digests differ: local-dir staging vs registry acquisition produced different bytes — check for content drift between %s and %s@%s, then suspect the load paths", absModuleDir, modulePath, moduleVersion)
	}

	fmt.Println("PASS: render-parity — local staging and registry acquisition are byte-identical")
	return nil
}
