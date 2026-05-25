// Package itest is the module fixture for the `opm module apply` integration
// test. It defines two stateless components, `web` (always rendered) and `api`
// (gated by #config.api.enabled), so the test can demonstrate that toggling
// `api.enabled` from true to false causes the api resources to be pruned on
// re-apply.
package itest

import (
	m "opmodel.dev/core/v1alpha1/module@v1"
	"opmodel.dev/opm/v1alpha1/schemas@v1"
)

m.#Module

#workloadComponent: "web"

metadata: {
	modulePath:       "example.com/modules"
	name:             "module-apply-itest"
	version:          "0.1.0"
	description:      "Integration-test fixture for opm module apply"
	defaultNamespace: "default"
}

#config: {
	web: {
		image: schemas.#Image & {
			repository: string | *"nginx"
			tag:        string | *"latest"
			digest:     string | *""
		}
		scaling: int & >=1 | *1
		port:    int & >0 & <=65535 | *8080
	}
	api: {
		enabled: bool | *true
		image: schemas.#Image & {
			repository: string | *"nginx"
			tag:        string | *"latest"
			digest:     string | *""
		}
		scaling: int & >=1 | *1
		port:    int & >0 & <=65535 | *3000
	}
}

debugValues: {
	web: {
		image: {repository: "nginx", tag: "latest", digest: ""}
		scaling: 1
		port:    8080
	}
	api: {
		enabled: true
		image: {repository: "nginx", tag: "latest", digest: ""}
		scaling: 1
		port:    3000
	}
}
