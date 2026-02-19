package build

import (
	"github.com/opmodel/cli/internal/build/module"
	"github.com/opmodel/cli/internal/build/release"
)

// ReleaseMetadata re-exports release.ReleaseMetadata as the public release-level metadata type.
type ReleaseMetadata = release.ReleaseMetadata

// ModuleMetadata re-exports module.ModuleMetadata as the public module-level metadata type.
type ModuleMetadata = module.ModuleMetadata

// ReleaseValidationError re-exports release.ValidationError for backward compatibility.
type ReleaseValidationError = release.ValidationError
