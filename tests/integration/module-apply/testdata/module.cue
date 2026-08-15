// Integration-test fixture for `opm module apply` (core@v2 line). The api
// component is gated on #config.api.enabled so the test can verify the prune
// path by re-applying with values_api_off.cue.
package module_apply_itest

import (
	m "opmodel.dev/core@v2"
	res "opmodel.dev/catalogs/opm/resources/v1beta1"
)

m.#Module

metadata: {
	name:             "module_apply_itest"
	modulePath:       "example.com/modules/module_apply_itest@v0"
	version:          "0.1.0"
	description: "Integration-test fixture for opm module apply"
}

#config: {
	web: {
		image:   res.#Image & {repository: string | *"nginx", tag: string | *"latest", digest: string | *""}
		scaling: int & >=1 | *1
	}
	api: {
		enabled: bool | *true
		image:   res.#Image & {repository: string | *"nginx", tag: string | *"latest", digest: string | *""}
		scaling: int & >=1 | *1
	}
}

debugValues: {
	web: {
		image: {repository: "nginx", tag: "latest", digest: ""}
		scaling: 1
	}
	api: {
		enabled: true
		image: {repository: "nginx", tag: "latest", digest: ""}
		scaling: 1
	}
}
