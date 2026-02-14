package build

import (
	"context"

	"cuelang.org/go/cue"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/opmodel/cli/internal/output"
)

// Executor runs transformer jobs sequentially (CUE's *cue.Context is not safe for concurrent use).
type Executor struct{}

// NewExecutor creates a new Executor.
func NewExecutor() *Executor {
	return &Executor{}
}

// Job is a unit of work: one transformer applied to one component.
type Job struct {
	Transformer *LoadedTransformer
	Component   *LoadedComponent
	Release     *BuiltRelease
}

// JobResult is the result of executing a job.
type JobResult struct {
	Component   string
	Transformer string
	Resources   []*unstructured.Unstructured
	Error       error
}

// ExecuteResult is the combined result of all jobs.
type ExecuteResult struct {
	Resources []*Resource
	Errors    []error
}

// ExecuteWithTransformers runs transformations sequentially.
//
// For each (transformer, component) pair from the match result, it:
//  1. Looks up the transformer's #transform definition
//  2. Injects #component and #context via FillPath
//  3. Extracts and decodes the output as Kubernetes resources
func (e *Executor) ExecuteWithTransformers(
	ctx context.Context,
	match *MatchResult,
	release *BuiltRelease,
	transformers map[string]*LoadedTransformer,
) *ExecuteResult {
	result := &ExecuteResult{Resources: make([]*Resource, 0), Errors: make([]error, 0)}

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
				Release:     release,
			})
		}
	}

	if len(jobs) == 0 {
		return result
	}

	output.Debug("executing jobs", "count", len(jobs))

	// Execute jobs
	for _, job := range jobs {
		// Check for context cancellation between jobs
		select {
		case <-ctx.Done():
			result.Errors = append(result.Errors, ctx.Err())
			return result
		default:
		}

		jobResult := e.executeJob(job)
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

	output.Debug("execution complete", "resources", len(result.Resources), "errors", len(result.Errors))
	return result
}

// executeJob executes a single transformer job.
//
// It injects the component value and context metadata into the transformer's
// #transform definition via CUE FillPath, then extracts and decodes the output.
func (e *Executor) executeJob(job Job) JobResult {
	result := JobResult{
		Component:   job.Component.Name,
		Transformer: job.Transformer.FQN,
		Resources:   make([]*unstructured.Unstructured, 0),
	}

	cueCtx := job.Transformer.Value.Context()

	transformValue := job.Transformer.Value.LookupPath(cue.ParsePath("#transform"))
	if !transformValue.Exists() {
		result.Error = &TransformError{
			ComponentName:  job.Component.Name,
			TransformerFQN: job.Transformer.FQN,
			Cause:          errMissingTransform,
		}
		return result
	}

	// Inject #component into the transformer
	unified := transformValue.FillPath(cue.ParsePath("#component"), job.Component.Value)
	if unified.Err() != nil {
		result.Error = &TransformError{
			ComponentName:  job.Component.Name,
			TransformerFQN: job.Transformer.FQN,
			Cause:          unified.Err(),
		}
		return result
	}

	// Build and inject #context
	tfCtx := NewTransformerContext(job.Release, job.Component)

	unified = unified.FillPath(cue.ParsePath("#context.name"), cueCtx.Encode(tfCtx.Name))
	unified = unified.FillPath(cue.ParsePath("#context.namespace"), cueCtx.Encode(tfCtx.Namespace))

	moduleReleaseMetaMap := map[string]any{
		"name":      tfCtx.ModuleReleaseMetadata.Name,
		"namespace": tfCtx.ModuleReleaseMetadata.Namespace,
		"fqn":       tfCtx.ModuleReleaseMetadata.FQN,
		"version":   tfCtx.ModuleReleaseMetadata.Version,
		"identity":  tfCtx.ModuleReleaseMetadata.Identity,
	}
	if len(tfCtx.ModuleReleaseMetadata.Labels) > 0 {
		moduleReleaseMetaMap["labels"] = tfCtx.ModuleReleaseMetadata.Labels
	}
	unified = unified.FillPath(cue.MakePath(cue.Def("context"), cue.Def("moduleReleaseMetadata")), cueCtx.Encode(moduleReleaseMetaMap))

	compMetaMap := map[string]any{
		"name": tfCtx.ComponentMetadata.Name,
	}
	if len(tfCtx.ComponentMetadata.Labels) > 0 {
		compMetaMap["labels"] = tfCtx.ComponentMetadata.Labels
	}
	if len(tfCtx.ComponentMetadata.Annotations) > 0 {
		compMetaMap["annotations"] = tfCtx.ComponentMetadata.Annotations
	}
	unified = unified.FillPath(cue.MakePath(cue.Def("context"), cue.Def("componentMetadata")), cueCtx.Encode(compMetaMap))

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
		// No output is valid — transformer doesn't produce resources for this component
		return result
	}

	if outputValue.Err() != nil {
		result.Error = &TransformError{
			ComponentName:  job.Component.Name,
			TransformerFQN: job.Transformer.FQN,
			Cause:          outputValue.Err(),
		}
		return result
	}

	// Decode output — handles three cases:
	// 1. List: iterate elements, decode each as a resource
	// 2. Struct with apiVersion: single resource (e.g., Deployment)
	// 3. Struct without apiVersion: map of resources keyed by name (e.g., PVC per volume)
	if outputValue.Kind() == cue.ListKind {
		iter, err := outputValue.List()
		if err != nil {
			result.Error = &TransformError{ComponentName: job.Component.Name, TransformerFQN: job.Transformer.FQN, Cause: err}
			return result
		}
		for iter.Next() {
			obj, err := e.decodeResource(iter.Value())
			if err != nil {
				result.Error = &TransformError{ComponentName: job.Component.Name, TransformerFQN: job.Transformer.FQN, Cause: err}
				return result
			}
			result.Resources = append(result.Resources, obj)
		}
	} else if e.isSingleResource(outputValue) {
		obj, err := e.decodeResource(outputValue)
		if err != nil {
			result.Error = &TransformError{ComponentName: job.Component.Name, TransformerFQN: job.Transformer.FQN, Cause: err}
			return result
		}
		result.Resources = append(result.Resources, obj)
	} else {
		// Map of resources: iterate struct fields and decode each value
		iter, err := outputValue.Fields()
		if err != nil {
			result.Error = &TransformError{ComponentName: job.Component.Name, TransformerFQN: job.Transformer.FQN, Cause: err}
			return result
		}
		for iter.Next() {
			obj, err := e.decodeResource(iter.Value())
			if err != nil {
				result.Error = &TransformError{ComponentName: job.Component.Name, TransformerFQN: job.Transformer.FQN, Cause: err}
				return result
			}
			result.Resources = append(result.Resources, obj)
		}
	}

	return result
}

// isSingleResource checks whether a CUE struct value represents a single Kubernetes
// resource (has apiVersion at top level) vs a map of multiple resources keyed by name.
func (e *Executor) isSingleResource(value cue.Value) bool {
	apiVersion := value.LookupPath(cue.ParsePath("apiVersion"))
	return apiVersion.Exists()
}

func (e *Executor) decodeResource(value cue.Value) (*unstructured.Unstructured, error) {
	var obj map[string]any
	if err := value.Decode(&obj); err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: obj}, nil
}

var errMissingTransform = &transformMissingError{}

type transformMissingError struct{}

func (e *transformMissingError) Error() string {
	return "transformer missing #transform function"
}
