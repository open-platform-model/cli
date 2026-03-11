package inventory

import pkginventory "github.com/opmodel/cli/pkg/inventory"

type (
	CreatedBy       = pkginventory.CreatedBy
	InventoryEntry  = pkginventory.InventoryEntry //nolint:revive // compatibility alias while contract moves to pkg/inventory
	ChangeEntry     = pkginventory.ChangeEntry
	ChangeSource    = pkginventory.ChangeSource
	InventoryList   = pkginventory.InventoryList //nolint:revive // compatibility alias while contract moves to pkg/inventory
	ReleaseMetadata = pkginventory.ReleaseMetadata
	ModuleMetadata  = pkginventory.ModuleMetadata
	InventorySecret = pkginventory.InventorySecret //nolint:revive // compatibility alias while contract moves to pkg/inventory
)

const (
	CreatedByCLI        = pkginventory.CreatedByCLI
	CreatedByController = pkginventory.CreatedByController
)

var (
	NormalizeCreatedBy   = pkginventory.NormalizeCreatedBy
	SecretName           = pkginventory.SecretName
	InventoryLabels      = pkginventory.InventoryLabels
	MarshalToSecret      = pkginventory.MarshalToSecret
	UnmarshalFromSecret  = pkginventory.UnmarshalFromSecret
	ComputeChangeID      = pkginventory.ComputeChangeID
	UpdateIndex          = pkginventory.UpdateIndex
	PruneHistory         = pkginventory.PruneHistory
	PrepareChange        = pkginventory.PrepareChange
	NewEntryFromResource = pkginventory.NewEntryFromResource
	IdentityEqual        = pkginventory.IdentityEqual
	K8sIdentityEqual     = pkginventory.K8sIdentityEqual
)
