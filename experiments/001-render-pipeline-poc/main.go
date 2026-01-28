package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"go.yaml.in/yaml/v3"
)

// Command-line flags
var (
	outputFile = flag.String("o", "", "Output file path (default: stdout)")
	verbose    = flag.Bool("v", false, "Verbose output (show pipeline phases)")
)

// ============================================================================
// Types - Per 013-cli-render-spec/data-model.md
// ============================================================================

// Job carries all necessary info to the worker (AST-based transport for thread safety)
// Per research.md: AST nodes are pure Go structs, independent of cue.Context, so thread-safe
type Job struct {
	ComponentName string
	// We pass the UNIFIED result (transformer + component + context) as AST
	// This ensures all references are resolved before crossing goroutine boundary
	UnifiedAST ast.Expr
}

// TransformerContext carries the CLI-set fields for core.#TransformerContext.
// The hidden definitions (#moduleMetadata, #componentMetadata) and derived labels
// (moduleLabels, componentLabels, controllerLabels, labels) are populated by CUE via FillPath.
type TransformerContext struct {
	Name      string `json:"name"`      // CLI-set release name
	Namespace string `json:"namespace"` // CLI-set target namespace
}

// Result carries either a successful output or an error (for fail-on-end aggregation)
type Result struct {
	ComponentName string
	Output        map[string]any // Decoded K8s resource
	Error         error
}

// log prints to stderr only in verbose mode
func log(format string, args ...any) {
	if *verbose {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
}

func main() {
	flag.Parse()

	log("=== Render Pipeline PoC (013-cli-render-spec) ===")
	log("")

	// --- PHASE 1: Module Loading & Validation (spec Section 3, Phase 1) ---
	log("[Phase 1] Loading CUE module...")
	mainCtx := cuecontext.New()
	expDir, _ := filepath.Abs(".")

	cfg := &load.Config{
		ModuleRoot: expDir,
		Dir:        expDir,
	}

	instances := load.Instances([]string{"poc.cue"}, cfg)
	if len(instances) != 1 || instances[0].Err != nil {
		if instances[0].Err != nil {
			panic(fmt.Sprintf("failed to load CUE instance: %v", instances[0].Err))
		}
		panic(fmt.Sprintf("expected 1 instance, got %d", len(instances)))
	}

	rootVal := mainCtx.BuildInstance(instances[0])
	// NOTE: We intentionally do not check rootVal.Err() here.
	// The CUE instance contains abstract definitions (#Module.#spec: _)
	// that are not concrete at the top level. This is expected — we only
	// access concrete paths (release, transformers) via LookupPath.

	// Extract release and its metadata for TransformerContext
	releaseVal := rootVal.LookupPath(cue.ParsePath("release"))
	releaseMetadataVal := releaseVal.LookupPath(cue.ParsePath("metadata"))
	releaseName, _ := releaseMetadataVal.LookupPath(cue.ParsePath("name")).String()
	releaseNamespace, _ := releaseMetadataVal.LookupPath(cue.ParsePath("namespace")).String()
	moduleVersion, _ := releaseMetadataVal.LookupPath(cue.ParsePath("version")).String()

	log("  Release: %s (namespace: %s, version: %s)", releaseName, releaseNamespace, moduleVersion)

	// --- PHASE 2 & 3: Component Matching (spec Section 3, Phases 2-3) ---
	log("")
	log("[Phase 2-3] Matching components to transformers...")

	componentsVal := releaseVal.LookupPath(cue.ParsePath("components"))
	iter, _ := componentsVal.Fields()

	var jobs []Job
	var unmatchedComponents []string

	// Build the base TransformerContext with CLI-set fields only.
	// The hidden definitions (#moduleMetadata, #componentMetadata) and all
	// derived labels are populated by CUE unification, not Go.
	baseContext := TransformerContext{
		Name:      releaseName,
		Namespace: releaseNamespace,
	}

	for iter.Next() {
		compName := iter.Selector().Unquoted()
		compVal := iter.Value()

		// Match transformer based on labels (simplified matching for PoC)
		// Full implementation would check requiredResources, requiredTraits, etc.
		var transformerPath string
		workloadType, _ := compVal.LookupPath(cue.ParsePath(`metadata.labels."core.opm.dev/workload-type"`)).String()

		switch workloadType {
		case "stateless":
			transformerPath = "#DeploymentTransformer.#transform"
		case "stateful":
			transformerPath = "#StatefulSetTransformer.#transform"
		default:
			unmatchedComponents = append(unmatchedComponents, compName)
			continue
		}

		transformerVal := rootVal.LookupPath(cue.ParsePath(transformerPath))
		log("  [MATCH] '%s' (workload-type=%s) -> %s", compName, workloadType, transformerPath)

		// Build context: encode CLI-set fields, then fill hidden CUE definitions.
		// CUE derives moduleLabels, componentLabels, controllerLabels, and labels
		// automatically from #moduleMetadata and #componentMetadata.
		compMetadataVal := compVal.LookupPath(cue.ParsePath("metadata"))
		contextVal := mainCtx.Encode(baseContext).
			FillPath(cue.ParsePath("#moduleMetadata"), releaseMetadataVal).
			FillPath(cue.ParsePath("#componentMetadata"), compMetadataVal)

		// IMPORTANT: Unify transformer with inputs IN THE MAIN CONTEXT
		// This resolves all #component and context references before crossing
		// the goroutine boundary. The result is fully concrete.
		transformInput := mainCtx.CompileString("{}").
			FillPath(cue.ParsePath("#component"), compVal).
			FillPath(cue.ParsePath("context"), contextVal)

		unified := transformerVal.Unify(transformInput)
		if err := unified.Err(); err != nil {
			log("  [ERROR] Unification failed for '%s': %v", compName, err)
			continue
		}

		// Export the unified result as AST for thread-safe transport
		// cue.Final() resolves references, cue.Concrete(true) ensures concrete values
		unifiedAST := unified.Syntax(cue.Final(), cue.Concrete(true)).(ast.Expr)

		jobs = append(jobs, Job{
			ComponentName: compName,
			UnifiedAST:    unifiedAST,
		})
	}

	// Per spec FR-019: Error on unmatched components (aggregated at end)
	// We record these as errors but continue processing other components
	var unmatchedErrors []error
	if len(unmatchedComponents) > 0 {
		for _, name := range unmatchedComponents {
			unmatchedErrors = append(unmatchedErrors,
				fmt.Errorf("no transformer matched component '%s'", name))
		}
		log("  [WARN] %d unmatched component(s) will be reported as errors", len(unmatchedComponents))
	}

	// --- PHASE 4: Parallel Transformer Execution (spec Section 3, Phase 4) ---
	log("")
	log("[Phase 4] Executing %d transformations in parallel...", len(jobs))

	resultChan := make(chan Result, len(jobs))
	var wg sync.WaitGroup

	for _, job := range jobs {
		wg.Add(1)
		go func(j Job) {
			defer wg.Done()
			result := runWorker(j)
			resultChan <- result
		}(job)
	}

	// Close channel when all workers complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// --- PHASE 5: Aggregation & Output (spec Section 3, Phase 5) ---
	log("")
	log("[Phase 5] Aggregating results (fail-on-end)...")

	var results []Result
	var errors []error

	// Add unmatched component errors to the error list
	errors = append(errors, unmatchedErrors...)

	for result := range resultChan {
		results = append(results, result)
		if result.Error != nil {
			errors = append(errors, result.Error)
		}
	}

	// Determine output destination
	var out io.Writer = os.Stdout
	if *outputFile != "" {
		f, err := os.Create(*outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		out = f
		log("Writing output to: %s", *outputFile)
	}

	// Output all results as YAML (even if some failed - fail-on-end per spec)
	log("")
	log("=== Generated Kubernetes Manifests (YAML) ===")

	successCount := 0
	for i, r := range results {
		if r.Error != nil {
			fmt.Fprintf(os.Stderr, "# ERROR: %s: %v\n", r.ComponentName, r.Error)
			continue
		}

		// Output as YAML (per spec FR-017, YAML is the default format)
		yamlBytes, _ := yaml.Marshal(r.Output)
		if successCount > 0 {
			fmt.Fprintln(out, "---") // YAML document separator
		}
		fmt.Fprintf(out, "# Source: %s\n%s", r.ComponentName, string(yamlBytes))
		successCount++
		_ = i // suppress unused warning
	}

	// Report aggregated errors at the end (fail-on-end per spec FR-024)
	log("")
	if len(errors) > 0 {
		fmt.Fprintf(os.Stderr, "FAILED: %d errors\n", len(errors))
		for _, err := range errors {
			fmt.Fprintf(os.Stderr, "  - %v\n", err)
		}
		os.Exit(1)
	} else {
		log("=== SUCCESS: All transformations completed ===")
	}
}

// runWorker executes in an isolated goroutine with its own cue.Context
// This is the core of the parallel execution pattern per spec research.md
func runWorker(job Job) Result {
	// Each worker gets its own isolated context (thread-safe)
	workerCtx := cuecontext.New()

	// Re-hydrate the unified AST in worker's context
	// The AST is already fully resolved (no #component/#context references)
	unified := workerCtx.BuildExpr(job.UnifiedAST)

	// Check for build errors
	if err := unified.Err(); err != nil {
		return Result{
			ComponentName: job.ComponentName,
			Error:         fmt.Errorf("AST re-hydration failed for '%s': %w", job.ComponentName, err),
		}
	}

	// Extract the output
	outputVal := unified.LookupPath(cue.ParsePath("output"))
	if err := outputVal.Err(); err != nil {
		return Result{
			ComponentName: job.ComponentName,
			Error:         fmt.Errorf("output extraction failed for '%s': %w", job.ComponentName, err),
		}
	}

	// Decode to Go map for JSON serialization
	var output map[string]any
	if err := outputVal.Decode(&output); err != nil {
		return Result{
			ComponentName: job.ComponentName,
			Error:         fmt.Errorf("output decode failed for '%s': %w", job.ComponentName, err),
		}
	}

	return Result{
		ComponentName: job.ComponentName,
		Output:        output,
	}
}
