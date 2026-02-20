// Package release provides release building functionality for OPM modules.
package release

import (
	"cuelang.org/go/cue"

	"github.com/opmodel/cli/internal/build/component"
	"github.com/opmodel/cli/internal/core"
)

// Options configures release building.
type Options struct {
	Name      string // Required: release name
	Namespace string // Required: target namespace
	PkgName   string // Internal: CUE package name (set by InspectModule, skip detectPackageName)
}

// BuiltRelease is the result of building a release.
type BuiltRelease struct {
	Value           cue.Value                       // The concrete module value (with #config injected)
	Components      map[string]*component.Component // Concrete components by name
	ReleaseMetadata core.ReleaseMetadata
	ModuleMetadata  core.ModuleMetadata
}
