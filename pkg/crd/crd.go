package crd

import (
	"fmt"

	"cuelang.org/go/cue"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/opmodel/cli/pkg/core"
)

// apiextensions/v1 API constants. Inlined rather than importing
// k8s.io/apiextensions-apiserver, which is not a direct dependency of this
// repo; the emitted manifest is an unstructured.Unstructured anyway.
const (
	crdAPIVersion   = "apiextensions.k8s.io/v1"
	crdKind         = "CustomResourceDefinition"
	scopeNamespaced = "Namespaced"
)

// Provenance label and annotation keys stamped on generated CRDs so callers
// can trace a live CRD back to the OPM module it was emitted from. Labels
// are for selection; annotations carry descriptive data.
//
// Defined locally for now; consolidate into pkg/core/labels.go if another
// package needs to read these keys.
const (
	labelModuleName    = "module.opmodel.dev/name"
	labelModuleVersion = "module.opmodel.dev/version"

	annotationModulePath        = "module.opmodel.dev/path"
	annotationModuleFQN         = "module.opmodel.dev/fqn"
	annotationModuleDescription = "module.opmodel.dev/description"
	annotationModuleUUID        = "module.opmodel.dev/uuid"
)

// Options configures CRD generation.
type Options struct {
	// Group is the API group the CRD will be registered under (e.g.
	// "module.opmodel.dev"). Required.
	Group string
}

// BuildCRD produces a CustomResourceDefinition for the module's #config
// definition. The returned manifest targets apiextensions.k8s.io/v1 and is
// always Namespaced-scoped without a status subresource (POC scope).
//
// Field sources:
//   - group: opts.Group (required).
//   - names: derived from metadata.name via DeriveNames.
//   - version: derived from metadata.version via DeriveVersion.
//   - schema: from #config via ExtractConfigSchema.
//   - scope: always "Namespaced".
//   - metadata.name: "<plural>.<group>" (the canonical CRD name convention).
func BuildCRD(modVal cue.Value, opts Options) (*unstructured.Unstructured, error) {
	if !modVal.Exists() {
		return nil, fmt.Errorf("module value does not exist")
	}
	if opts.Group == "" {
		return nil, fmt.Errorf("CRD group is required")
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
	schema, err := ExtractConfigSchema(modVal)
	if err != nil {
		return nil, err
	}

	labels, annotations := buildProvenance(modVal, moduleName, moduleVersion)

	crdMeta := map[string]any{
		"name": names.Plural + "." + opts.Group,
	}
	if len(labels) > 0 {
		crdMeta["labels"] = toAnyMap(labels)
	}
	if len(annotations) > 0 {
		crdMeta["annotations"] = toAnyMap(annotations)
	}

	crd := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": crdAPIVersion,
			"kind":       crdKind,
			"metadata":   crdMeta,
			"spec": map[string]any{
				"group": opts.Group,
				"names": map[string]any{
					"kind":     names.Kind,
					"listKind": names.ListKind,
					"plural":   names.Plural,
					"singular": names.Singular,
				},
				"scope": scopeNamespaced,
				"versions": []any{
					map[string]any{
						"name":    version,
						"served":  true,
						"storage": true,
						"schema": map[string]any{
							"openAPIV3Schema": schema,
						},
					},
				},
			},
		},
	}
	return crd, nil
}

// lookupString resolves a cue.Path and returns the concrete string value.
// Returns a descriptive error when the field is missing, non-string, or empty.
func lookupString(v cue.Value, path string) (string, error) {
	field := v.LookupPath(cue.ParsePath(path))
	if !field.Exists() {
		return "", fmt.Errorf("%s is not set on module", path)
	}
	s, err := field.String()
	if err != nil {
		return "", fmt.Errorf("%s must be a string: %w", path, err)
	}
	if s == "" {
		return "", fmt.Errorf("%s is empty", path)
	}
	return s, nil
}

// lookupOptionalString resolves a cue.Path and returns the string value, or
// "" if the field is missing, not concrete, or not a string. Non-fatal —
// optional provenance fields should not break CRD emission.
func lookupOptionalString(v cue.Value, path string) string {
	field := v.LookupPath(cue.ParsePath(path))
	if !field.Exists() {
		return ""
	}
	s, err := field.String()
	if err != nil {
		return ""
	}
	return s
}

// lookupStringMap decodes a struct-valued field at path into a map of string
// keys to string values. Returns nil if the field is absent, not a struct,
// or cannot be decoded. Non-fatal for the same reason as lookupOptionalString.
func lookupStringMap(v cue.Value, path string) map[string]string {
	field := v.LookupPath(cue.ParsePath(path))
	if !field.Exists() {
		return nil
	}
	iter, err := field.Fields()
	if err != nil {
		return nil
	}
	out := map[string]string{}
	for iter.Next() {
		// Unquoted() strips the surrounding quotes that CUE prints for
		// selectors with dots or hyphens (e.g. "app.kubernetes.io/component"),
		// but only string selectors support it. Skip anything else.
		sel := iter.Selector()
		if !sel.IsString() {
			continue
		}
		s, err := iter.Value().String()
		if err != nil {
			continue
		}
		out[sel.Unquoted()] = s
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// buildProvenance composes the labels and annotations stamped on the CRD so
// that the generated manifest points back at its source module.
//
// Labels carry selector-friendly identity (managed-by + module name/version);
// annotations carry descriptive data (path, fqn, description, uuid).
// Module-level metadata.labels and metadata.annotations are passed through
// as the base layer, with OPM-owned keys overlaid last so module authors
// cannot accidentally shadow, e.g., app.kubernetes.io/managed-by.
func buildProvenance(modVal cue.Value, moduleName, moduleVersion string) (labels, annotations map[string]string) {
	labels = map[string]string{}
	for k, v := range lookupStringMap(modVal, "metadata.labels") {
		labels[k] = v
	}
	labels[core.LabelManagedBy] = core.LabelManagedByValue
	labels[labelModuleName] = moduleName
	labels[labelModuleVersion] = moduleVersion

	annotations = map[string]string{}
	for k, v := range lookupStringMap(modVal, "metadata.annotations") {
		annotations[k] = v
	}
	if path := lookupOptionalString(modVal, "metadata.modulePath"); path != "" {
		annotations[annotationModulePath] = path
	}
	if fqn := lookupOptionalString(modVal, "metadata.fqn"); fqn != "" {
		annotations[annotationModuleFQN] = fqn
	}
	if desc := lookupOptionalString(modVal, "metadata.description"); desc != "" {
		annotations[annotationModuleDescription] = desc
	}
	if uuid := lookupOptionalString(modVal, "metadata.uuid"); uuid != "" {
		annotations[annotationModuleUUID] = uuid
	}

	if len(annotations) == 0 {
		annotations = nil
	}
	return labels, annotations
}

// toAnyMap converts a map[string]string into map[string]any so it can be
// embedded in an unstructured.Unstructured object.
func toAnyMap(m map[string]string) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
