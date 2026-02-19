package build

import (
	"github.com/opmodel/cli/internal/build/module"
	"github.com/opmodel/cli/internal/build/release"
)

// ReleaseBuilder re-exports the release.Builder type for backward compatibility.
// The type is defined in build/release and re-exported here as a type alias.
type ReleaseBuilder = release.Builder

// ReleaseOptions re-exports release.Options for backward compatibility.
type ReleaseOptions = release.Options

// BuiltRelease re-exports release.BuiltRelease for backward compatibility.
type BuiltRelease = release.BuiltRelease

// ReleaseMetadata re-exports release.ReleaseMetadata as the public release-level metadata type.
type ReleaseMetadata = release.ReleaseMetadata

// ModuleMetadata re-exports module.ModuleMetadata as the public module-level metadata type.
type ModuleMetadata = module.ModuleMetadata

// ReleaseValidationError re-exports release.ValidationError for backward compatibility.
type ReleaseValidationError = release.ValidationError

// NewReleaseBuilder re-exports release.NewBuilder for backward compatibility.
var NewReleaseBuilder = release.NewBuilder
