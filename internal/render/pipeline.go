package render

import (
	"context"
	"fmt"
)

// Pipeline orchestrates the 5-phase hybrid render process.
type Pipeline struct {
	Options *Options
}

// NewPipeline creates a new render pipeline with the given options.
func NewPipeline(opts *Options) *Pipeline {
	return &Pipeline{Options: opts}
}

// Render executes the full render pipeline and returns the result.
// This implements the 5-phase hybrid Go+CUE architecture:
//
//	Phase 1: Module Loading
//	Phase 2: Provider Loading & Metadata Extraction
//	Phase 3: Component Matching (CUE-computed)
//	Phase 4: Parallel Transformer Execution
//	Phase 5: Aggregation & Output
func (p *Pipeline) Render(ctx context.Context) (*RenderResult, error) {
	var phases []PhaseRecord

	if p.Options.Verbose {
		fmt.Println("=== Hybrid Render Pipeline ===")
		fmt.Println("")
	}

	// PHASE 1: Module Loading
	if p.Options.Verbose {
		fmt.Println("[Phase 1] Loading CUE module...")
	}
	rootVal, mainCtx, phase1Record, err := loadModule(p.Options.Dir)
	if err != nil {
		return nil, fmt.Errorf("phase 1 (module loading) failed: %w", err)
	}
	phases = append(phases, phase1Record)

	// PHASE 2: Provider Loading & Metadata Extraction
	if p.Options.Verbose {
		fmt.Println("[Phase 2] Loading provider and transformers...")
	}
	meta, phase2Record, err := extractMetadata(rootVal)
	if err != nil {
		return nil, fmt.Errorf("phase 2 (provider loading) failed: %w", err)
	}
	phases = append(phases, phase2Record)

	if p.Options.Verbose {
		fmt.Printf("  Release: %s (namespace: %s, version: %s)\n",
			meta.ReleaseName, meta.ReleaseNamespace, meta.ModuleVersion)
	}

	// PHASE 3: Component Matching
	if p.Options.Verbose {
		fmt.Println("")
		fmt.Println("[Phase 3] Component matching...")
	}
	jobs, unmatchedComponents, phase3Record, err := computeMatches(mainCtx, meta, p.Options.Verbose)
	if err != nil {
		return nil, fmt.Errorf("phase 3 (component matching) failed: %w", err)
	}
	phases = append(phases, phase3Record)

	if p.Options.Verbose {
		fmt.Printf("  Total matches: %d\n", len(jobs))
	}

	// Per spec FR-019: Error on unmatched components (aggregated at end)
	var unmatchedErrors []error
	if len(unmatchedComponents) > 0 {
		for _, name := range unmatchedComponents {
			unmatchedErrors = append(unmatchedErrors,
				fmt.Errorf("no transformer matched component '%s'", name))
		}
		if p.Options.Verbose {
			fmt.Printf("  [WARN] %d unmatched component(s) will be reported as errors\n", len(unmatchedComponents))
		}
	}

	// PHASE 4: Parallel Execution
	if p.Options.Verbose {
		fmt.Println("")
		fmt.Printf("[Phase 4] Executing %d transformations in parallel...\n", len(jobs))
	}
	results, phase4Record := executeTransforms(jobs, p.Options.Verbose)
	phases = append(phases, phase4Record)

	// PHASE 5: Aggregation & Output
	if p.Options.Verbose {
		fmt.Println("")
		fmt.Println("[Phase 5] Aggregating results...")
	}
	phase5Record, errors, manifests, err := aggregateResults(results, unmatchedErrors)
	if err != nil {
		return nil, fmt.Errorf("phase 5 (aggregation) failed: %w", err)
	}
	phases = append(phases, phase5Record)

	// Always print timing summary in verbose mode
	if p.Options.Verbose {
		printTimingSummary(phases)
		fmt.Println("")
	}

	// Report aggregated errors at the end (fail-on-end per spec FR-024)
	if len(errors) > 0 {
		if p.Options.Verbose {
			fmt.Printf("FAILED: %d errors\n", len(errors))
			for _, err := range errors {
				fmt.Printf("  - %v\n", err)
			}
		}
		// Return the result even with errors, so caller can decide how to handle
		return &RenderResult{
			Manifests:        manifests,
			ModuleName:       meta.ReleaseName,
			ModuleVersion:    meta.ModuleVersion,
			ReleaseNamespace: meta.ReleaseNamespace,
			Phases:           phases,
			Errors:           errors,
		}, fmt.Errorf("render pipeline completed with %d error(s)", len(errors))
	}

	if p.Options.Verbose {
		fmt.Printf("=== SUCCESS: %d Kubernetes resources from %d matches ===\n", len(manifests), len(jobs))
	}

	return &RenderResult{
		Manifests:        manifests,
		ModuleName:       meta.ReleaseName,
		ModuleVersion:    meta.ModuleVersion,
		ReleaseNamespace: meta.ReleaseNamespace,
		Phases:           phases,
		Errors:           nil,
	}, nil
}
