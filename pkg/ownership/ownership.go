package ownership

import "fmt"

const (
	// CreatedByCLI marks an instance as created by the CLI.
	CreatedByCLI = "cli"
	// CreatedByController marks an instance as created by a controller.
	CreatedByController = "controller"
)

// EnsureCLIMutable enforces the no-takeover policy for CLI mutating workflows.
// If the instance was created by a controller, the CLI must refuse to mutate it.
// Empty or unknown createdBy values are treated as CLI-owned (legacy compat).
func EnsureCLIMutable(createdBy, instanceName, instanceNamespace string) error {
	if createdBy != CreatedByController {
		return nil
	}
	return fmt.Errorf("instance %q in namespace %q is controller-managed and cannot be changed by the CLI",
		instanceName,
		instanceNamespace,
	)
}
