package build

import "github.com/opmodel/cli/internal/build/release"

// ReleaseBuilder re-exports the release.Builder type for backward compatibility.
// The type is defined in build/release and re-exported here as a type alias.
type ReleaseBuilder = release.Builder

// ReleaseOptions re-exports release.Options for backward compatibility.
type ReleaseOptions = release.Options

// ModuleInspection re-exports module.Inspection via the release package for backward compatibility.
// Use build/module.Inspection directly for new code.

// BuiltRelease re-exports release.BuiltRelease for backward compatibility.
type BuiltRelease = release.BuiltRelease

// ReleaseMetadata re-exports release.Metadata for backward compatibility.
type ReleaseMetadata = release.Metadata

// ReleaseValidationError re-exports release.ValidationError for backward compatibility.
type ReleaseValidationError = release.ValidationError

// NewReleaseBuilder re-exports release.NewBuilder for backward compatibility.
var NewReleaseBuilder = release.NewBuilder
