// Package build provides the render pipeline implementation.
package build

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"

	"github.com/opmodel/cli/internal/build/component"
	"github.com/opmodel/cli/internal/build/module"
	"github.com/opmodel/cli/internal/build/release"
	"github.com/opmodel/cli/internal/build/transform"
	"github.com/opmodel/cli/internal/core"
	"github.com/opmodel/cli/internal/output"
)

// pipeline implements the Pipeline interface.
// It orchestrates module loading, release building, provider loading,
// component matching, and transformer execution.
type pipeline struct {
	cueCtx         *cue.Context
	providers      map[string]cue.Value
	registry       string
	releaseBuilder *release.Builder
	provider       *transform.ProviderLoader
	matcher        *transform.Matcher
	executor       *transform.Executor
}

// NewPipeline creates a new Pipeline implementation.
// cueCtx is the shared CUE evaluation context; must be the same context used to
// compile the provider values to avoid cross-runtime panics. If nil, a fresh
// context is created (suitable when no pre-compiled provider values are passed).
// providers maps provider names to their CUE values (from config.Providers).
// registry is the CUE registry URL (from config.Registry).
func NewPipeline(cueCtx *cue.Context, providers map[string]cue.Value, registry string) Pipeline {
	if cueCtx == nil {
		cueCtx = cuecontext.New()
	}
	return &pipeline{
		cueCtx:         cueCtx,
		providers:      providers,
		registry:       registry,
		releaseBuilder: release.NewBuilder(cueCtx, registry),
		provider:       transform.NewProviderLoader(providers),
		matcher:        transform.NewMatcher(),
		executor:       transform.NewExecutor(),
	}
}

// Render executes the pipeline and returns results.
//
// The render process follows these phases:
//  1. Load module: resolve path + AST inspection → *core.Module (module.Load + mod.Validate)
//  2. Build #ModuleRelease via AST overlay (loads module with overlay + values)
//     2a. ValidateValues: user values against #config schema
//     2b. Validate: all components are concrete
//  3. Load provider and transformers (ProviderLoader)
//  4. Match components to transformers (Matcher)
//  5. Execute transformers (Executor)
//  6. Build and return RenderResult
//
// Fatal errors (module not found, provider missing, incomplete values) return error.
// Render errors (unmatched components, transform failures) are in RenderResult.Errors.
func (p *pipeline) Render(ctx context.Context, opts RenderOptions) (*RenderResult, error) {
	// Validate options
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	// Phase 1: Load module — resolves path, inspects metadata via AST
	mod, err := module.Load(p.cueCtx, opts.ModulePath, opts.Registry)
	if err != nil {
		return nil, err
	}
	if err := mod.Validate(); err != nil {
		return nil, err
	}

	// When no --values flags are provided, values.cue must exist on disk.
	// When --values flags ARE provided, values.cue on disk is ignored (stubbed
	// out during Build) so the external values take full precedence.
	if len(opts.Values) == 0 {
		valuesPath := filepath.Join(mod.ModulePath, "values.cue")
		if _, err := os.Stat(valuesPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("values.cue not found in %s — provide values via values.cue or --values flag", mod.ModulePath)
		}
	}

	// Log which values files are being used
	if len(opts.Values) > 0 {
		valuesFileNames := make([]string, len(opts.Values))
		for i, vf := range opts.Values {
			valuesFileNames[i] = filepath.Base(vf)
		}
		output.Debug("using values files", "files", strings.Join(valuesFileNames, ", "))
	} else {
		output.Debug("using default values.cue")
	}

	// Phase 2: Build #ModuleRelease (loads module with AST overlay, unifies values)
	releaseName := opts.Name
	if releaseName == "" {
		releaseName = mod.Metadata.Name
	}
	namespace := p.resolveNamespace(opts.Namespace, mod.Metadata.DefaultNamespace)
	if namespace == "" {
		return nil, &NamespaceRequiredError{ModuleName: mod.Metadata.Name}
	}

	rel, err := p.releaseBuilder.Build(mod.ModulePath, release.Options{
		Name:      releaseName,
		Namespace: namespace,
		PkgName:   mod.PkgName(),
	}, opts.Values)
	if err != nil {
		return nil, err // Fatal: release building failed (likely incomplete values)
	}

	// Phase 2a: Validate user values against #config schema
	if err := rel.ValidateValues(); err != nil {
		return nil, err
	}

	// Phase 2b: Validate all components are concrete (ready for matching)
	if err := rel.Validate(); err != nil {
		return nil, err
	}

	output.Debug("release built",
		"name", rel.Metadata.Name,
		"namespace", rel.Metadata.Namespace,
		"components", len(rel.Components),
	)

	// Phase 3: Load provider
	providerName := opts.Provider
	// Default to the only configured provider if not specified
	if providerName == "" && len(p.providers) == 1 {
		for name := range p.providers { //nolint:revive // single iteration for auto-select
			providerName = name
			break
		}
	}
	provider, err := p.provider.Load(ctx, providerName)
	if err != nil {
		return nil, err // Fatal: provider loading failed
	}

	// Phase 4: Match components to transformers
	components := p.componentsToSlice(rel.Components)
	matchResult := p.matcher.Match(components, provider.Transformers)
	matchPlan := matchResult.ToMatchPlan()

	// Collect errors for unmatched components
	var errs []error
	for _, comp := range matchResult.Unmatched {
		errs = append(errs, &UnmatchedComponentError{
			ComponentName: comp.Name,
			Available:     provider.Requirements(),
		})
	}

	// Phase 5: Execute transformers (only for matched components)
	var resources []*core.Resource
	if len(matchResult.ByTransformer) > 0 {
		// Build transformer map for executor
		transformerMap := make(map[string]*transform.LoadedTransformer)
		for _, tf := range provider.Transformers {
			transformerMap[tf.FQN] = tf
		}
		execResult := p.executor.ExecuteWithTransformers(ctx, matchResult, rel, transformerMap)
		resources = execResult.Resources
		errs = append(errs, execResult.Errors...)
	}

	// Phase 6: Build result
	// Sort resources with deterministic 5-key total ordering:
	// weight → group → kind → namespace → name
	// This matches the digest sort in internal/inventory, making opm mod build
	// output deterministic for equal-weight resources.
	sort.SliceStable(resources, func(i, j int) bool {
		ri, rj := resources[i], resources[j]
		wi := core.GetWeight(ri.GVK())
		wj := core.GetWeight(rj.GVK())
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

	// Collect warnings (e.g., unhandled traits)
	warnings := collectWarnings(matchResult)

	return &RenderResult{
		Resources: resources,
		Release:   *rel.Metadata,
		Module:    *rel.Module.Metadata,
		MatchPlan: matchPlan,
		Errors:    errs,
		Warnings:  warnings,
	}, nil
}

// resolveNamespace resolves the target namespace using precedence:
// 1. --namespace flag (highest)
// 2. module.metadata.defaultNamespace
func (p *pipeline) resolveNamespace(flagValue, defaultNamespace string) string {
	if flagValue != "" {
		return flagValue
	}
	return defaultNamespace
}

// componentsToSlice converts a map of core.Component to a slice of build/component.Component
// for use by the matcher and executor. This conversion is a temporary bridge until the
// core-component consolidation change unifies these two types.
func (p *pipeline) componentsToSlice(m map[string]*core.Component) []*component.Component {
	result := make([]*component.Component, 0, len(m))
	for _, comp := range m {
		name := ""
		labels := map[string]string{}
		annotations := map[string]string{}
		if comp.Metadata != nil {
			name = comp.Metadata.Name
			if comp.Metadata.Labels != nil {
				labels = comp.Metadata.Labels
			}
			if comp.Metadata.Annotations != nil {
				annotations = comp.Metadata.Annotations
			}
		}
		result = append(result, &component.Component{
			Name:        name,
			Labels:      labels,
			Annotations: annotations,
			Resources:   comp.Resources,
			Traits:      comp.Traits,
			Value:       comp.Value,
		})
	}
	return result
}

// collectWarnings gathers non-fatal warnings from the match result.
//
// A trait is considered "unhandled" only if NO matched transformer handles it.
// This means if ServiceTransformer requires Expose trait and DeploymentTransformer
// doesn't, the Expose trait is still considered handled (by ServiceTransformer).
func collectWarnings(result *transform.MatchResult) []string {
	var warnings []string

	// Step 1: Count how many transformers matched each component
	componentMatchCount := make(map[string]int)
	for i := range result.Details {
		detail := &result.Details[i]
		if detail.Matched {
			componentMatchCount[detail.ComponentName]++
		}
	}

	// Step 2: Count how many matched transformers consider each trait unhandled
	// Key: component name, Value: map of trait -> count of transformers that don't handle it
	traitUnhandledCount := make(map[string]map[string]int)
	for i := range result.Details {
		detail := &result.Details[i]
		if detail.Matched {
			if traitUnhandledCount[detail.ComponentName] == nil {
				traitUnhandledCount[detail.ComponentName] = make(map[string]int)
			}
			for _, trait := range detail.UnhandledTraits {
				traitUnhandledCount[detail.ComponentName][trait]++
			}
		}
	}

	// Step 3: A trait is truly unhandled only if ALL matched transformers
	// consider it unhandled (i.e., no transformer handles it)
	for componentName, traitCounts := range traitUnhandledCount {
		matchCount := componentMatchCount[componentName]
		for trait, unhandledCount := range traitCounts {
			if unhandledCount == matchCount {
				warnings = append(warnings,
					"component "+componentName+": unhandled trait "+trait)
			}
		}
	}

	return warnings
}
