package transformer

import (
	"context"
	"encoding/json"
	"fmt"

	"cuelang.org/go/cue"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/opmodel/cli/internal/core"
	"github.com/opmodel/cli/internal/core/modulerelease"
	opmerrors "github.com/opmodel/cli/internal/errors"
)

// Execute sequentially runs the transform function of each matched
// (transformer, component) pair and returns the generated resources.
//
// Execution is sequential because *cue.Context is not safe for concurrent use.
// Resources are returned in match-plan order (deterministic, sorted by component
// name then transformer name because Provider.Match sorts before iterating).
//
// A non-nil empty []*Resource slice is returned when m.Matches is empty.
// Any per-match errors are collected in the returned []error; execution
// continues past individual match errors so all matches are attempted.
// Context cancellation is checked between matches.
func (p *TransformerMatchPlan) Execute(ctx context.Context, rel *modulerelease.ModuleRelease) ([]*core.Resource, []error) {
	resources := make([]*core.Resource, 0)
	var errs []error

	for _, match := range p.Matches {
		if !match.Matched {
			continue
		}

		select {
		case <-ctx.Done():
			errs = append(errs, ctx.Err())
			return resources, errs
		default:
		}

		compName := ""
		if match.Component != nil && match.Component.Metadata != nil {
			compName = match.Component.Metadata.Name
		}
		tfFQN := match.Transformer.GetFQN()

		res, err := p.executeMatch(match, rel, compName, tfFQN)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		resources = append(resources, res...)
	}

	return resources, errs
}

// executeMatch runs a single (transformer, component) pair and returns the resources.
func (p *TransformerMatchPlan) executeMatch(
	match *TransformerMatch,
	rel *modulerelease.ModuleRelease,
	compName, tfFQN string,
) ([]*core.Resource, error) {
	cueCtx := p.cueCtx

	// Resolve the #transform CUE value from the transformer.
	// Transformer stores the Transform field directly (extracted during LoadProvider).
	transformValue := match.Transformer.Transform
	if !transformValue.Exists() {
		return nil, &opmerrors.TransformError{
			ComponentName:  compName,
			TransformerFQN: tfFQN,
			Cause:          errMissingTransform,
		}
	}

	// Materialize the component value before injection.
	//
	// CUE values from module evaluation carry schema constraints (e.g., matchN
	// validators on #VolumeSchema). These constraints interact badly with
	// `if field != _|_` guards in transformer comprehensions when the value is
	// injected via FillPath — the guard sees the optional-field template from
	// the constraint rather than the concrete data. JSON round-tripping produces
	// an equivalent concrete value without schema constraints, which allows
	// transformer comprehensions to correctly evaluate conditional branches.
	//
	// See: CUE matchN + FillPath interaction causing hostPath volumes to be
	// silently dropped in DaemonSet/Deployment/StatefulSet transformers.
	materializedComp, err := materialize(cueCtx, match.Component.Value)
	if err != nil {
		return nil, &opmerrors.TransformError{
			ComponentName:  compName,
			TransformerFQN: tfFQN,
			Cause:          fmt.Errorf("materializing component: %w", err),
		}
	}

	// Inject #component into the transformer.
	unified := transformValue.FillPath(cue.ParsePath("#component"), materializedComp)
	if unified.Err() != nil {
		return nil, &opmerrors.TransformError{
			ComponentName:  compName,
			TransformerFQN: tfFQN,
			Cause:          unified.Err(),
		}
	}

	// Build and inject #context.
	tfCtx := NewTransformerContext(rel, match.Component)
	unified = unified.FillPath(cue.ParsePath("#context.name"), cueCtx.Encode(tfCtx.Name))
	unified = unified.FillPath(cue.ParsePath("#context.namespace"), cueCtx.Encode(tfCtx.Namespace))

	ctxMap := tfCtx.ToMap()
	unified = unified.FillPath(
		cue.MakePath(cue.Def("context"), cue.Def("moduleReleaseMetadata")),
		cueCtx.Encode(ctxMap["#moduleReleaseMetadata"]),
	)
	unified = unified.FillPath(
		cue.MakePath(cue.Def("context"), cue.Def("componentMetadata")),
		cueCtx.Encode(ctxMap["#componentMetadata"]),
	)

	if unified.Err() != nil {
		return nil, &opmerrors.TransformError{
			ComponentName:  compName,
			TransformerFQN: tfFQN,
			Cause:          unified.Err(),
		}
	}

	// Extract output.
	outputValue := unified.LookupPath(cue.ParsePath("output"))
	if !outputValue.Exists() {
		return []*core.Resource{}, nil
	}

	if outputValue.Err() != nil {
		return nil, &opmerrors.TransformError{
			ComponentName:  compName,
			TransformerFQN: tfFQN,
			Cause:          outputValue.Err(),
		}
	}

	// Decode output — three forms: list, single resource (has apiVersion), or map of resources.
	if outputValue.Kind() == cue.ListKind {
		return decodeResourceList(outputValue, compName, tfFQN)
	} else if isSingleResource(outputValue) {
		obj, err := decodeResource(outputValue)
		if err != nil {
			return nil, &opmerrors.TransformError{ComponentName: compName, TransformerFQN: tfFQN, Cause: err}
		}
		return []*core.Resource{{Object: obj, Component: compName, Transformer: tfFQN}}, nil
	}
	return decodeResourceMap(outputValue, compName, tfFQN)
}

// isSingleResource checks whether a CUE struct value represents a single Kubernetes resource.
func isSingleResource(value cue.Value) bool {
	return value.LookupPath(cue.ParsePath("apiVersion")).Exists()
}

// decodeResourceList decodes a CUE list of Kubernetes resource objects.
func decodeResourceList(value cue.Value, compName, tfFQN string) ([]*core.Resource, error) {
	var resources []*core.Resource
	iter, err := value.List()
	if err != nil {
		return nil, &opmerrors.TransformError{ComponentName: compName, TransformerFQN: tfFQN, Cause: err}
	}
	for iter.Next() {
		obj, err := decodeResource(iter.Value())
		if err != nil {
			return nil, &opmerrors.TransformError{ComponentName: compName, TransformerFQN: tfFQN, Cause: err}
		}
		resources = append(resources, &core.Resource{Object: obj, Component: compName, Transformer: tfFQN})
	}
	return resources, nil
}

// decodeResourceMap decodes a CUE struct of named Kubernetes resource objects.
func decodeResourceMap(value cue.Value, compName, tfFQN string) ([]*core.Resource, error) {
	var resources []*core.Resource
	iter, err := value.Fields()
	if err != nil {
		return nil, &opmerrors.TransformError{ComponentName: compName, TransformerFQN: tfFQN, Cause: err}
	}
	for iter.Next() {
		obj, err := decodeResource(iter.Value())
		if err != nil {
			return nil, &opmerrors.TransformError{ComponentName: compName, TransformerFQN: tfFQN, Cause: err}
		}
		resources = append(resources, &core.Resource{Object: obj, Component: compName, Transformer: tfFQN})
	}
	return resources, nil
}

// decodeResource decodes a single CUE value into an Unstructured Kubernetes object.
func decodeResource(value cue.Value) (*unstructured.Unstructured, error) {
	var obj map[string]any
	if err := value.Decode(&obj); err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: obj}, nil
}

// materialize converts a CUE value to a constraint-free equivalent by
// round-tripping through JSON. This strips schema validators (e.g., matchN)
// that can interfere with transformer comprehension guards while preserving
// the concrete data.
func materialize(ctx *cue.Context, v cue.Value) (cue.Value, error) {
	var data any
	if err := v.Decode(&data); err != nil {
		return cue.Value{}, fmt.Errorf("decoding CUE value: %w", err)
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return cue.Value{}, fmt.Errorf("marshalling to JSON: %w", err)
	}
	result := ctx.CompileBytes(raw)
	if result.Err() != nil {
		return cue.Value{}, fmt.Errorf("compiling materialized JSON: %w", result.Err())
	}
	return result, nil
}

var errMissingTransform = &transformMissingError{}

type transformMissingError struct{}

func (e *transformMissingError) Error() string {
	return "transformer missing #transform function"
}
