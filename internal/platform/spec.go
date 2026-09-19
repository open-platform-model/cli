// Package platform resolves the platform module every render consumes, by
// precedence: --platform <dir> > cluster Platform CR > local default module
// ~/.opm/platform/ (0006:D11/D12/D17/D21/D22; 0019:D5/D7).
//
// Every source resolves to a platform module directory the kernel acquires
// with AcquirePlatformFromDir. The cluster CR is turned into such a module
// first, through the library's generator (opm/helper/platformmodule): the
// same helper and the same acquisition the operator's PlatformReconciler
// runs, so the CLI's platform ingestion is structurally the operator's own.
package platform

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

// Spec is the CLI's typed platform document: what a cluster Platform CR's
// spec carries and what the write-if-absent seed writes back. It is the one
// shape CR decode (DecodeCRSpec) and the seed decoded from a built platform
// (SpecFromPlatform) share.
type Spec struct {
	// Name is the platform name (metadata.name of the CR form).
	Name string
	// Type is the informational platform discriminator (core #Platform.type).
	Type string
	// Entries are the catalog subscriptions in ascending Path order.
	Entries []Entry
	// SkewPolicy is the CR's spec.skewPolicy verbatim ("Warn", "Refuse" or
	// empty when unset). Only DecodeCRSpec fills it; a seed never writes it.
	SkewPolicy string
}

// Entry is one catalog subscription: the major-qualified catalog path (the
// #registry key), the bare SemVer build it names and whether it is enabled.
type Entry struct {
	Path string
	// Version is empty only on a legacy stored CR (see DecodeCRSpec).
	Version string
	Enable  bool
}

// wireSpec is the wire shape of a Platform CR's spec. Name is kept out: the
// CR carries it in metadata.name.
type wireSpec struct {
	Type       string                      `json:"type"`
	Registry   map[string]wireSubscription `json:"registry,omitempty"`
	SkewPolicy string                      `json:"skewPolicy,omitempty"`
}

// wireSubscription mirrors the CR Subscription shape: optional enable plus
// the scalar version naming exactly one catalog build.
type wireSubscription struct {
	Enable  *bool  `json:"enable,omitempty"`
	Version string `json:"version,omitempty"`
}

// toSpec converts the wire shape into the typed Spec. A nil enable resolves
// to the schema default (true). Entries are sorted by path so the result is
// deterministic whatever order the map iterates in.
func (w wireSpec) toSpec(name string) Spec {
	s := Spec{Name: name, Type: w.Type, SkewPolicy: w.SkewPolicy}
	if len(w.Registry) > 0 {
		s.Entries = make([]Entry, 0, len(w.Registry))
		for path, sub := range w.Registry {
			s.Entries = append(s.Entries, Entry{
				Path:    path,
				Version: sub.Version,
				Enable:  sub.Enable == nil || *sub.Enable,
			})
		}
		sort.Slice(s.Entries, func(i, j int) bool { return s.Entries[i].Path < s.Entries[j].Path })
	}
	return s
}

// wireFromSpec converts a Spec into the wire shape the CR carries — the
// document write-if-absent creates (0006:D12). Every entry's enable is written
// explicitly: the Spec came from a built platform where it is concrete, so
// the CR states exactly what the render consumed.
func wireFromSpec(s Spec) wireSpec {
	w := wireSpec{Type: s.Type, SkewPolicy: s.SkewPolicy}
	if len(s.Entries) > 0 {
		w.Registry = make(map[string]wireSubscription, len(s.Entries))
		for _, e := range s.Entries {
			enable := e.Enable
			w.Registry[e.Path] = wireSubscription{Enable: &enable, Version: e.Version}
		}
	}
	return w
}

// DecodeCRSpec decodes a cluster Platform CR's spec (as an unstructured map)
// into a Spec. name is the CR's metadata.name.
//
// Deliberately light validation: the CR spec was already admitted by the
// CRD's OpenAPI schema server-side, so only the one field the CRD cannot
// default (spec.type) is re-checked here. Anything else surfaces from
// platform-module generation or the kernel's acquisition.
//
// Legacy stored shapes are tolerated permanently (this is a CR read, never a
// module): a `filter` key is ignored (json.Unmarshal drops unknown fields),
// and a subscription with no `version` decodes with an empty Version and
// fails only at generation (GenerateClusterModule), wrapped with the
// legacy-CR hint. Stored CRs keep their pre-v2 shape in etcd until their next
// spec write, so this tolerance is not transitional.
func DecodeCRSpec(spec map[string]any, name string) (Spec, error) {
	// JSON round-trip: the CR spec is the wire shape the CRD serializes, so
	// this is an explicit, lossless mapping.
	raw, err := json.Marshal(spec)
	if err != nil {
		return Spec{}, fmt.Errorf("encoding Platform CR spec: %w", err)
	}
	var w wireSpec
	if err := json.Unmarshal(raw, &w); err != nil {
		return Spec{}, fmt.Errorf("decoding Platform CR spec: %w", err)
	}
	if w.Type == "" {
		return Spec{}, fmt.Errorf("cluster Platform %q has no spec.type", name)
	}
	return w.toSpec(name), nil
}

// Registry entry sources, as the operator records them on
// status.registry[].source: an authored subscription, or a catalog an
// accepted-and-active TransformerRegistration contributed (0015:D3).
const (
	EntrySourceSubscription = "Subscription"
	EntrySourceRegistration = "Registration"
)

// ReadyState is the Platform's Ready condition as resolution reads it:
// status and reason only. It is reported, never acted on - a refused
// Platform still has a last good package, and that package is what the
// cluster renders against.
type ReadyState struct {
	Status string
	Reason string
}

// Effective is the registry the operator generated the running package
// from, decoded off the Platform CR's status (0015:D13/D17). Entries is
// empty when the operator has recorded no registry - a solo cluster, or an
// operator predating the field - and resolution then generates from the
// spec.
type Effective struct {
	// Entries are the resolved registry's catalogs in ascending Path order,
	// each carrying the version the operator pinned and whether its
	// transformers register.
	Entries []Entry
	// Sources maps each entry's path to how the catalog reached the
	// platform: EntrySourceSubscription or EntrySourceRegistration.
	Sources map[string]string
	// PackageIdentity is status.packageIdentity verbatim (gen-N[-digest8]),
	// empty when none is recorded.
	PackageIdentity string
	// ObservedGeneration is the CR generation the recorded package was
	// generated from; behind metadata.generation means the spec has moved
	// on since.
	ObservedGeneration int64
	// OperatorVersion is the operator that last patched the status, empty
	// on a solo cluster.
	OperatorVersion string
	// Ready is the Ready condition, nil when absent.
	Ready *ReadyState
}

// wireStatus is the wire shape of a Platform CR's status, limited to the
// fields resolution reads. Unknown fields are dropped by json.Unmarshal, so
// a newer operator's status decodes without error.
type wireStatus struct {
	ObservedGeneration int64               `json:"observedGeneration,omitempty"`
	PackageIdentity    string              `json:"packageIdentity,omitempty"`
	Registry           []wireResolvedEntry `json:"registry,omitempty"`
	Conditions         []wireCondition     `json:"conditions,omitempty"`
	OperatorVersion    string              `json:"operatorVersion,omitempty"`
}

// wireResolvedEntry mirrors the operator's ResolvedRegistryEntry. enabled is
// omitzero on the operator side, so an absent field means a disabled
// catalog - the opposite default from spec.registry's enable.
type wireResolvedEntry struct {
	Catalog string `json:"catalog"`
	Version string `json:"version"`
	Enabled bool   `json:"enabled,omitempty"`
	Source  string `json:"source"`
}

// wireCondition is the part of a metav1.Condition resolution reports.
type wireCondition struct {
	Type   string `json:"type"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

// readyConditionType is the Platform status condition the operator
// summarizes module generation on.
const readyConditionType = "Ready"

// DecodeCR decodes a cluster Platform document into its spec and the
// effective registry the operator recorded on status. The Spec half is
// exactly DecodeCRSpec, legacy tolerance included. The Effective half is nil
// when the CR carries no status at all; otherwise it is returned even with
// no registry rows, so a caller can still report the operator version and
// the generation the operator observed.
//
// A status registry row without a version is refused naming the row: the
// operator never writes one, so it means a hand-edited status, and
// generating from it would silently drop the catalog's pin.
func DecodeCR(doc *ClusterPlatform) (Spec, *Effective, error) {
	if doc == nil {
		return Spec{}, nil, errors.New("no cluster Platform document to decode")
	}
	s, err := DecodeCRSpec(doc.Spec, doc.Name)
	if err != nil {
		return Spec{}, nil, err
	}
	if doc.Status == nil {
		return s, nil, nil
	}

	// JSON round-trip, as DecodeCRSpec does: the status is the wire shape
	// the CRD serializes, so this is an explicit, lossless mapping.
	raw, err := json.Marshal(doc.Status)
	if err != nil {
		return Spec{}, nil, fmt.Errorf("encoding Platform CR status: %w", err)
	}
	var w wireStatus
	if err := json.Unmarshal(raw, &w); err != nil {
		return Spec{}, nil, fmt.Errorf("decoding Platform CR status: %w", err)
	}

	eff := &Effective{
		PackageIdentity:    w.PackageIdentity,
		ObservedGeneration: w.ObservedGeneration,
		OperatorVersion:    w.OperatorVersion,
	}
	if len(w.Registry) > 0 {
		eff.Entries = make([]Entry, 0, len(w.Registry))
		eff.Sources = make(map[string]string, len(w.Registry))
		for _, row := range w.Registry {
			if row.Version == "" {
				return Spec{}, nil, fmt.Errorf(
					"cluster Platform %q status.registry entry %q has no version; the operator never records one, so the status was written by hand",
					doc.Name, row.Catalog)
			}
			eff.Entries = append(eff.Entries, Entry{Path: row.Catalog, Version: row.Version, Enable: row.Enabled})
			eff.Sources[row.Catalog] = row.Source
		}
		sort.Slice(eff.Entries, func(i, j int) bool { return eff.Entries[i].Path < eff.Entries[j].Path })
	}
	for _, c := range w.Conditions {
		if c.Type == readyConditionType {
			eff.Ready = &ReadyState{Status: c.Status, Reason: c.Reason}
			break
		}
	}
	return s, eff, nil
}
