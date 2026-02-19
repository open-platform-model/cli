package build

import "github.com/opmodel/cli/internal/build/module"

// LoadedComponent is a component with extracted metadata.
// Components are extracted by ReleaseBuilder during the build phase.
// The type is defined in build/module and re-exported here for backward compatibility.
type LoadedComponent = module.LoadedComponent
