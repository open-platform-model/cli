// Package minimal is the smallest useful OPM module: one stateless workload,
// defined in a single file. Start here, split into more files as the module
// grows (the standard template shows that shape).
//
// The identity/ package is this module's single source of path and version;
// everything identity-shaped in `metadata` derives from it, so nothing in
// this file needs editing when the module's path or version changes.
package minimal

import (
	"strings"

	m "opmodel.dev/core@v2"
	bp "opmodel.dev/catalogs/opm/blueprints/v1beta1"
	res "opmodel.dev/catalogs/opm/resources/v1beta1"

	id "opmodel.dev/templates/minimal/identity"
)

m.#Module

// Module metadata — modulePath and version are the identity package's values,
// and name is the path's leaf (enhancements 0010 D8, 0011 D12). Edit
// identity/identity.cue, not this block.
metadata: {
	_segments:  strings.Split(strings.SplitN(id.ModulePath, "@", 2)[0], "/")
	name:       _segments[len(_segments)-1]
	modulePath: id.ModulePath
	// Interpolated rather than referenced so the value is concrete before
	// defaults are finalized — the registry loader's shape gate requires a
	// concrete metadata.version, and id.Version is a defaulted disjunction.
	version:     "\(id.Version)"
	description: "A minimal OPM module - one stateless workload"
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

	// Container port
	port: int & >0 & <=65535 | *80
}

// debugValues concretize #config so `opm module vet` and local rendering can
// evaluate the module without an instance.
debugValues: {
	image: {
		repository: "nginx"
		tag:        "1.29"
		digest:     ""
	}
	replicas: 1
	port:     80
}

// #components defines the workload. The StatelessWorkload blueprint stamps
// workload-type=stateless, which selects the Deployment transformer.
#components: {
	app: {
		bp.#StatelessWorkload

		metadata: name: "app"

		spec: statelessWorkload: {
			container: {
				name:  "app"
				image: #config.image
				ports: http: {
					name:       "http"
					targetPort: #config.port
				}
			}
			scaling: count: #config.replicas
			restartPolicy: "Always"
		}
	}
}
