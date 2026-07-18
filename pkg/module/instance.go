package module

// InstanceMetadata contains instance-level identity information.
// Used for K8s inventory tracking, resource labeling, and CLI output.
//
// Was: ReleaseMetadata (enhancement 0002 D8 hard-rename).
type InstanceMetadata struct {
	// Name is the instance name (from --name or module.metadata.name).
	Name string `json:"name"`

	// Namespace is the target namespace.
	Namespace string `json:"namespace"`

	// UUID is the instance identity UUID.
	// Computed by CUE as SHA1(OPMNamespace, moduleUUID:name:namespace).
	UUID string `json:"uuid"`

	// Labels are the merged instance labels (module labels + standard opm labels).
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations are the merged instance annotations.
	Annotations map[string]string `json:"annotations,omitempty"`
}
