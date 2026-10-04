package inventory

import pkginventory "github.com/open-platform-model/cli/pkg/inventory"

type (
	InventoryEntry = pkginventory.InventoryEntry //nolint:revive // compatibility alias while contract moves to pkg/inventory
	Inventory      = pkginventory.Inventory
	K8sIdentity    = pkginventory.K8sIdentity
	AdmitSet       = pkginventory.AdmitSet
)

var (
	NewEntryFromResource = pkginventory.NewEntryFromResource
	IdentityEqual        = pkginventory.IdentityEqual
	K8sIdentityEqual     = pkginventory.K8sIdentityEqual
	ComputeStaleSet      = pkginventory.ComputeStaleSet
	ComputeDigest        = pkginventory.ComputeDigest
)
