package engine

import (
	"context"
	"fmt"

	"cuelang.org/go/cue"
	"github.com/charmbracelet/log"

	"github.com/opmodel/cli/pkg/core"
	"github.com/opmodel/cli/pkg/modulerelease"
)

// moduleReleaseContextData is the Go-side mirror of #TransformerContext.#moduleReleaseMetadata.
// Field names use json tags that match the CUE definition fields.
type moduleReleaseContextData struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	FQN         string            `json:"fqn"`
	Version     string            `json:"version"`
	UUID        string            `json:"uuid"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// componentContextData is the Go-side mirror of #TransformerContext.#componentMetadata.
type componentContextData struct {
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// executeTransforms runs the CUE #transform for each matched (component, transformer)
// pair in the plan and returns the decoded resources.
//
// schemaComponents is the original (non-finalized) components value — used for
// reading definition fields (metadata.labels, metadata.annotations) for #context.
// dataComponents is the finalized, constraint-free components value — used for
// FillPath injection into transformer #transform without schema conflicts.
//
// Execution is sequential: *cue.Context is not goroutine-safe.
// Resources are returned in the deterministic order produced by MatchedPairs().
// Per-pair errors are collected and returned alongside any successful resources.
func executeTransforms(
	ctx context.Context,
	cueCtx *cue.Context,
	plan *MatchPlan,
	providerVal cue.Value,
	schemaComponents cue.Value,
	dataComponents cue.Value,
	rel *modulerelease.ModuleRelease,
) ([]*core.Resource, []error) {
	resources := make([]*core.Resource, 0)
	var errs []error

	for _, pair := range plan.MatchedPairs() {
		select {
		case <-ctx.Done():
			return resources, append(errs, ctx.Err())
		default:
		}

		res, err := executePair(cueCtx, providerVal, schemaComponents, dataComponents, rel, pair)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		resources = append(resources, res...)
	}

	return resources, errs
}

// executePair runs the CUE #transform for a single (component, transformer) matched pair.
//
// The flow:
//  1. Look up the transformer's #transform from the provider raw value.
//  2. Look up the component from dataComponents (already finalized — no constraints).
//  3. FillPath #component with the data component value directly (no materialize needed).
//  4. FillPath #context.* fields (name, namespace, #moduleReleaseMetadata, #componentMetadata).
//     Metadata is read from schemaComponents which preserves definition fields.
//  5. Look up and decode the output field.
func executePair(
	cueCtx *cue.Context,
	providerVal cue.Value,
	schemaComponents cue.Value,
	dataComponents cue.Value,
	rel *modulerelease.ModuleRelease,
	pair MatchedPair,
) ([]*core.Resource, error) {
	compName := pair.ComponentName
	tfFQN := pair.TransformerFQN

	// Retrieve the transformer's #transform definition from the provider value.
	transformVal := providerVal.
		LookupPath(cue.ParsePath("#transformers")).
		LookupPath(cue.MakePath(cue.Str(tfFQN))).
		LookupPath(cue.ParsePath("#transform"))

	if !transformVal.Exists() {
		return nil, fmt.Errorf("component %q / transformer %q: #transform not found in provider", compName, tfFQN)
	}
	if err := transformVal.Err(); err != nil {
		return nil, fmt.Errorf("component %q / transformer %q: #transform error: %w", compName, tfFQN, err)
	}

	// Retrieve the finalized (constraint-free) component value from dataComponents.
	// No materialize() round-trip needed — components were finalized at load time.
	dataComp := dataComponents.LookupPath(cue.MakePath(cue.Str(compName)))
	if !dataComp.Exists() {
		return nil, fmt.Errorf("component %q not found in data components value", compName)
	}

	// Retrieve the schema component value for metadata extraction (#context injection).
	// schemaComponents preserves definition fields that are stripped by finalization.
	schemaComp := schemaComponents.LookupPath(cue.MakePath(cue.Str(compName)))

	// Inject #component using the finalized data value — safe for FillPath without
	// schema constraint conflicts.
	unified := transformVal.FillPath(cue.ParsePath("#component"), dataComp)
	if err := unified.Err(); err != nil {
		return nil, fmt.Errorf("component %q / transformer %q: filling #component: %w", compName, tfFQN, err)
	}

	// Build and inject #context. Reads metadata from schemaComp (has definitions).
	unified, err := injectContext(cueCtx, unified, rel, compName, schemaComp)
	if err != nil {
		return nil, fmt.Errorf("component %q / transformer %q: injecting #context: %w", compName, tfFQN, err)
	}

	// Extract the output field.
	outputVal := unified.LookupPath(cue.ParsePath("output"))
	if !outputVal.Exists() {
		return []*core.Resource{}, nil
	}
	if err := outputVal.Err(); err != nil {
		return nil, fmt.Errorf("component %q / transformer %q: evaluating output: %w", compName, tfFQN, err)
	}

	releaseName := rel.Metadata.Name

	// Decode the output into resources. Three supported forms:
	//   1. List of resources  — cue.ListKind
	//   2. Single resource    — cue.StructKind with both "apiVersion" and "kind" fields
	//   3. Map of resources   — cue.StructKind without top-level "apiVersion"/"kind"
	//
	// Fix for DEBT.md #6: The single-resource heuristic now requires both "apiVersion"
	// and "kind" to be present, making it more resilient than checking "apiVersion" alone.
	// Transformer authors MUST ensure output conforms to one of these three forms.
	switch outputVal.Kind() {
	case cue.ListKind:
		return collectResourceList(outputVal, releaseName, compName, tfFQN)
	case cue.StructKind:
		if isSingleResource(outputVal) {
			return []*core.Resource{{Value: outputVal, Release: releaseName, Component: compName, Transformer: tfFQN}}, nil
		}
		return collectResourceMap(outputVal, releaseName, compName, tfFQN)
	default:
		return nil, fmt.Errorf("component %q / transformer %q: unexpected output kind %s", compName, tfFQN, outputVal.Kind())
	}
}

// injectContext fills all #context fields into the unified transformer value.
//
// Uses typed structs (moduleReleaseContextData, componentContextData) encoded via
// cueCtx.Encode() rather than manually constructed map[string]any values. This
// keeps the injection type-safe and ensures the Go struct mirrors the CUE schema.
//
// compVal should be the schema component (from rel.MatchComponents()) so that
// metadata.labels and metadata.annotations are accessible even after finalization.
//
// Fix for DEBT.md #1: Metadata decode errors are logged at WARN level instead
// of being silently discarded. If a CUE value exists but cannot be decoded, the
// operator is informed with the component name and field path.
func injectContext(
	cueCtx *cue.Context,
	unified cue.Value,
	rel *modulerelease.ModuleRelease,
	compName string,
	compVal cue.Value,
) (cue.Value, error) {
	// #moduleReleaseMetadata — encode the typed struct directly.
	// Combines fields from both ReleaseMetadata and ModuleMetadata to mirror
	// the #TransformerContext.#moduleReleaseMetadata CUE schema.
	mrmData := moduleReleaseContextData{
		Name:        rel.Metadata.Name,
		Namespace:   rel.Metadata.Namespace,
		FQN:         rel.Module.Metadata.FQN,
		Version:     rel.Module.Metadata.Version,
		UUID:        rel.Metadata.UUID,
		Labels:      rel.Metadata.Labels,
		Annotations: rel.Metadata.Annotations,
	}
	unified = unified.FillPath(
		cue.MakePath(cue.Def("context"), cue.Def("moduleReleaseMetadata")),
		cueCtx.Encode(mrmData),
	)

	// #componentMetadata — decode labels/annotations from CUE value, then encode
	// back as a typed struct. Stays entirely in CUE-land: Decode() for reading,
	// Encode() for writing back.
	//
	// Fix for DEBT.md #1: Decode errors are logged at WARN level so malformed
	// metadata fields are surfaced to the operator rather than silently producing
	// empty labels/annotations in generated manifests.
	compMeta := componentContextData{Name: compName}
	if labelsVal := compVal.LookupPath(cue.ParsePath("metadata.labels")); labelsVal.Exists() {
		if err := labelsVal.Decode(&compMeta.Labels); err != nil {
			log.Warn("component metadata.labels could not be decoded; labels will be empty in transformer context",
				"component", compName,
				"err", err,
			)
		}
	}
	if annotationsVal := compVal.LookupPath(cue.ParsePath("metadata.annotations")); annotationsVal.Exists() {
		if err := annotationsVal.Decode(&compMeta.Annotations); err != nil {
			log.Warn("component metadata.annotations could not be decoded; annotations will be empty in transformer context",
				"component", compName,
				"err", err,
			)
		}
	}
	unified = unified.FillPath(
		cue.MakePath(cue.Def("context"), cue.Def("componentMetadata")),
		cueCtx.Encode(compMeta),
	)

	if err := unified.Err(); err != nil {
		return cue.Value{}, err
	}
	return unified, nil
}

// isSingleResource reports whether a CUE struct value is a single Kubernetes resource.
//
// A resource is considered "single" when it has both an "apiVersion" field and a
// "kind" field at the top level. Requiring both fields is more resilient than
// checking "apiVersion" alone (fix for DEBT.md #6).
//
// Transformer output must conform to one of three forms:
//   - list of resources    (cue.ListKind)
//   - single resource      (cue.StructKind with apiVersion + kind)
//   - map of named resources (cue.StructKind without apiVersion/kind at root)
func isSingleResource(v cue.Value) bool {
	return v.LookupPath(cue.ParsePath("apiVersion")).Exists() &&
		v.LookupPath(cue.ParsePath("kind")).Exists()
}

// collectResourceList wraps each item in a CUE list as a Resource,
// keeping the CUE value intact without any intermediate decoding.
func collectResourceList(v cue.Value, releaseName, compName, tfFQN string) ([]*core.Resource, error) {
	var resources []*core.Resource
	iter, err := v.List()
	if err != nil {
		return nil, fmt.Errorf("component %q / transformer %q: iterating output list: %w", compName, tfFQN, err)
	}
	for iter.Next() {
		resources = append(resources, &core.Resource{Value: iter.Value(), Release: releaseName, Component: compName, Transformer: tfFQN})
	}
	return resources, nil
}

// collectResourceMap wraps each field value in a CUE struct as a Resource,
// keeping the CUE value intact without any intermediate decoding.
func collectResourceMap(v cue.Value, releaseName, compName, tfFQN string) ([]*core.Resource, error) {
	var resources []*core.Resource
	iter, err := v.Fields()
	if err != nil {
		return nil, fmt.Errorf("component %q / transformer %q: iterating output map: %w", compName, tfFQN, err)
	}
	for iter.Next() {
		resources = append(resources, &core.Resource{Value: iter.Value(), Release: releaseName, Component: compName, Transformer: tfFQN})
	}
	return resources, nil
}
