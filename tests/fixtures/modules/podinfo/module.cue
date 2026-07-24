// Vendored from opm-operator/test/fixtures/modules/podinfo (module.cue) for the
// render-parity integration program, so this repo's tests carry no sibling
// checkout dependency. Byte-for-byte drift from the operator's copy is
// acceptable by design: render-parity's correctness comes from comparing the
// CLI and kernel render paths over the SAME fixture, not from matching the
// operator's copy (enhancement 0006 slice C2, design LD3).
//
// podinfo — stateless web example module (opmodel.dev/core@v1). Renders a
// Deployment + Service via the catalog's deployment- and service-transformers,
// with an HTTP livenessProbe (/healthz) and readinessProbe (/readyz) on the
// podinfo HTTP port (9898).
package podinfo

import (
	m "opmodel.dev/core@v1"
	res "opmodel.dev/catalogs/opm/resources"
)

m.#Module

metadata: {
	modulePath:  "opmodel.dev/modules/test"
	name:        "podinfo"
	version:     "0.1.3"
	description: "Stateless web example — Deployment + Service with HTTP liveness/readiness probes"
}

#config: {
	// Container image. Defaults to upstream podinfo; override repository/tag/digest
	// via the ModuleRelease values to pin a specific build.
	image: res.#Image & {repository: string | *"ghcr.io/stefanprodan/podinfo", tag: string | *"6.7.1", digest: string | *""}

	// Number of Deployment replicas.
	replicas: int | *1
}

debugValues: {
	image: {repository: "ghcr.io/stefanprodan/podinfo", tag: "6.7.1", digest: ""}
	replicas: 1
}
