package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"

	"github.com/opmodel/cli/internal/builder"
	"github.com/opmodel/cli/internal/core"
	"github.com/opmodel/cli/internal/core/module"
	coreprovider "github.com/opmodel/cli/internal/core/provider"
	"github.com/opmodel/cli/internal/core/transformer"
	"github.com/opmodel/cli/internal/loader"
	"github.com/opmodel/cli/internal/output"
)

// pipeline implements the Pipeline interface.
// It orchestrates module loading, provider loading, release building,
// component matching, and transformer execution using the phase packages.
type pipeline struct {
	cueCtx    *cue.Context
	providers map[string]cue.Value
	registry  string
}

// NewPipeline creates a new Pipeline implementation.
//
// cueCtx is the shared CUE evaluation context. If nil, a fresh context is
// created. The same context must be used across all phase packages to avoid
// cross-runtime panics.
//
// providers maps provider names to their CUE values (from config.Providers).
// registry is the CUE registry URL (from config.Registry).
func NewPipeline(cueCtx *cue.Context, providers map[string]cue.Value, registry string) Pipeline {
	if cueCtx == nil {
		cueCtx = cuecontext.New()
	}
	return &pipeline{
		cueCtx:    cueCtx,
		providers: providers,
		registry:  registry,
	}
}

// Render executes the pipeline and returns results.
//
// Phase sequence:
//  1. PREPARATION:    loader.LoadModule() → *core.Module
//  2. PROVIDER LOAD:  loader.LoadProvider() → *core.Provider
//  3. BUILD:          builder.Build() → *core.ModuleRelease; then ValidateValues + Validate
//  4. MATCHING:       core.Provider.Match() → *core.TransformerMatchPlan
//  5. GENERATE:       matchPlan.Execute() → []*core.Resource + []error
//
// Fatal errors from phases 1-4 return (nil, err). Generate errors land in
// RenderResult.Errors. Context cancellation during GENERATE is fatal.
func (p *pipeline) Render(ctx context.Context, opts RenderOptions) (*RenderResult, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	mod, releaseName, namespace, err := p.prepare(opts)
	if err != nil {
		return nil, err
	}

	// Phase 2: PROVIDER LOAD — parse transformers and metadata from the provider CUE value.
	coreProvider, err := loader.LoadProvider(p.cueCtx, opts.Provider, p.providers)
	if err != nil {
		return nil, err
	}

	// Phase 3: BUILD — inject module + values into #ModuleRelease via FillPath (Approach C).
	rel, err := builder.Build(p.cueCtx, mod, builder.Options{
		Name:      releaseName,
		Namespace: namespace,
	}, opts.Values)
	if err != nil {
		return nil, err
	}
	if err := rel.ValidateValues(); err != nil {
		return nil, err
	}
	if err := rel.Validate(); err != nil {
		return nil, err
	}

	output.Debug("release built",
		"name", rel.Metadata.Name,
		"namespace", rel.Metadata.Namespace,
		"components", len(rel.Components),
	)

	// Phase 4: MATCHING — match all components against provider transformers.
	matchPlan := coreProvider.Match(rel.Components)

	errs := collectUnmatchedErrors(matchPlan, coreProvider)
	errs, warnings := collectMatchWarnings(matchPlan, errs, opts.Strict)

	// Phase 5: GENERATE — execute transformers via TransformerMatchPlan.Execute().
	resources, execErrs := matchPlan.Execute(ctx, rel)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	errs = append(errs, execErrs...)

	sortResources(resources)

	// Ensure non-nil slices for consistent consumer behavior.
	if resources == nil {
		resources = make([]*core.Resource, 0)
	}
	if errs == nil {
		errs = make([]error, 0)
	}
	if warnings == nil {
		warnings = make([]string, 0)
	}

	return &RenderResult{
		Resources: resources,
		Release:   *rel.Metadata,
		Module:    *rel.Module.Metadata,
		MatchPlan: matchPlan.ToLegacyMatchPlan(),
		Errors:    errs,
		Warnings:  warnings,
	}, nil
}

// prepare runs Phase 1 (PREPARATION): loads the module, validates it,
// checks for values.cue if no --values flags were provided, logs the
// values source, and resolves the release name + namespace.
func (p *pipeline) prepare(opts RenderOptions) (*module.Module, string, string, error) { //nolint:gocritic // unnamedResult: named returns would shadow inner err vars
	mod, err := loader.LoadModule(p.cueCtx, opts.ModulePath, p.registry)
	if err != nil {
		return nil, "", "", err
	}
	if err := mod.Validate(); err != nil {
		return nil, "", "", err
	}

	// When no --values flags are provided, values.cue must exist on disk.
	if len(opts.Values) == 0 {
		valuesPath := filepath.Join(mod.ModulePath, "values.cue")
		if _, statErr := os.Stat(valuesPath); os.IsNotExist(statErr) {
			return nil, "", "", fmt.Errorf("values.cue not found in %s — provide values via values.cue or --values flag", mod.ModulePath)
		}
	}

	if len(opts.Values) > 0 {
		names := make([]string, len(opts.Values))
		for i, vf := range opts.Values {
			names[i] = filepath.Base(vf)
		}
		output.Debug("using values files", "files", strings.Join(names, ", "))
	} else {
		output.Debug("using default values.cue")
	}

	releaseName := opts.Name
	if releaseName == "" {
		releaseName = mod.Metadata.Name
	}
	namespace := resolveNamespace(opts.Namespace, mod.Metadata.DefaultNamespace)
	if namespace == "" {
		return nil, "", "", &NamespaceRequiredError{ModuleName: mod.Metadata.Name}
	}

	return mod, releaseName, namespace, nil
}

// collectUnmatchedErrors returns an error for each component that had no
// matching transformer.
func collectUnmatchedErrors(matchPlan *transformer.TransformerMatchPlan, cp *coreprovider.Provider) []error {
	errs := make([]error, 0, len(matchPlan.Unmatched))
	for _, compName := range matchPlan.Unmatched {
		errs = append(errs, &UnmatchedComponentError{
			ComponentName: compName,
			Available:     cp.Requirements(),
		})
	}
	return errs
}

// collectMatchWarnings collects unhandled-trait warnings from the match plan.
// In strict mode warnings are promoted to errors; otherwise they are non-fatal.
func collectMatchWarnings(matchPlan *transformer.TransformerMatchPlan, errs []error, strict bool) ([]error, []string) { //nolint:gocritic // unnamedResult: naming (errors, warnings) adds no clarity over the types
	rawWarnings := transformer.CollectWarnings(matchPlan)
	var warnings []string
	if strict {
		for _, w := range rawWarnings {
			errs = append(errs, fmt.Errorf("%s", w)) //nolint:err113 // warning-as-error; no typed error needed here
		}
	} else {
		warnings = rawWarnings
	}
	return errs, warnings
}

// sortResources sorts resources with a deterministic 5-key total ordering:
// weight → group → kind → namespace → name.
// This matches the digest sort in internal/inventory, making opm mod build
// output deterministic for equal-weight resources.
func sortResources(resources []*core.Resource) {
	sort.SliceStable(resources, func(i, j int) bool {
		ri, rj := resources[i], resources[j]
		wi, wj := core.GetWeight(ri.GVK()), core.GetWeight(rj.GVK())
		if wi != wj {
			return wi < wj
		}
		gi, gj := ri.GVK().Group, rj.GVK().Group
		if gi != gj {
			return gi < gj
		}
		ki, kj := ri.GVK().Kind, rj.GVK().Kind
		if ki != kj {
			return ki < kj
		}
		nsi, nsj := ri.Namespace(), rj.Namespace()
		if nsi != nsj {
			return nsi < nsj
		}
		return ri.Name() < rj.Name()
	})
}

// resolveNamespace resolves the target namespace using precedence:
//  1. --namespace flag (highest)
//  2. module.metadata.defaultNamespace
func resolveNamespace(flagValue, defaultNamespace string) string {
	if flagValue != "" {
		return flagValue
	}
	return defaultNamespace
}
