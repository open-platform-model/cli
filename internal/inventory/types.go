package inventory

import (
	pkginventory "github.com/opmodel/cli/pkg/inventory"
	"github.com/opmodel/cli/pkg/ownership"
)

type CreatedBy string

const (
	CreatedByCLI        CreatedBy = CreatedBy(ownership.CreatedByCLI)
	CreatedByController CreatedBy = CreatedBy(ownership.CreatedByController)
)

// APIVersionV1Alpha1 is the apiVersion stamped on persisted instance/module
// metadata, enabling future migration from Secrets to CRDs.
const APIVersionV1Alpha1 = "core.opmodel.dev/v1alpha1"

func NormalizeCreatedBy(createdBy CreatedBy) CreatedBy {
	if createdBy == CreatedByController {
		return CreatedByController
	}
	return CreatedByCLI
}

type InstanceMetadata struct {
	Kind               string `json:"kind"`
	APIVersion         string `json:"apiVersion"`
	InstanceName       string `json:"name"`
	InstanceNamespace  string `json:"namespace"`
	InstanceID         string `json:"uuid"`
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
}

type ModuleMetadata struct {
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
	Name       string `json:"name"`
	UUID       string `json:"uuid,omitempty"`
	Version    string `json:"version,omitempty"`
}

type InstanceInventoryRecord struct {
	CreatedBy        CreatedBy              `json:"createdBy,omitempty"`
	InstanceMetadata InstanceMetadata       `json:"instanceMetadata"`
	ModuleMetadata   ModuleMetadata         `json:"moduleMetadata"`
	Inventory        pkginventory.Inventory `json:"inventory"`

	resourceVersion string
}

func (r *InstanceInventoryRecord) ResourceVersion() string {
	return r.resourceVersion
}

func (r *InstanceInventoryRecord) SetResourceVersion(resourceVersion string) {
	r.resourceVersion = resourceVersion
}

func (r *InstanceInventoryRecord) NormalizedCreatedBy() CreatedBy {
	return NormalizeCreatedBy(r.CreatedBy)
}
