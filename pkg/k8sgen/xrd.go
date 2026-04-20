package k8sgen

import (
	"fmt"

	"cuelang.org/go/cue"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// apiextensions.crossplane.io/v2 API constants. v2-only; v1 removed claims
// and defaulted XRs to Namespaced, which matches how OPM modules are meant
// to be consumed. Inlined rather than importing crossplane-runtime, which
// is not a direct dependency of this repo; the emitted manifest is an
// unstructured.Unstructured anyway.
const (
	xrdAPIVersion = "apiextensions.crossplane.io/v2"
	xrdKind       = "CompositeResourceDefinition"
)

// XRDScope is the spec.scope value on a Crossplane v2 XRD.
type XRDScope string

// Crossplane v2 XRD scopes.
const (
	XRDScopeNamespaced    XRDScope = "Namespaced"
	XRDScopeCluster       XRDScope = "Cluster"
	XRDScopeLegacyCluster XRDScope = "LegacyCluster"
)

// XRDOptions configures XRD generation.
type XRDOptions struct {
	// Group is the API group the XRD will be registered under (e.g.
	// "module.opmodel.dev"). Required.
	Group string

	// Scope controls whether instances of the XR are namespaced or
	// cluster-scoped. Defaults to Namespaced when empty — the same default
	// Crossplane v2 uses.
	Scope XRDScope
}

// BuildXRD produces a Crossplane v2 CompositeResourceDefinition for the
// module's #config definition. The returned manifest targets
// apiextensions.crossplane.io/v2.
//
// Field sources:
//   - apiVersion/kind: fixed (apiextensions.crossplane.io/v2,
//     CompositeResourceDefinition).
//   - group: opts.Group (required).
//   - scope: opts.Scope (default Namespaced).
//   - names: derived from metadata.name via DeriveNames.
//   - version: derived from metadata.version via DeriveVersion.
//   - schema: #config wrapped under properties.spec; see ExtractConfigSchema.
//     status is not emitted in POC scope — compositions own status in v2.
//   - metadata.name: "<plural>.<group>" (canonical convention).
//
// The reserved-field check rejects a #config whose top-level properties
// include "crossplane"; Crossplane v2 reserves spec.crossplane.* and
// status.crossplane.* for its own use.
func BuildXRD(modVal cue.Value, opts XRDOptions) (*unstructured.Unstructured, error) {
	if !modVal.Exists() {
		return nil, fmt.Errorf("module value does not exist")
	}
	if opts.Group == "" {
		return nil, fmt.Errorf("XRD group is required")
	}
	scope := opts.Scope
	if scope == "" {
		scope = XRDScopeNamespaced
	}
	switch scope {
	case XRDScopeNamespaced, XRDScopeCluster, XRDScopeLegacyCluster:
	default:
		return nil, fmt.Errorf(
			"invalid XRD scope %q; want Namespaced, Cluster, or LegacyCluster",
			scope,
		)
	}

	moduleName, err := lookupString(modVal, "metadata.name")
	if err != nil {
		return nil, fmt.Errorf("reading metadata.name: %w", err)
	}
	moduleVersion, err := lookupString(modVal, "metadata.version")
	if err != nil {
		return nil, fmt.Errorf("reading metadata.version: %w", err)
	}

	names, err := DeriveNames(moduleName)
	if err != nil {
		return nil, err
	}
	version, err := DeriveVersion(moduleVersion)
	if err != nil {
		return nil, err
	}
	configSchema, err := ExtractConfigSchema(modVal)
	if err != nil {
		return nil, err
	}
	if err := rejectCrossplaneReservedField(configSchema); err != nil {
		return nil, err
	}

	openAPIV3Schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"spec": configSchema,
		},
		"required": []any{"spec"},
	}

	labels, annotations := buildProvenance(modVal, moduleName, moduleVersion)

	xrdMeta := map[string]any{
		"name": names.Plural + "." + opts.Group,
	}
	if len(labels) > 0 {
		xrdMeta["labels"] = toAnyMap(labels)
	}
	if len(annotations) > 0 {
		xrdMeta["annotations"] = toAnyMap(annotations)
	}

	xrd := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": xrdAPIVersion,
			"kind":       xrdKind,
			"metadata":   xrdMeta,
			"spec": map[string]any{
				"scope": string(scope),
				"group": opts.Group,
				"names": map[string]any{
					"kind":     names.Kind,
					"listKind": names.ListKind,
					"plural":   names.Plural,
					"singular": names.Singular,
				},
				"versions": []any{
					map[string]any{
						"name":          version,
						"served":        true,
						"referenceable": true,
						"storage":       true,
						"schema": map[string]any{
							"openAPIV3Schema": openAPIV3Schema,
						},
					},
				},
			},
		},
	}
	return xrd, nil
}

// rejectCrossplaneReservedField returns an error if the extracted config
// schema's top-level properties include a "crossplane" key. Crossplane v2
// reserves spec.crossplane.* and status.crossplane.* for its own use, so a
// module that declares either at the root of #config would produce an XRD
// the API server rejects.
func rejectCrossplaneReservedField(configSchema map[string]any) error {
	props, ok := configSchema["properties"].(map[string]any)
	if !ok {
		return nil
	}
	if _, has := props["crossplane"]; has {
		return fmt.Errorf(
			"#config declares a top-level field named %q which Crossplane v2 reserves for its own use; rename the field",
			"crossplane",
		)
	}
	return nil
}
