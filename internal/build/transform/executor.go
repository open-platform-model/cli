package transform

import (
	"context"
	"fmt"

	"cuelang.org/go/cue"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/opmodel/cli/internal/build/module"
	"github.com/opmodel/cli/internal/build/release"
	"github.com/opmodel/cli/internal/output"
)

// TransformError indicates transformer execution failed.
//
//nolint:revive // stutter is intentional: this type is re-exported as build.TransformError
type TransformError struct {
	ComponentName  string
	TransformerFQN string
	Cause          error
}

func (e *TransformError) Error() string {
	return fmt.Sprintf("component %q, transformer %q: %v",
		e.ComponentName, e.TransformerFQN, e.Cause)
}

func (e *TransformError) Unwrap() error {
	return e.Cause
}

// Component returns the component name where the error occurred.
// Implements the build.RenderError interface.
func (e *TransformError) Component() string {
	return e.ComponentName
}

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

	// Decode output
	//nolint:gocritic // ifElseChain: conditions are not comparable constants, switch is not applicable
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

// NewTransformerContextForComponent is a helper that creates a context from a release and component.
// It exists to avoid import cycles — callers use it instead of NewTransformerContext directly.
func NewTransformerContextForComponent(rel *release.BuiltRelease, comp *module.LoadedComponent) *TransformerContext {
	return NewTransformerContext(rel, comp)
}

var errMissingTransform = &transformMissingError{}

type transformMissingError struct{}

func (e *transformMissingError) Error() string {
	return "transformer missing #transform function"
}
