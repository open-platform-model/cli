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

func NormalizeCreatedBy(createdBy CreatedBy) CreatedBy {
	if createdBy == CreatedByController {
		return CreatedByController
	}
	return CreatedByCLI
}

type ReleaseMetadata struct {
	Kind               string `json:"kind"`
	APIVersion         string `json:"apiVersion"`
	ReleaseName        string `json:"name"`
	ReleaseNamespace   string `json:"namespace"`
	ReleaseID          string `json:"uuid"`
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
}

type ModuleMetadata struct {
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
	Name       string `json:"name"`
	UUID       string `json:"uuid,omitempty"`
	Version    string `json:"version,omitempty"`
}

type ReleaseInventoryRecord struct {
	CreatedBy       CreatedBy              `json:"createdBy,omitempty"`
	ReleaseMetadata ReleaseMetadata        `json:"releaseMetadata"`
	ModuleMetadata  ModuleMetadata         `json:"moduleMetadata"`
	Inventory       pkginventory.Inventory `json:"inventory"`

	resourceVersion string
}

func (r *ReleaseInventoryRecord) ResourceVersion() string {
	return r.resourceVersion
}

func (r *ReleaseInventoryRecord) SetResourceVersion(resourceVersion string) {
	r.resourceVersion = resourceVersion
}

func (r *ReleaseInventoryRecord) NormalizedCreatedBy() CreatedBy {
	return NormalizeCreatedBy(r.CreatedBy)
}
