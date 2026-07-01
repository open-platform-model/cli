package module

import (
	"context"
	"fmt"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/cli/pkg/validate"
)

// ParseModuleInstance validates values, fills them into the instance spec,
// ensures the result is concrete, decodes metadata, and constructs Instance.
//
// Was: ParseModuleRelease (enhancement 0002 D8 hard-rename).
func ParseModuleInstance(_ context.Context, spec cue.Value, mod Module, values []cue.Value) (*Instance, error) {
	// Best-effort name for error messages — metadata.name may already be
	// concrete before values filling (it comes from the module definition).
	name := bestEffortInstanceName(spec, mod)

	// Validate and merge values against the module config schema.
	merged, cfgErr := validate.Config(mod.Config, values, "module", name)
	if cfgErr != nil {
		return nil, cfgErr
	}

	// Fill merged values into the instance spec.
	if merged.Exists() {
		spec = spec.FillPath(cue.ParsePath("values"), merged)
		if err := spec.Err(); err != nil {
			return nil, fmt.Errorf("filling values into instance spec: %w", err)
		}
	}

	// Validate the filled spec is fully concrete.
	if err := spec.Validate(cue.Concrete(true)); err != nil {
		return nil, fmt.Errorf("instance %q: not fully concrete: %w", name, err)
	}

	// Decode instance metadata from the concrete spec.
	metadata, err := decodeInstanceMetadata(spec, name)
	if err != nil {
		return nil, err
	}

	return &Instance{
		Metadata: metadata,
		Module:   mod,
		Spec:     spec,
		Values:   merged,
	}, nil
}

// decodeInstanceMetadata extracts and decodes InstanceMetadata from a concrete spec value.
func decodeInstanceMetadata(spec cue.Value, name string) (*InstanceMetadata, error) {
	metaVal := spec.LookupPath(cue.ParsePath("metadata"))
	if !metaVal.Exists() {
		return nil, fmt.Errorf("instance %q: metadata field is required", name)
	}
	meta := &InstanceMetadata{}
	if err := metaVal.Decode(meta); err != nil {
		return nil, fmt.Errorf("instance %q: decoding metadata: %w", name, err)
	}
	return meta, nil
}

// bestEffortInstanceName tries to extract an instance name for error messages.
// Falls back to the module name if the instance name is not yet available.
func bestEffortInstanceName(spec cue.Value, mod Module) string {
	nameVal := spec.LookupPath(cue.ParsePath("metadata.name"))
	if nameVal.Exists() {
		if s, err := nameVal.String(); err == nil {
			return s
		}
	}
	if mod.Metadata != nil && mod.Metadata.Name != "" {
		return mod.Metadata.Name
	}
	return "<unknown>"
}
