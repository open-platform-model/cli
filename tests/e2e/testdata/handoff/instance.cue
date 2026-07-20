// Renderable instance fixture for the handoff/adoption e2e tests.
//
// Unlike the identity-only fixtures elsewhere in the suite, this one actually
// renders: it imports the podinfo module from the registry, so the resulting
// ModuleInstance carries a registry-resolvable spec.module and a real render
// digest. Both are preconditions for `opm instance handoff` — a locally-sourced
// module is refused outright (0006 D38), and the digest gate needs something to
// compare against.
//
// Requires opmodel.dev/modules/test/podinfo@v0 v0.1.3 in the configured
// registry (the local kind registry publishes it).
package handoff_instance

import (
	core "opmodel.dev/core@v1"
	podinfo "opmodel.dev/modules/test/podinfo@v0"
)

core.#ModuleInstance

metadata: {
	name:      "e2e-handoff"
	namespace: "default"
}

#module: podinfo

// Mirrors the module's debugValues. Stated explicitly rather than relying on
// the schema defaults so the values round-trip test has a field it can change
// and observe on the CR.
values: {
	image: {
		repository: "ghcr.io/stefanprodan/podinfo"
		tag:        "6.7.1"
		digest:     ""
	}
	replicas: 1
}
