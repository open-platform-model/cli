package inventory

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	pkgcore "github.com/open-platform-model/cli/pkg/core"
)

// The ModuleInstance and Platform CRD coordinates. These are hardcoded rather
// than imported from opm-operator's types package — the CLI has no Go module
// dependency on opm-operator (enhancement 0006 D13). This package is the single
// definition; other CLI packages (e.g. internal/operator) consume it from here
// instead of keeping private copies.
const (
	// GroupOpmodel is the API group for the ModuleInstance and Platform CRDs.
	GroupOpmodel = "opmodel.dev"
	// VersionV1Alpha1 is the served/storage version of both CRDs.
	VersionV1Alpha1 = "v1alpha1"

	// KindModuleInstance is the ModuleInstance CRD kind.
	KindModuleInstance = "ModuleInstance"
	// ResourceModuleInstances is the ModuleInstance CRD plural resource name.
	ResourceModuleInstances = "moduleinstances"
	// CRDNameModuleInstances is the ModuleInstance CustomResourceDefinition name.
	CRDNameModuleInstances = "moduleinstances.opmodel.dev"

	// KindPlatform is the Platform CRD kind.
	KindPlatform = "Platform"
	// ResourcePlatforms is the Platform CRD plural resource name.
	ResourcePlatforms = "platforms"
	// PlatformSingletonName is the only permitted name of the cluster-scoped
	// Platform singleton.
	PlatformSingletonName = "cluster"

	// APIVersionModuleInstance is the apiVersion string written on the CR document.
	APIVersionModuleInstance = GroupOpmodel + "/" + VersionV1Alpha1
)

// Ownership marker values (spec.owner), matching the CRD's enum.
const (
	// OwnerCLI marks an instance the CLI manages as the direct-resource executor.
	OwnerCLI = "cli"
	// OwnerOperator marks an instance the operator reconciles.
	OwnerOperator = "operator"
)

const (
	// AnnotationSource is the render-provenance annotation. Value SourceLocal
	// signals the last apply's module bytes did not come from pure registry
	// resolution. It is a fail-closed signal consumed by the handoff pre-gate
	// (slice C3); no gate in this slice reads it as an authority.
	AnnotationSource = "module-instance.opmodel.dev/source"
	// SourceLocal is the AnnotationSource value stamped for local renders.
	SourceLocal = "local"
)

// LabelInstanceUUID is the label the render stamps on every resource carrying
// the deterministic instance UUID; status.instanceUUID is extracted from it.
const LabelInstanceUUID = pkgcore.LabelModuleInstanceUUID

// ModuleInstanceGVR is the ModuleInstance CRD's GroupVersionResource.
var ModuleInstanceGVR = schema.GroupVersionResource{
	Group:    GroupOpmodel,
	Version:  VersionV1Alpha1,
	Resource: ResourceModuleInstances,
}

// PlatformGVR is the Platform CRD's GroupVersionResource.
var PlatformGVR = schema.GroupVersionResource{
	Group:    GroupOpmodel,
	Version:  VersionV1Alpha1,
	Resource: ResourcePlatforms,
}
