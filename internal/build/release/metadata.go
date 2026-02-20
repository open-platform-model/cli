package release

import (
	"github.com/opmodel/cli/internal/core"
)

// extractReleaseMetadata constructs a core.ReleaseMetadata from the module's
// Go-level metadata, build options, and a deterministic UUID computed via
// core.ComputeReleaseUUID(). No CUE path lookup needed.
//
// The release UUID is deterministic: same (fqn, name, namespace) always yields
// the same UUID v5, enabling stable resource identity across re-applies.
func extractReleaseMetadata(mod *core.Module, opts Options) core.ReleaseMetadata {
	fqn := ""
	labels := make(map[string]string)

	if mod.Metadata != nil {
		fqn = mod.Metadata.FQN
		// Copy module labels as the base set for the release
		for k, v := range mod.Metadata.Labels {
			labels[k] = v
		}
	}

	// Compute deterministic release UUID from (fqn, name, namespace)
	releaseUUID := core.ComputeReleaseUUID(fqn, opts.Name, opts.Namespace)

	// Add standard release labels
	labels[core.LabelReleaseName] = opts.Name
	labels[core.LabelReleaseUUID] = releaseUUID

	return core.ReleaseMetadata{
		Name:      opts.Name,
		Namespace: opts.Namespace,
		UUID:      releaseUUID,
		Labels:    labels,
	}
}
