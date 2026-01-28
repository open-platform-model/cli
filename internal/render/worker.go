package render

import (
	"fmt"
	"sync"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

// executeTransforms performs Phase 4: Parallel transformer execution.
func executeTransforms(jobs []Job, verbose bool) ([]Result, PhaseRecord) {
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
		if verbose && result.Error == nil {
			fmt.Printf("  [WORKER] %s/%s: %v\n", result.TransformerID, result.ComponentName, result.Duration)
		}
	}

	record := PhaseRecord{
		Name:     "Parallel Execution",
		Duration: time.Since(phase4Start),
		Details:  fmt.Sprintf("%d workers (max: %v)", len(jobs), maxWorkerDuration),
	}

	return results, record
}

// runWorker executes in an isolated goroutine with its own cue.Context.
// This is the core of the parallel execution pattern.
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
