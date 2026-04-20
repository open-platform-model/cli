package k8sgen

import (
	"fmt"

	"cuelang.org/go/cue"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Crossplane Composition API constants. Note apiextensions.crossplane.io/v1
// (not v2) — Crossplane v2 kept Composition on v1 even as XRD moved to v2.
// Inlined rather than importing crossplane-runtime, which is not a direct
// dependency of this repo; the emitted manifest is an
// unstructured.Unstructured anyway.
const (
	compositionAPIVersion = "apiextensions.crossplane.io/v1"
	compositionKind       = "Composition"
	compositionMode       = "Pipeline"
)

// function-opm input contract. Matches the function-opm Go struct the
// function uses to deserialize its composition input.
const (
	defaultFunctionName    = "function-opm"
	defaultCompStepName    = "render-opm-module"
	defaultInputAPIVersion = "template.fn.crossplane.io/v1beta1"
	inputKind              = "Input"
)

// CompositionOptions configures Composition generation.
type CompositionOptions struct {
	// Group is the API group the emitted XR lives under (e.g.
	// "module.opmodel.dev"). Required — must match the paired XRD's group
	// so spec.compositeTypeRef binds to the right XR kind.
	Group string

	// FunctionName is the Crossplane function referenced by the pipeline
	// step. Defaults to "function-opm" when empty.
	FunctionName string

	// StepName is the pipeline step name. Defaults to "render-opm-module"
	// when empty.
	StepName string

	// InputAPIVersion is the apiVersion of the function input payload.
	// Defaults to "template.fn.crossplane.io/v1beta1" when empty — the
	// group function-opm currently uses.
	InputAPIVersion string
}

// BuildComposition produces a Crossplane Composition that binds a paired
// XRD to function-opm and tells the function which OPM module to render.
// The returned manifest targets apiextensions.crossplane.io/v1.
//
// Field sources:
//   - apiVersion/kind: fixed (apiextensions.crossplane.io/v1, Composition).
//   - metadata.name: module metadata.name verbatim (compositeTypeRef does
//     the structural binding, so the Composition name has no load-bearing
//     role).
//   - spec.compositeTypeRef.apiVersion: "<opts.Group>/<DeriveVersion(...)>".
//   - spec.compositeTypeRef.kind: DeriveNames(metadata.name).Kind.
//   - spec.mode: always "Pipeline".
//   - spec.pipeline[0].step: opts.StepName (default "render-opm-module").
//   - spec.pipeline[0].functionRef.name: opts.FunctionName (default
//     "function-opm").
//   - spec.pipeline[0].input.module.path: metadata.modulePath +
//     "/" + metadata.name (the full module identifier).
//   - spec.pipeline[0].input.module.version: metadata.version verbatim.
//
// modulePath is required on the module (not optional as it is for CRD/XRD
// provenance annotations); without it the function has no module to
// render.
func BuildComposition(modVal cue.Value, opts CompositionOptions) (*unstructured.Unstructured, error) {
	if !modVal.Exists() {
		return nil, fmt.Errorf("module value does not exist")
	}
	if opts.Group == "" {
		return nil, fmt.Errorf("composition group is required")
	}
	if opts.FunctionName == "" {
		opts.FunctionName = defaultFunctionName
	}
	if opts.StepName == "" {
		opts.StepName = defaultCompStepName
	}
	if opts.InputAPIVersion == "" {
		opts.InputAPIVersion = defaultInputAPIVersion
	}

	moduleName, err := lookupString(modVal, "metadata.name")
	if err != nil {
		return nil, fmt.Errorf("reading metadata.name: %w", err)
	}
	moduleVersion, err := lookupString(modVal, "metadata.version")
	if err != nil {
		return nil, fmt.Errorf("reading metadata.version: %w", err)
	}
	modulePath, err := lookupString(modVal, "metadata.modulePath")
	if err != nil {
		return nil, fmt.Errorf(
			"reading metadata.modulePath (required for Composition input): %w", err,
		)
	}

	names, err := DeriveNames(moduleName)
	if err != nil {
		return nil, err
	}
	version, err := DeriveVersion(moduleVersion)
	if err != nil {
		return nil, err
	}

	labels, annotations := buildProvenance(modVal, moduleName, moduleVersion)

	compMeta := map[string]any{
		"name": moduleName,
	}
	if len(labels) > 0 {
		compMeta["labels"] = toAnyMap(labels)
	}
	if len(annotations) > 0 {
		compMeta["annotations"] = toAnyMap(annotations)
	}

	comp := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": compositionAPIVersion,
			"kind":       compositionKind,
			"metadata":   compMeta,
			"spec": map[string]any{
				"compositeTypeRef": map[string]any{
					"apiVersion": opts.Group + "/" + version,
					"kind":       names.Kind,
				},
				"mode": compositionMode,
				"pipeline": []any{
					map[string]any{
						"step": opts.StepName,
						"functionRef": map[string]any{
							"name": opts.FunctionName,
						},
						"input": map[string]any{
							"apiVersion": opts.InputAPIVersion,
							"kind":       inputKind,
							"module": map[string]any{
								"path":    modulePath + "/" + moduleName,
								"version": moduleVersion,
							},
						},
					},
				},
			},
		},
	}
	return comp, nil
}
