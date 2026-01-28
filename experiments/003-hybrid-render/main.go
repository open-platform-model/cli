package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"go.yaml.in/yaml/v3"
)

// Command-line flags
var (
	outputFile = flag.String("o", "", "Output YAML to file (progress always shown on stdout)")
	verbose    = flag.Bool("v", false, "Verbose output (show per-match and per-worker details)")
)

// ============================================================================
// Types - Per 013-cli-render-spec/data-model.md
// ============================================================================

// Job carries all necessary info to the worker (AST-based transport for thread safety)
type Job struct {
	TransformerID string
	ComponentName string
	// We pass the UNIFIED result (transformer + component + context) as AST
	// This ensures all references are resolved before crossing goroutine boundary
	UnifiedAST ast.Expr
}

// TransformerContext carries the CLI-set fields for core.#TransformerContext.
type TransformerContext struct {
	Name      string `json:"name"`      // CLI-set release name
	Namespace string `json:"namespace"` // CLI-set target namespace
}

// Result carries either a successful output or an error (for fail-on-end aggregation)
type Result struct {
	TransformerID string
	ComponentName string
	Output        map[string]any // Decoded K8s resource
	Error         error
	Duration      time.Duration // Worker execution time
}

// PhaseStep represents a timed sub-step within a phase
type PhaseStep struct {
	Name     string
	Duration time.Duration
}

// PhaseRecord captures timing for an entire pipeline phase
type PhaseRecord struct {
	Name     string
	Duration time.Duration
	Steps    []PhaseStep
	Details  string // Human-readable summary (e.g., "8 jobs from 7 components")
}

// progress prints to stdout always (main pipeline progress)
func progress(format string, args ...any) {
	fmt.Fprintf(os.Stdout, format+"\n", args...)
}

// detail prints to stdout only in verbose mode (per-match, per-worker details)
func detail(format string, args ...any) {
	if *verbose {
		fmt.Fprintf(os.Stdout, format+"\n", args...)
	}
}

// printTimingSummary outputs an ASCII table with phase timing details
func printTimingSummary(phases []PhaseRecord) {
	fmt.Fprintln(os.Stdout, "")
	fmt.Fprintln(os.Stdout, "╭──────────────────────────────┬──────────┬───────────────────────────────────────────────╮")
	fmt.Fprintln(os.Stdout, "│ Phase                        │ Duration │ Details                                       │")
	fmt.Fprintln(os.Stdout, "├──────────────────────────────┼──────────┼───────────────────────────────────────────────┤")

	var totalDuration time.Duration
	for i, phase := range phases {
		totalDuration += phase.Duration

		// Format duration (keep it concise)
		durStr := formatDuration(phase.Duration)

		// Build details string (include sub-steps if any)
		details := phase.Details
		if len(phase.Steps) > 0 {
			stepStrs := make([]string, len(phase.Steps))
			for j, step := range phase.Steps {
				stepStrs[j] = fmt.Sprintf("%s: %s", step.Name, formatDuration(step.Duration))
			}
			if details != "" {
				details = fmt.Sprintf("%s | %s", details, stepStrs[0])
				if len(stepStrs) > 1 {
					details = fmt.Sprintf("%s, %s", details, stepStrs[1])
				}
			} else {
				details = stepStrs[0]
				if len(stepStrs) > 1 {
					details = fmt.Sprintf("%s, %s", details, stepStrs[1])
				}
			}
		}

		// Truncate details if too long
		if len(details) > 45 {
			details = details[:42] + "..."
		}

		fmt.Fprintf(os.Stdout, "│ %d. %-25s │ %8s │ %-45s │\n", i+1, phase.Name, durStr, details)
	}

	fmt.Fprintln(os.Stdout, "├──────────────────────────────┼──────────┼───────────────────────────────────────────────┤")
	fmt.Fprintf(os.Stdout, "│ %-28s │ %8s │ %-45s │\n", "Total", formatDuration(totalDuration), "Pipeline complete")
	fmt.Fprintln(os.Stdout, "╰──────────────────────────────┴──────────┴───────────────────────────────────────────────╯")
}

// formatDuration returns a concise duration string
func formatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

// ============================================================================
// Phase Functions
// ============================================================================

// ProviderMetadata holds extracted provider information
type ProviderMetadata struct {
	MatchingPlanVal   cue.Value
	ModuleMetadataVal cue.Value
	ReleaseName       string
	ReleaseNamespace  string
	ModuleVersion     string
}

// loadCUEModule performs Phase 1: CUE module loading and validation
func loadCUEModule(dir string) (cue.Value, *cue.Context, PhaseRecord, error) {
	phase1Start := time.Now()
	var phase1Steps []PhaseStep

	mainCtx := cuecontext.New()
	expDir, err := filepath.Abs(dir)
	if err != nil {
		return cue.Value{}, nil, PhaseRecord{}, fmt.Errorf("failed to resolve directory: %w", err)
	}

	cfg := &load.Config{
		ModuleRoot: expDir,
		Dir:        expDir,
	}

	loadStart := time.Now()
	instances := load.Instances([]string{"."}, cfg)
	if len(instances) != 1 {
		return cue.Value{}, nil, PhaseRecord{}, fmt.Errorf("expected 1 instance, got %d", len(instances))
	}
	if instances[0].Err != nil {
		return cue.Value{}, nil, PhaseRecord{}, fmt.Errorf("failed to load CUE instance: %w", instances[0].Err)
	}
	phase1Steps = append(phase1Steps, PhaseStep{Name: "load.Instances", Duration: time.Since(loadStart)})

	buildStart := time.Now()
	rootVal := mainCtx.BuildInstance(instances[0])
	// NOTE: We intentionally do not check rootVal.Err() here.
	// The CUE instance contains abstract definitions that are not concrete at the top level.
	phase1Steps = append(phase1Steps, PhaseStep{Name: "BuildInstance", Duration: time.Since(buildStart)})

	record := PhaseRecord{
		Name:     "Module Loading",
		Duration: time.Since(phase1Start),
		Steps:    phase1Steps,
		Details:  "CUE module loaded",
	}

	return rootVal, mainCtx, record, nil
}

// extractProviderMetadata performs Phase 2: Provider loading and metadata extraction
func extractProviderMetadata(rootVal cue.Value) (ProviderMetadata, PhaseRecord, error) {
	phase2Start := time.Now()
	var phase2Steps []PhaseStep

	// Load the matching plan (computed by CUE)
	lookupStart := time.Now()
	matchingPlanVal := rootVal.LookupPath(cue.ParsePath("matchingPlan"))
	if err := matchingPlanVal.Err(); err != nil {
		return ProviderMetadata{}, PhaseRecord{}, fmt.Errorf("failed to load matchingPlan: %w", err)
	}
	phase2Steps = append(phase2Steps, PhaseStep{Name: "LookupPath(matchingPlan)", Duration: time.Since(lookupStart)})

	// Extract module release metadata for TransformerContext
	metadataStart := time.Now()
	moduleReleaseVal := rootVal.LookupPath(cue.ParsePath("allBlueprintsModuleRelease"))
	moduleMetadataVal := moduleReleaseVal.LookupPath(cue.ParsePath("metadata"))
	releaseName, _ := moduleMetadataVal.LookupPath(cue.ParsePath("name")).String()
	releaseNamespace, _ := moduleMetadataVal.LookupPath(cue.ParsePath("namespace")).String()
	moduleVersion, _ := moduleMetadataVal.LookupPath(cue.ParsePath("version")).String()
	phase2Steps = append(phase2Steps, PhaseStep{Name: "Extract metadata", Duration: time.Since(metadataStart)})

	progress("  Release: %s (namespace: %s, version: %s)", releaseName, releaseNamespace, moduleVersion)

	record := PhaseRecord{
		Name:     "Provider Loading",
		Duration: time.Since(phase2Start),
		Steps:    phase2Steps,
		Details:  fmt.Sprintf("Release: %s", releaseName),
	}

	meta := ProviderMetadata{
		MatchingPlanVal:   matchingPlanVal,
		ModuleMetadataVal: moduleMetadataVal,
		ReleaseName:       releaseName,
		ReleaseNamespace:  releaseNamespace,
		ModuleVersion:     moduleVersion,
	}

	return meta, record, nil
}

// computeMatches performs Phase 3: CUE-computed component matching
func computeMatches(mainCtx *cue.Context, meta ProviderMetadata) ([]Job, []string, PhaseRecord, error) {
	phase3Start := time.Now()

	// The matching plan is already computed by CUE (via #MatchTransformers)
	matchedTransformersVal := meta.MatchingPlanVal
	if err := matchedTransformersVal.Err(); err != nil {
		return nil, nil, PhaseRecord{}, fmt.Errorf("failed to read matchingPlan: %w", err)
	}

	// Build the base TransformerContext with CLI-set fields only
	baseContext := TransformerContext{
		Name:      meta.ReleaseName,
		Namespace: meta.ReleaseNamespace,
	}

	// Iterate the CUE-computed matchedTransformers map
	var jobs []Job
	var unmatchedComponents []string

	matchIter, _ := matchedTransformersVal.Fields()
	matchCount := 0
	for matchIter.Next() {
		transformerID := matchIter.Selector().Unquoted()
		matchVal := matchIter.Value()

		transformerVal := matchVal.LookupPath(cue.ParsePath("transformer"))
		componentsVal := matchVal.LookupPath(cue.ParsePath("components"))

		// Get the #transform function
		transformFuncVal := transformerVal.LookupPath(cue.ParsePath("#transform"))

		// Iterate components matched to this transformer
		compIter, _ := componentsVal.List()
		for compIter.Next() {
			compVal := compIter.Value()
			compMetadataVal := compVal.LookupPath(cue.ParsePath("metadata"))
			compName, _ := compMetadataVal.LookupPath(cue.ParsePath("name")).String()

			matchCount++
			detail("  [MATCH %d] '%s' -> %s", matchCount, compName, transformerID)

			// Build context: encode CLI-set fields, then fill hidden CUE definitions
			contextVal := mainCtx.Encode(baseContext).
				FillPath(cue.ParsePath("#moduleMetadata"), meta.ModuleMetadataVal).
				FillPath(cue.ParsePath("#componentMetadata"), compMetadataVal)

			// IMPORTANT: Unify transformer with inputs IN THE MAIN CONTEXT
			transformInput := mainCtx.CompileString("{}").
				FillPath(cue.ParsePath("#component"), compVal).
				FillPath(cue.ParsePath("context"), contextVal)

			unified := transformFuncVal.Unify(transformInput)
			if err := unified.Err(); err != nil {
				fmt.Fprintf(os.Stderr, "  [ERROR] Unification failed for '%s' with %s: %v\n", compName, transformerID, err)
				unmatchedComponents = append(unmatchedComponents, compName)
				continue
			}

			// Export the unified result as AST for thread-safe transport
			unifiedAST := unified.Syntax(cue.Final(), cue.Concrete(true)).(ast.Expr)

			jobs = append(jobs, Job{
				TransformerID: transformerID,
				ComponentName: compName,
				UnifiedAST:    unifiedAST,
			})
		}
	}

	progress("  Total matches: %d", matchCount)

	record := PhaseRecord{
		Name:     "Component Matching",
		Duration: time.Since(phase3Start),
		Details:  fmt.Sprintf("%d jobs created", len(jobs)),
	}

	return jobs, unmatchedComponents, record, nil
}

// executeTransforms performs Phase 4: Parallel transformer execution
func executeTransforms(jobs []Job) ([]Result, PhaseRecord) {
	phase4Start := time.Now()

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

	// Collect results
	var results []Result
	var maxWorkerDuration time.Duration
	for result := range resultChan {
		results = append(results, result)
		if result.Duration > maxWorkerDuration {
			maxWorkerDuration = result.Duration
		}
		// Verbose: show per-worker timing
		if result.Error == nil {
			detail("  [WORKER] %s/%s: %v", result.TransformerID, result.ComponentName, result.Duration)
		}
	}

	record := PhaseRecord{
		Name:     "Parallel Execution",
		Duration: time.Since(phase4Start),
		Details:  fmt.Sprintf("%d workers (max: %v)", len(jobs), maxWorkerDuration),
	}

	return results, record
}

// aggregateResults performs Phase 5: Result aggregation and YAML output
func aggregateResults(results []Result, unmatchedErrors []error) (PhaseRecord, []error, int, error) {
	phase5Start := time.Now()
	var phase5Steps []PhaseStep

	var errors []error
	errors = append(errors, unmatchedErrors...)

	collectionStart := time.Now()
	for _, result := range results {
		if result.Error != nil {
			errors = append(errors, result.Error)
		}
	}
	collectionDuration := time.Since(collectionStart)
	phase5Steps = append(phase5Steps, PhaseStep{Name: "Collect results", Duration: collectionDuration})

	// Sort results for deterministic output (FR-023)
	sort.Slice(results, func(i, j int) bool {
		if results[i].TransformerID != results[j].TransformerID {
			return results[i].TransformerID < results[j].TransformerID
		}
		return results[i].ComponentName < results[j].ComponentName
	})

	// Output YAML only if -o is set
	outputStart := time.Now()
	successCount := 0
	var outputDuration time.Duration

	if *outputFile != "" {
		f, err := os.Create(*outputFile)
		if err != nil {
			return PhaseRecord{}, nil, 0, fmt.Errorf("failed to create output file: %w", err)
		}
		defer f.Close()

		progress("Writing YAML to: %s", *outputFile)

		for _, r := range results {
			if r.Error != nil {
				continue
			}

			// Output as YAML (per spec FR-017, YAML is the default format)
			yamlBytes, _ := yaml.Marshal(r.Output)
			if successCount > 0 {
				fmt.Fprintln(f, "---") // YAML document separator
			}
			fmt.Fprintf(f, "# Source: %s/%s\n%s", r.TransformerID, r.ComponentName, string(yamlBytes))
			successCount++
		}
		outputDuration = time.Since(outputStart)
	} else {
		// No YAML output, just count successes
		for _, r := range results {
			if r.Error == nil {
				successCount++
			}
		}
	}

	phase5Steps = append(phase5Steps, PhaseStep{Name: "YAML marshal", Duration: outputDuration})

	record := PhaseRecord{
		Name:     "Aggregation & Output",
		Duration: time.Since(phase5Start),
		Steps:    phase5Steps,
		Details:  fmt.Sprintf("%d resources, %d errors", successCount, len(errors)),
	}

	return record, errors, successCount, nil
}

// ============================================================================
// Main Pipeline Orchestrator
// ============================================================================

func main() {
	flag.Parse()

	progress("=== Hybrid Render Pipeline ===")
	progress("")

	var phases []PhaseRecord

	// PHASE 1: Module Loading
	progress("[Phase 1] Loading CUE module...")
	rootVal, mainCtx, phase1Record, err := loadCUEModule(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAILED: %v\n", err)
		os.Exit(1)
	}
	phases = append(phases, phase1Record)

	// PHASE 2: Provider Loading
	progress("[Phase 2] Loading provider and transformers...")
	meta, phase2Record, err := extractProviderMetadata(rootVal)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAILED: %v\n", err)
		os.Exit(1)
	}
	phases = append(phases, phase2Record)

	// PHASE 3: Component Matching
	progress("")
	progress("[Phase 3] Component matching...")
	jobs, unmatchedComponents, phase3Record, err := computeMatches(mainCtx, meta)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAILED: %v\n", err)
		os.Exit(1)
	}
	phases = append(phases, phase3Record)

	// Per spec FR-019: Error on unmatched components (aggregated at end)
	var unmatchedErrors []error
	if len(unmatchedComponents) > 0 {
		for _, name := range unmatchedComponents {
			unmatchedErrors = append(unmatchedErrors,
				fmt.Errorf("no transformer matched component '%s'", name))
		}
		fmt.Fprintf(os.Stderr, "  [WARN] %d unmatched component(s) will be reported as errors\n", len(unmatchedComponents))
	}

	// PHASE 4: Parallel Execution
	progress("")
	progress("[Phase 4] Executing %d transformations in parallel...", len(jobs))
	results, phase4Record := executeTransforms(jobs)
	phases = append(phases, phase4Record)

	// PHASE 5: Aggregation & Output
	progress("")
	progress("[Phase 5] Aggregating results...")
	phase5Record, errors, successCount, err := aggregateResults(results, unmatchedErrors)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAILED: %v\n", err)
		os.Exit(1)
	}
	phases = append(phases, phase5Record)

	// Always print timing summary
	printTimingSummary(phases)

	// Report aggregated errors at the end (fail-on-end per spec FR-024)
	progress("")
	if len(errors) > 0 {
		fmt.Fprintf(os.Stderr, "FAILED: %d errors\n", len(errors))
		for _, err := range errors {
			fmt.Fprintf(os.Stderr, "  - %v\n", err)
		}
		os.Exit(1)
	} else {
		matchCount := len(jobs) // matchCount is the same as number of jobs created
		progress("=== SUCCESS: %d Kubernetes resources from %d matches ===", successCount, matchCount)
	}
}

// runWorker executes in an isolated goroutine with its own cue.Context
// This is the core of the parallel execution pattern per spec research.md
func runWorker(job Job) Result {
	start := time.Now()

	// Each worker gets its own isolated context (thread-safe)
	workerCtx := cuecontext.New()

	// Re-hydrate the unified AST in worker's context
	// The AST is already fully resolved (no #component/context references)
	unified := workerCtx.BuildExpr(job.UnifiedAST)

	// Check for build errors
	if err := unified.Err(); err != nil {
		return Result{
			TransformerID: job.TransformerID,
			ComponentName: job.ComponentName,
			Error:         fmt.Errorf("AST re-hydration failed for '%s/%s': %w", job.TransformerID, job.ComponentName, err),
			Duration:      time.Since(start),
		}
	}

	// Extract the output
	outputVal := unified.LookupPath(cue.ParsePath("output"))
	if err := outputVal.Err(); err != nil {
		return Result{
			TransformerID: job.TransformerID,
			ComponentName: job.ComponentName,
			Error:         fmt.Errorf("output extraction failed for '%s/%s': %w", job.TransformerID, job.ComponentName, err),
			Duration:      time.Since(start),
		}
	}

	// Decode to Go map for YAML serialization
	var output map[string]any
	if err := outputVal.Decode(&output); err != nil {
		return Result{
			TransformerID: job.TransformerID,
			ComponentName: job.ComponentName,
			Error:         fmt.Errorf("output decode failed for '%s/%s': %w", job.TransformerID, job.ComponentName, err),
			Duration:      time.Since(start),
		}
	}

	return Result{
		TransformerID: job.TransformerID,
		ComponentName: job.ComponentName,
		Output:        output,
		Duration:      time.Since(start),
	}
}
