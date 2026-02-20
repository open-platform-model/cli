package transform

import (
	"context"

	"cuelang.org/go/cue"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/opmodel/cli/internal/build/release"
	"github.com/opmodel/cli/internal/core"
	"github.com/opmodel/cli/internal/output"
)

// Executor runs transformer jobs sequentially (CUE's *cue.Context is not safe for concurrent use).
type Executor struct{}

// NewExecutor creates a new Executor.
func NewExecutor() *Executor {
	return &Executor{}
}

// ExecuteWithTransformers runs transformations sequentially.
func (e *Executor) ExecuteWithTransformers(
	ctx context.Context,
	match *MatchResult,
	rel *release.BuiltRelease,
	transformers map[string]*LoadedTransformer,
) *ExecuteResult {
	result := &ExecuteResult{Resources: make([]*core.Resource, 0), Errors: make([]error, 0)}

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
				Release:     rel,
			})
		}
	}

	if len(jobs) == 0 {
		return result
	}

	output.Debug("executing jobs", "count", len(jobs))

	for _, job := range jobs {
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
			result.Resources = append(result.Resources, &core.Resource{
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
func (e *Executor) executeJob(job Job) JobResult {
	result := JobResult{
		Component:   job.Component.Name,
		Transformer: job.Transformer.FQN,
		Resources:   make([]*unstructured.Unstructured, 0),
	}

	cueCtx := job.Transformer.Value.Context()

	transformValue := job.Transformer.Value.LookupPath(cue.ParsePath("#transform"))
	if !transformValue.Exists() {
		result.Error = &core.TransformError{
			ComponentName:  job.Component.Name,
			TransformerFQN: job.Transformer.FQN,
			Cause:          errMissingTransform,
		}
		return result
	}

	// Inject #component into the transformer
	unified := transformValue.FillPath(cue.ParsePath("#component"), job.Component.Value)
	if unified.Err() != nil {
		result.Error = &core.TransformError{
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

	ctxMap := tfCtx.ToMap()
	unified = unified.FillPath(cue.MakePath(cue.Def("context"), cue.Def("moduleReleaseMetadata")), cueCtx.Encode(ctxMap["#moduleReleaseMetadata"]))
	unified = unified.FillPath(cue.MakePath(cue.Def("context"), cue.Def("componentMetadata")), cueCtx.Encode(ctxMap["#componentMetadata"]))

	if unified.Err() != nil {
		result.Error = &core.TransformError{
			ComponentName:  job.Component.Name,
			TransformerFQN: job.Transformer.FQN,
			Cause:          unified.Err(),
		}
		return result
	}

	// Extract output
	outputValue := unified.LookupPath(cue.ParsePath("output"))
	if !outputValue.Exists() {
		return result
	}

	if outputValue.Err() != nil {
		result.Error = &core.TransformError{
			ComponentName:  job.Component.Name,
			TransformerFQN: job.Transformer.FQN,
			Cause:          outputValue.Err(),
		}
		return result
	}

	// Decode output
	//nolint:gocritic // ifElseChain: conditions are not comparable constants, switch is not applicable
	if outputValue.Kind() == cue.ListKind {
		iter, err := outputValue.List()
		if err != nil {
			result.Error = &core.TransformError{ComponentName: job.Component.Name, TransformerFQN: job.Transformer.FQN, Cause: err}
			return result
		}
		for iter.Next() {
			obj, err := e.decodeResource(iter.Value())
			if err != nil {
				result.Error = &core.TransformError{ComponentName: job.Component.Name, TransformerFQN: job.Transformer.FQN, Cause: err}
				return result
			}
			result.Resources = append(result.Resources, obj)
		}
	} else if e.isSingleResource(outputValue) {
		obj, err := e.decodeResource(outputValue)
		if err != nil {
			result.Error = &core.TransformError{ComponentName: job.Component.Name, TransformerFQN: job.Transformer.FQN, Cause: err}
			return result
		}
		result.Resources = append(result.Resources, obj)
	} else {
		iter, err := outputValue.Fields()
		if err != nil {
			result.Error = &core.TransformError{ComponentName: job.Component.Name, TransformerFQN: job.Transformer.FQN, Cause: err}
			return result
		}
		for iter.Next() {
			obj, err := e.decodeResource(iter.Value())
			if err != nil {
				result.Error = &core.TransformError{ComponentName: job.Component.Name, TransformerFQN: job.Transformer.FQN, Cause: err}
				return result
			}
			result.Resources = append(result.Resources, obj)
		}
	}

	return result
}

// isSingleResource checks whether a CUE struct value represents a single Kubernetes resource.
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
