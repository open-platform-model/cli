// Package standard is the default OPM module starting point, with separated
// concerns:
//   - module.cue:     metadata and the #config contract (with defaults)
//   - components.cue: component definitions
//
// The identity/ package is this module's single source of path and version;
// everything identity-shaped in `metadata` derives from it, so nothing in
// this file needs editing when the module's path or version changes.
package standard

import (
	"strings"

	m "opmodel.dev/core@v2"
	res "opmodel.dev/catalogs/opm/resources/v1beta1"

	id "opmodel.dev/templates/standard/identity"
)

m.#Module

// Module metadata — modulePath and version are the identity package's values,
// and name is the path's leaf (enhancements 0010 D8, 0011 D12). Edit
// identity/identity.cue, not this block.
metadata: {
	_segments: strings.Split(strings.SplitN(id.ModulePath, "@", 2)[0], "/")
	name:        _segments[len(_segments)-1]
	modulePath:  id.ModulePath
	version:     id.Version
	description: "A standard OPM module - one exposed stateless workload"
}

// #config is the module's configuration contract. Instances override these
// fields at deploy time; every field carries a working default.
#config: {
	// Container image
	image: res.#Image & {
		repository: string | *"nginx"
		tag:        string | *"1.29"
		digest:     string | *""
	}

	// Replica count
	replicas: int & >=1 | *1

	// Container/Service port
	port: int & >0 & <=65535 | *80

	// Kubernetes Service type
	serviceType: "ClusterIP" | "NodePort" | "LoadBalancer" | *"ClusterIP"
}

// debugValues concretize #config so `opm module vet` and local rendering can
// evaluate the module without an instance.
debugValues: {
	image: {
		repository: "nginx"
		tag:        "1.29"
		digest:     ""
	}
	replicas:    1
	port:        80
	serviceType: "ClusterIP"
}
