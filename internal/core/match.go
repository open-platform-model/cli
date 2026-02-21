package core

import (
	"context"

	"cuelang.org/go/cue"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
func (m *TransformerMatchPlan) Execute(ctx context.Context, rel *ModuleRelease) ([]*Resource, []error) {
	resources := make([]*Resource, 0)
	var errs []error

	for _, match := range m.Matches {
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
		tfFQN := ""
		if match.Transformer != nil && match.Transformer.Metadata != nil {
			tfFQN = match.Transformer.Metadata.FQN
		}

		res, err := m.executeMatch(match, rel, compName, tfFQN)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		resources = append(resources, res...)
	}

	return resources, errs
}

// executeMatch runs a single (transformer, component) pair and returns the resources.
func (m *TransformerMatchPlan) executeMatch(
	match *TransformerMatch,
	rel *ModuleRelease,
	compName, tfFQN string,
) ([]*Resource, error) {
	cueCtx := m.cueCtx

	// Resolve the #transform CUE value from the transformer.
	// core.Transformer stores the Transform field directly (extracted during LoadProvider).
	transformValue := match.Transformer.Transform
	if !transformValue.Exists() {
		return nil, &TransformError{
			ComponentName:  compName,
			TransformerFQN: tfFQN,
			Cause:          errMissingTransform,
		}
	}

	// Inject #component into the transformer.
	unified := transformValue.FillPath(cue.ParsePath("#component"), match.Component.Value)
	if unified.Err() != nil {
		return nil, &TransformError{
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
		return nil, &TransformError{
			ComponentName:  compName,
			TransformerFQN: tfFQN,
			Cause:          unified.Err(),
		}
	}

	// Extract output.
	outputValue := unified.LookupPath(cue.ParsePath("output"))
	if !outputValue.Exists() {
		return []*Resource{}, nil
	}

	if outputValue.Err() != nil {
		return nil, &TransformError{
			ComponentName:  compName,
			TransformerFQN: tfFQN,
			Cause:          outputValue.Err(),
		}
	}

	// Decode output — three forms: list, single resource (has apiVersion), or map of resources.
	//nolint:gocritic // ifElseChain: conditions are not comparable constants, switch is not applicable
	if outputValue.Kind() == cue.ListKind {
		return decodeResourceList(outputValue, compName, tfFQN)
	} else if isSingleResource(outputValue) {
		obj, err := decodeResource(outputValue)
		if err != nil {
			return nil, &TransformError{ComponentName: compName, TransformerFQN: tfFQN, Cause: err}
		}
		return []*Resource{{Object: obj, Component: compName, Transformer: tfFQN}}, nil
	}
	return decodeResourceMap(outputValue, compName, tfFQN)
}

// isSingleResource checks whether a CUE struct value represents a single Kubernetes resource.
func isSingleResource(value cue.Value) bool {
	return value.LookupPath(cue.ParsePath("apiVersion")).Exists()
}

// decodeResourceList decodes a CUE list of Kubernetes resource objects.
func decodeResourceList(value cue.Value, compName, tfFQN string) ([]*Resource, error) {
	var resources []*Resource
	iter, err := value.List()
	if err != nil {
		return nil, &TransformError{ComponentName: compName, TransformerFQN: tfFQN, Cause: err}
	}
	for iter.Next() {
		obj, err := decodeResource(iter.Value())
		if err != nil {
			return nil, &TransformError{ComponentName: compName, TransformerFQN: tfFQN, Cause: err}
		}
		resources = append(resources, &Resource{Object: obj, Component: compName, Transformer: tfFQN})
	}
	return resources, nil
}

// decodeResourceMap decodes a CUE struct of named Kubernetes resource objects.
func decodeResourceMap(value cue.Value, compName, tfFQN string) ([]*Resource, error) {
	var resources []*Resource
	iter, err := value.Fields()
	if err != nil {
		return nil, &TransformError{ComponentName: compName, TransformerFQN: tfFQN, Cause: err}
	}
	for iter.Next() {
		obj, err := decodeResource(iter.Value())
		if err != nil {
			return nil, &TransformError{ComponentName: compName, TransformerFQN: tfFQN, Cause: err}
		}
		resources = append(resources, &Resource{Object: obj, Component: compName, Transformer: tfFQN})
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

var errMissingTransform = &transformMissingError{}

type transformMissingError struct{}

func (e *transformMissingError) Error() string {
	return "transformer missing #transform function"
}
