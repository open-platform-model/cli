// Package platform resolves the platform spec every render consumes, by
// precedence: --platform flag file > cluster Platform CR > local default
// (enhancement 0006 D11/D12/D17/D21/D22/D39).
//
// Transitional: the local default still reads the legacy data-only
// platform.cue beside the config file. `opm config init` writes the module
// form ~/.opm/platform/ since cli-config-platform-module (0019 D5);
// cli-render-switch moves this package onto platform module directories.
//
// All three sources decode through one wire mapping into synth.PlatformInput
// and materialize via the same kernel calls the operator's PlatformReconciler
// makes, so the CLI's platform ingestion is structurally the operator's own.
package platform

import (
	"encoding/json"
	"fmt"
	"strings"

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

// wireSubscription mirrors core #Subscription / the CR Subscription shape:
// optional enable plus the required scalar version naming exactly one
// catalog build. Version is empty only on legacy stored CRs (see
// DecodeCRSpec); synthesis enforces it via ErrSubscriptionMissingVersion.
type wireSubscription struct {
	Enable  *bool  `json:"enable,omitempty"`
	Version string `json:"version,omitempty"`
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
			in.Subscriptions[path] = synth.SubscriptionSpec{
				Enable:  sub.Enable,
				Version: sub.Version,
			}
		}
	}
	return in
}

// LegacyDefaultPlatformFile is the pre-0019 data-only local platform file
// (what `opm config init` wrote before the local default became the module
// at ~/.opm/platform/), pinning the same catalog builds as
// config.DefaultCatalogPins so it never drifts from the seeded module.
//
// It keeps this render path and its integration mains exercisable while
// resolution still reads the data shape; cli-render-switch moves resolution
// onto platform module directories and deletes it together with DecodeFile.
var LegacyDefaultPlatformFile = fmt.Sprintf(`name: "cluster"
type: "kubernetes"

registry: {
	%q: {
		version: %q
	}
	%q: {
		version: %q
	}
}
`, config.DefaultCatalogPaths[0], strings.TrimPrefix(config.DefaultCatalogPins[0], "v"),
	config.DefaultCatalogPaths[1], strings.TrimPrefix(config.DefaultCatalogPins[1], "v"))

// DecodeFile validates the legacy data-only platform file at path (embedded
// projection schema — config.LoadLegacyPlatformFile, one read/compile) and
// decodes it into a synth.PlatformInput. Deleted by cli-render-switch, which
// acquires platform module directories instead.
func DecodeFile(path string) (synth.PlatformInput, error) {
	value, err := config.LoadLegacyPlatformFile(path)
	if err != nil {
		return synth.PlatformInput{}, err
	}

	var w wireSpec
	if err := value.Decode(&w); err != nil {
		return synth.PlatformInput{}, fmt.Errorf("decoding platform file %s: %w", path, err)
	}
	return w.toInput(), nil
}

// DecodeCRSpec decodes a cluster Platform CR's spec (as an unstructured map)
// into a synth.PlatformInput. name is the CR's metadata.name.
//
// Deliberately lighter validation than DecodeFile: the CR spec was already
// admitted by the CRD's OpenAPI schema server-side, so only the one field the
// CRD cannot default (spec.type) is re-checked here. Shape errors that slip
// through surface from Materialize.
//
// Legacy stored shapes are tolerated permanently (never for files): a `filter`
// key is ignored (json.Unmarshal drops unknown fields), and a subscription
// with no `version` decodes with Version "" and fails only at synthesis via
// the kernel's ErrSubscriptionMissingVersion — wrapped with the legacy-CR hint
// by WrapClusterMaterializeError. Stored CRs keep their pre-v2 shape in etcd
// until their next spec write, so this tolerance is not transitional.
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
			w.Registry[path] = wireSubscription{
				Enable:  sub.Enable,
				Version: sub.Version,
			}
		}
	}
	return w
}
