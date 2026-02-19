package build

import "github.com/opmodel/cli/internal/build/transform"

// TransformError re-exports transform.TransformError for backward compatibility.
// Used by external packages (e.g., cmdutil) to detect transform failures.
type TransformError = transform.TransformError

// TransformerRequirements re-exports transform.TransformerRequirements for backward compatibility.
type TransformerRequirements = transform.TransformerRequirements
