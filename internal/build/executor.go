package build

import (
	"context"
	"sync"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/opmodel/cli/internal/output"
)

// Executor runs transformers in parallel.
type Executor struct {
	workers int
}

// NewExecutor creates a new Executor with the specified worker count.
func NewExecutor(workers int) *Executor {
	if workers < 1 {
		workers = 1
	}
	return &Executor{workers: workers}
}

// Job is a unit of work for a worker.
type Job struct {
	Transformer *LoadedTransformer
	Component   *LoadedComponent
	Module      *LoadedModule
}

// JobResult is the result of executing a job.
type JobResult struct {
	Component   string
	Transformer string
	Resources   []*unstructured.Unstructured // May produce multiple resources
	Error       error
}

// ExecuteResult is the combined result of all jobs.
type ExecuteResult struct {
	Resources []*Resource
	Errors    []error
}

// Execute runs all transformations in parallel.
//
// Per FR-B-021 and FR-B-022:
//   - Execute transformers in parallel using worker pool
//   - Each worker has isolated cue.Context
func (e *Executor) Execute(ctx context.Context, match *MatchResult, module *LoadedModule) *ExecuteResult {
	result := &ExecuteResult{
		Resources: make([]*Resource, 0),
		Errors:    make([]error, 0),
	}

	// Build job list
	var jobs []Job
	for tfFQN, components := range match.ByTransformer {
		// Find transformer by FQN
		var transformer *LoadedTransformer
		for _, detail := range match.Details {
			if detail.TransformerFQN == tfFQN && detail.Matched {
				// Find the transformer from the component's match
				for _, comp := range components {
					if detail.ComponentName == comp.Name {
						// We need to look up the transformer from somewhere
						// For now, we'll store transformer reference in the job
						break
					}
				}
			}
		}

		// Since we don't have direct access to the transformer, we need to pass it through
		// In a real implementation, we'd either store transformers in MatchResult or
		// have a separate lookup. For now, we'll skip if transformer not found.
		if transformer == nil {
			continue
		}

		for _, comp := range components {
			jobs = append(jobs, Job{
				Transformer: transformer,
				Component:   comp,
				Module:      module,
			})
		}
	}

	if len(jobs) == 0 {
		return result
	}

	// Create channels
	jobChan := make(chan Job, len(jobs))
	resultChan := make(chan JobResult, len(jobs))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < e.workers && i < len(jobs); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e.runWorker(ctx, jobChan, resultChan)
		}()
	}

	// Send jobs
	for _, job := range jobs {
		jobChan <- job
	}
	close(jobChan)

	// Wait for workers to finish
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	for jobResult := range resultChan {
		if jobResult.Error != nil {
			result.Errors = append(result.Errors, jobResult.Error)
			continue
		}

		for _, obj := range jobResult.Resources {
			result.Resources = append(result.Resources, &Resource{
				Object:      obj,
				Component:   jobResult.Component,
				Transformer: jobResult.Transformer,
			})
		}
	}

	return result
}

// ExecuteWithTransformers runs transformations with explicit transformer map.
// This is the primary execution method that accepts transformers directly.
func (e *Executor) ExecuteWithTransformers(
	ctx context.Context,
	match *MatchResult,
	module *LoadedModule,
	transformers map[string]*LoadedTransformer,
) *ExecuteResult {
	result := &ExecuteResult{
		Resources: make([]*Resource, 0),
		Errors:    make([]error, 0),
	}

	// Build job list
	var jobs []Job
	for tfFQN, components := range match.ByTransformer {
		transformer, ok := transformers[tfFQN]
		if !ok {
			output.Debug("transformer not found for FQN", "fqn", tfFQN)
			continue
		}

		for _, comp := range components {
			jobs = append(jobs, Job{
				Transformer: transformer,
				Component:   comp,
				Module:      module,
			})
		}
	}

	if len(jobs) == 0 {
		return result
	}

	output.Debug("executing jobs",
		"count", len(jobs),
		"workers", e.workers,
	)

	// Create channels
	jobChan := make(chan Job, len(jobs))
	resultChan := make(chan JobResult, len(jobs))

	// Start workers
	var wg sync.WaitGroup
	workerCount := e.workers
	if workerCount > len(jobs) {
		workerCount = len(jobs)
	}

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e.runWorker(ctx, jobChan, resultChan)
		}()
	}

	// Send jobs
	for _, job := range jobs {
		jobChan <- job
	}
	close(jobChan)

	// Wait for workers to finish
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	for jobResult := range resultChan {
		if jobResult.Error != nil {
			result.Errors = append(result.Errors, jobResult.Error)
			continue
		}

		for _, obj := range jobResult.Resources {
			result.Resources = append(result.Resources, &Resource{
				Object:      obj,
				Component:   jobResult.Component,
				Transformer: jobResult.Transformer,
			})
		}
	}

	output.Debug("execution complete",
		"resources", len(result.Resources),
		"errors", len(result.Errors),
	)

	return result
}

// runWorker processes jobs from the job channel.
func (e *Executor) runWorker(ctx context.Context, jobs <-chan Job, results chan<- JobResult) {
	// Each worker gets its own CUE context (per FR-B-022)
	cueCtx := cuecontext.New()

	for job := range jobs {
		select {
		case <-ctx.Done():
			results <- JobResult{
				Component:   job.Component.Name,
				Transformer: job.Transformer.FQN,
				Error:       ctx.Err(),
			}
			return
		default:
			result := e.executeJob(cueCtx, job)
			results <- result
		}
	}
}

// executeJob executes a single transformation.
func (e *Executor) executeJob(cueCtx *cue.Context, job Job) JobResult {
	result := JobResult{
		Component:   job.Component.Name,
		Transformer: job.Transformer.FQN,
		Resources:   make([]*unstructured.Unstructured, 0),
	}

	// Build transformer context
	tfCtx := BuildTransformerContext(job.Module, job.Component)

	// Get the #transform function from the transformer
	transformValue := job.Transformer.Value.LookupPath(cue.ParsePath("#transform"))
	if !transformValue.Exists() {
		result.Error = &TransformError{
			ComponentName:  job.Component.Name,
			TransformerFQN: job.Transformer.FQN,
			Cause:          errMissingTransform,
		}
		return result
	}

	// Build input for transformation
	// The transformer expects: #component, #context
	input := cueCtx.Encode(map[string]any{
		"#component": job.Component.Value,
		"#context":   tfCtx,
	})

	if input.Err() != nil {
		result.Error = &TransformError{
			ComponentName:  job.Component.Name,
			TransformerFQN: job.Transformer.FQN,
			Cause:          input.Err(),
		}
		return result
	}

	// Unify transformer with input
	unified := transformValue.Unify(input)
	if unified.Err() != nil {
		result.Error = &TransformError{
			ComponentName:  job.Component.Name,
			TransformerFQN: job.Transformer.FQN,
			Cause:          unified.Err(),
		}
		return result
	}

	// Extract output
	outputValue := unified.LookupPath(cue.ParsePath("output"))
	if !outputValue.Exists() {
		// No output is valid (transformer produces nothing)
		return result
	}

	// Output can be a single resource or a list
	if outputValue.Kind() == cue.ListKind {
		// List of resources
		iter, err := outputValue.List()
		if err != nil {
			result.Error = &TransformError{
				ComponentName:  job.Component.Name,
				TransformerFQN: job.Transformer.FQN,
				Cause:          err,
			}
			return result
		}

		for iter.Next() {
			obj, err := e.decodeResource(iter.Value())
			if err != nil {
				result.Error = &TransformError{
					ComponentName:  job.Component.Name,
					TransformerFQN: job.Transformer.FQN,
					Cause:          err,
				}
				return result
			}
			result.Resources = append(result.Resources, obj)
		}
	} else {
		// Single resource
		obj, err := e.decodeResource(outputValue)
		if err != nil {
			result.Error = &TransformError{
				ComponentName:  job.Component.Name,
				TransformerFQN: job.Transformer.FQN,
				Cause:          err,
			}
			return result
		}
		result.Resources = append(result.Resources, obj)
	}

	return result
}

// decodeResource decodes a CUE value to an unstructured Kubernetes resource.
func (e *Executor) decodeResource(value cue.Value) (*unstructured.Unstructured, error) {
	var obj map[string]any
	if err := value.Decode(&obj); err != nil {
		return nil, err
	}

	return &unstructured.Unstructured{Object: obj}, nil
}

// errMissingTransform is returned when transformer lacks #transform.
var errMissingTransform = &transformMissingError{}

type transformMissingError struct{}

func (e *transformMissingError) Error() string {
	return "transformer missing #transform function"
}
