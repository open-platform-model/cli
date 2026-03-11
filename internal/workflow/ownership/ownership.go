package ownership

import (
	"fmt"

	pkginventory "github.com/opmodel/cli/pkg/inventory"
)

// EnsureCLIMutable enforces the no-takeover policy for CLI mutating workflows.
func EnsureCLIMutable(inv *pkginventory.InventorySecret) error {
	if inv == nil {
		return nil
	}
	if inv.ReleaseMetadata.NormalizedCreatedBy() != pkginventory.CreatedByController {
		return nil
	}
	return fmt.Errorf("release %q in namespace %q is controller-managed and cannot be changed by the CLI",
		inv.ReleaseMetadata.ReleaseName,
		inv.ReleaseMetadata.ReleaseNamespace,
	)
}
