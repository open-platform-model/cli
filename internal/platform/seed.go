package platform

import (
	"errors"
	"fmt"
	"sort"

	"cuelang.org/go/cue"

	libplatform "github.com/open-platform-model/library/opm/platform"
)

// registryPath is the #Platform.#registry definition: the path-keyed map of
// catalog entries whose enable and derived version the seed reads.
var registryPath = cue.MakePath(cue.Def("registry"))

// SpecFromPlatform decodes the seed document from a built platform value
// (D12): metadata.name and type from the decoded metadata, and for each
// #registry entry its key, enable and the version core derived from the
// embedded catalog. The module the render consumed is the source of truth,
// so the derived version is the only correct one; parsing cue.mod would be a
// second answer. SkewPolicy is never set: the seed writes no policy.
func SpecFromPlatform(p *libplatform.Platform) (Spec, error) {
	if p == nil || p.Metadata == nil {
		return Spec{}, errors.New("platform has no decoded metadata")
	}
	s := Spec{Name: p.Metadata.Name, Type: p.Metadata.Type}
	if s.Type == "" {
		return Spec{}, fmt.Errorf("platform %q has no type", s.Name)
	}

	reg := p.Package.LookupPath(registryPath)
	if !reg.Exists() {
		return s, nil
	}
	it, err := reg.Fields()
	if err != nil {
		return Spec{}, fmt.Errorf("reading platform #registry: %w", err)
	}
	for it.Next() {
		path := it.Selector().String()
		if it.Selector().LabelType() == cue.StringLabel {
			path = it.Selector().Unquoted()
		}
		var e struct {
			Enable  bool   `json:"enable"`
			Version string `json:"version"`
		}
		if err := it.Value().Decode(&e); err != nil {
			return Spec{}, fmt.Errorf("reading platform #registry entry %q: %w", path, err)
		}
		if e.Version == "" {
			return Spec{}, fmt.Errorf("platform #registry entry %q has no derived version", path)
		}
		s.Entries = append(s.Entries, Entry{Path: path, Version: e.Version, Enable: e.Enable})
	}
	sort.Slice(s.Entries, func(i, j int) bool { return s.Entries[i].Path < s.Entries[j].Path })
	return s, nil
}
