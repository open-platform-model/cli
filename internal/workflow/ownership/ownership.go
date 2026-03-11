package ownership

import (
	"fmt"

	"github.com/opmodel/cli/internal/inventory"
)

// EnsureCLIMutable enforces the no-takeover policy for CLI mutating workflows.
func EnsureCLIMutable(inv *inventory.ReleaseInventoryRecord) error {
	if inv == nil {
		return nil
	}
	if inv.NormalizedCreatedBy() != inventory.CreatedByController {
		return nil
	}
	return fmt.Errorf("release %q in namespace %q is controller-managed and cannot be changed by the CLI",
		inv.ReleaseMetadata.ReleaseName,
		inv.ReleaseMetadata.ReleaseNamespace,
	)
}
