// Package core defines shared domain types used across OPM packages.
// It depends only on stdlib and k8s.io/apimachinery — no CUE, no internal packages.
package core

import "github.com/google/uuid"

// OPMNamespace is the UUID v5 namespace for computing deterministic release identities.
// Derived as: uuid.NewSHA1(uuid.NameSpaceDNS, []byte("opmodel.dev")).String()
// Matches the value in catalog/v0/core/common.cue.
const OPMNamespace = "11bc6112-a6e8-4021-bec9-b3ad246f9466"

// ComputeReleaseUUID computes a deterministic UUID v5 for a release using the OPM namespace.
// The input string is fqn:name:namespace. Same inputs always produce the same UUID.
func ComputeReleaseUUID(fqn, name, namespace string) string {
	ns := uuid.MustParse(OPMNamespace)
	return uuid.NewSHA1(ns, []byte(fqn+":"+name+":"+namespace)).String()
}

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
