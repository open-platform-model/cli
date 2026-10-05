// Renderable instance fixture for the controller-owned e2e tests.
//
// Unlike the identity-only fixtures elsewhere in the suite, this one actually
// renders: it imports the podinfo module from the registry, so the resulting
// ModuleInstance carries a registry-resolvable spec.module and a real render
// digest — what the controller needs to reconcile it once the tests patch
// spec.owner to "controller" (the thin-editor path refuses a locally-sourced
// module, 0006:D38).
//
// Requires testing.opmodel.dev/modules/cli/podinfo@v0 at the version
// cue.mod/module.cue pins, in the configured registry. It is published to GHCR
// by .github/workflows/publish-fixtures.yml on merge, so the default registry
// mapping resolves it; PR CI resolves it from a job-local registry seeded from
// the tree (hack/fixtures.sh seed).
package controller_owned_instance

import (
	core "opmodel.dev/core@v2"
	podinfo "testing.opmodel.dev/modules/cli/podinfo@v0"
)

core.#ModuleInstance

metadata: {
	name:      "e2e-controller-owned"
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
