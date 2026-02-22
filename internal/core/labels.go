// Package core provides shared primitives used across OPM packages:
// resource identifiers (Resource), label constants, UUID helpers, and weights.
// Domain types (Module, ModuleRelease, Component, Transformer, Provider) live in
// sub-packages: core/module, core/modulerelease, core/component, core/transformer,
// core/provider.
// This package depends only on stdlib — no CUE, no internal packages.
package core

// OPM Kubernetes label keys applied to all managed resources.
const (
	// LabelManagedBy is the standard Kubernetes label indicating the manager.
	// Value is always "open-platform-model".
	LabelManagedBy = "app.kubernetes.io/managed-by"

	// LabelManagedByValue is the value for the LabelManagedBy label.
	LabelManagedByValue = "open-platform-model"

	// LabelReleaseName is the release name label.
	LabelReleaseName = "module-release.opmodel.dev/name"

	// LabelReleaseNamespace is the release namespace label.
	LabelReleaseNamespace = "module-release.opmodel.dev/namespace"

	// LabelComponent is the OPM infrastructure label that categorizes the type
	// of OPM-managed object (e.g., "inventory"). Distinct from component names
	// set by CUE transformers on application resources.
	LabelComponent = "opmodel.dev/component"

	// LabelModuleUUID is the module identity UUID label for resource discovery.
	LabelModuleUUID = "module.opmodel.dev/uuid"

	// LabelReleaseUUID is the release identity UUID label for resource discovery.
	LabelReleaseUUID = "module-release.opmodel.dev/uuid"
)
