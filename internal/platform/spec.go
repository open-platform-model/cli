// Package platform resolves the platform spec every render consumes, by
// precedence: --platform flag file > cluster Platform CR > local default
// ~/.opm/platform.cue (enhancement 0006 D11/D12/D17/D21/D22/D39).
//
// All three sources decode through one wire mapping into synth.PlatformInput
// and materialize via the same kernel calls the operator's PlatformReconciler
// makes, so the CLI's platform ingestion is structurally the operator's own.
package platform

import (
	"encoding/json"
	"fmt"
	"os"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"

	"github.com/open-platform-model/library/opm/helper/synth"

	"github.com/open-platform-model/cli/internal/config"
)

// wireSpec is the shared wire shape of a platform spec: the data-only
// ~/.opm/platform.cue projection and the Platform CR's spec are the same
// document (the file additionally carries name, which the CR keeps in
// metadata.name).
type wireSpec struct {
	Name     string                      `json:"name,omitempty"`
	Type     string                      `json:"type"`
	Registry map[string]wireSubscription `json:"registry,omitempty"`
}

// wireSubscription mirrors core #Subscription / the CR Subscription shape.
type wireSubscription struct {
	Enable *bool       `json:"enable,omitempty"`
	Filter *wireFilter `json:"filter,omitempty"`
}

// wireFilter mirrors core #SubscriptionFilter.
type wireFilter struct {
	Range string   `json:"range,omitempty"`
	Allow []string `json:"allow,omitempty"`
	Deny  []string `json:"deny,omitempty"`
}

// toInput converts the wire shape into the kernel's typed platform input.
// SchemaCache is left nil: Kernel.SynthesizePlatform defaults it to the
// kernel-owned cache.
func (w wireSpec) toInput() synth.PlatformInput {
	in := synth.PlatformInput{
		Name: w.Name,
		Type: w.Type,
	}
	if len(w.Registry) > 0 {
		in.Subscriptions = make(map[string]synth.SubscriptionSpec, len(w.Registry))
		for path, sub := range w.Registry {
			spec := synth.SubscriptionSpec{Enable: sub.Enable}
			if sub.Filter != nil {
				spec.Filter = &synth.FilterSpec{
					Range: sub.Filter.Range,
					Allow: sub.Filter.Allow,
					Deny:  sub.Filter.Deny,
				}
			}
			in.Subscriptions[path] = spec
		}
	}
	return in
}

// DecodeFile validates the platform file at path (data-only, embedded
// projection schema — config.ValidatePlatformFile) and decodes it into a
// synth.PlatformInput.
func DecodeFile(path string) (synth.PlatformInput, error) {
	if err := config.ValidatePlatformFile(path); err != nil {
		return synth.PlatformInput{}, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return synth.PlatformInput{}, fmt.Errorf("reading platform file: %w", err)
	}

	value := cuecontext.New().CompileBytes(content, cue.Filename(path))
	if value.Err() != nil {
		return synth.PlatformInput{}, fmt.Errorf("compiling platform file %s: %w", path, value.Err())
	}

	var w wireSpec
	if err := value.Decode(&w); err != nil {
		return synth.PlatformInput{}, fmt.Errorf("decoding platform file %s: %w", path, err)
	}
	return w.toInput(), nil
}

// DecodeCRSpec decodes a cluster Platform CR's spec (as an unstructured map)
// into a synth.PlatformInput. name is the CR's metadata.name.
func DecodeCRSpec(spec map[string]any, name string) (synth.PlatformInput, error) {
	// JSON round-trip: the CR spec is the same wire shape, produced by the
	// CRD's serialization, so this is an explicit, lossless mapping.
	raw, err := json.Marshal(spec)
	if err != nil {
		return synth.PlatformInput{}, fmt.Errorf("encoding Platform CR spec: %w", err)
	}
	var w wireSpec
	if err := json.Unmarshal(raw, &w); err != nil {
		return synth.PlatformInput{}, fmt.Errorf("decoding Platform CR spec: %w", err)
	}
	if w.Type == "" {
		return synth.PlatformInput{}, fmt.Errorf("cluster Platform %q has no spec.type", name)
	}
	w.Name = name
	return w.toInput(), nil
}

// wireFromInput converts a typed platform input back into the wire shape —
// the inverse of toInput, used to build the cluster Platform document for
// write-if-absent (D12).
func wireFromInput(in synth.PlatformInput) wireSpec {
	w := wireSpec{
		Name: in.Name,
		Type: in.Type,
	}
	if len(in.Subscriptions) > 0 {
		w.Registry = make(map[string]wireSubscription, len(in.Subscriptions))
		for path, sub := range in.Subscriptions {
			ws := wireSubscription{Enable: sub.Enable}
			if sub.Filter != nil {
				ws.Filter = &wireFilter{
					Range: sub.Filter.Range,
					Allow: sub.Filter.Allow,
					Deny:  sub.Filter.Deny,
				}
			}
			w.Registry[path] = ws
		}
	}
	return w
}
