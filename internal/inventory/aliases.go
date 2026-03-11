package inventory

import pkginventory "github.com/opmodel/cli/pkg/inventory"

type (
	InventoryEntry = pkginventory.InventoryEntry //nolint:revive // compatibility alias while contract moves to pkg/inventory
	Inventory      = pkginventory.Inventory
)

var (
	NewEntryFromResource = pkginventory.NewEntryFromResource
	IdentityEqual        = pkginventory.IdentityEqual
	K8sIdentityEqual     = pkginventory.K8sIdentityEqual
	ComputeStaleSet      = pkginventory.ComputeStaleSet
	ComputeDigest        = pkginventory.ComputeDigest
)
