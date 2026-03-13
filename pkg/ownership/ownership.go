package ownership

import "fmt"

const (
	// CreatedByCLI marks a release as created by the CLI.
	CreatedByCLI = "cli"
	// CreatedByController marks a release as created by a controller.
	CreatedByController = "controller"
)

// EnsureCLIMutable enforces the no-takeover policy for CLI mutating workflows.
// If the release was created by a controller, the CLI must refuse to mutate it.
// Empty or unknown createdBy values are treated as CLI-owned (legacy compat).
func EnsureCLIMutable(createdBy, releaseName, releaseNamespace string) error {
	if createdBy != CreatedByController {
		return nil
	}
	return fmt.Errorf("release %q in namespace %q is controller-managed and cannot be changed by the CLI",
		releaseName,
		releaseNamespace,
	)
}
