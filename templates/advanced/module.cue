// Package advanced is the showcase OPM module: multiple components (a web
// frontend, an API, a background worker, and a stateful cache), trait
// attachments beyond the blueprints (Expose, PodMetadata, an optional
// HTTPRoute), and #config values plumbed through every component.
//
// Files:
//   - module.cue:     metadata and the #config contract (with defaults)
//   - components.cue: component definitions
//
// The identity/ package is this module's single source of path and version;
// everything identity-shaped in `metadata` derives from it, so nothing in
// this file needs editing when the module's path or version changes.
package advanced

import (
	"strings"

	m "opmodel.dev/core@v2"
	res "opmodel.dev/catalogs/opm/resources/v1beta1"

	id "opmodel.dev/templates/advanced/identity"
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
	description: "An advanced OPM module - web, api, worker, and a stateful cache"
}

// #config is the module's configuration contract. Instances override these
// fields at deploy time; every field carries a working default.
#config: {
	// Web frontend
	web: {
		image: res.#Image & {
			repository: string | *"nginx"
			tag:        string | *"1.29"
			digest:     string | *""
		}
		replicas: int & >=1 | *2
		port:     int & >0 & <=65535 | *80
	}

	// API backend
	api: {
		image: res.#Image & {
			repository: string | *"caddy"
			tag:        string | *"2.10"
			digest:     string | *""
		}
		replicas: int & >=1 | *2
		port:     int & >0 & <=65535 | *8080
		logLevel: "debug" | "info" | "warn" | "error" | *"info"
	}

	// Background worker
	worker: {
		image: res.#Image & {
			repository: string | *"busybox"
			tag:        string | *"1.37"
			digest:     string | *""
		}
		// Seconds between work cycles
		interval: int & >=1 | *60
	}

	// Stateful cache
	cache: {
		image: res.#Image & {
			repository: string | *"valkey/valkey"
			tag:        string | *"8"
			digest:     string | *""
		}
		port: int & >0 & <=65535 | *6379
		storage: {
			size:         string | *"1Gi"
			storageClass: string | *"standard"
		}
	}

	// Optional Gateway API HTTPRoute for the web frontend. When set, an
	// HTTPRoute resource is rendered pointing at the web Service.
	httpRoute?: {
		hostnames: [...string]
		gatewayRef?: {
			name:      string
			namespace: string
		}
	}
}

// debugValues concretize #config so `opm module vet` and local rendering can
// evaluate the module without an instance.
debugValues: {
	web: {
		image: {
			repository: "nginx"
			tag:        "1.29"
			digest:     ""
		}
		replicas: 2
		port:     80
	}
	api: {
		image: {
			repository: "caddy"
			tag:        "2.10"
			digest:     ""
		}
		replicas: 2
		port:     8080
		logLevel: "info"
	}
	worker: {
		image: {
			repository: "busybox"
			tag:        "1.37"
			digest:     ""
		}
		interval: 60
	}
	cache: {
		image: {
			repository: "valkey/valkey"
			tag:        "8"
			digest:     ""
		}
		port: 6379
		storage: {
			size:         "1Gi"
			storageClass: "standard"
		}
	}
}
