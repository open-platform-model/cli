// Vet fixture on the current schema line (opmodel.dev/core@v1) exercising secret
// discovery: #config carries res.#Secret contract fields (the $opm-tagged
// contract the secret transformer discovers) plus a container component wired to
// the image. Ported from the retired v1alpha1 catalog line; the old standalone
// values.cue is folded into debugValues (a stray top-level `values:` field would
// break the closed #Module).
package secretsmodule

import (
	m "opmodel.dev/core@v1"
	res "opmodel.dev/catalogs/opm/resources"
)

m.#Module

metadata: {
	modulePath: "example.com/modules"
	name:       "secrets-module"
	version:    "0.1.0"
}

#config: {
	image: res.#Image

	db: {
		password: res.#Secret & {
			$secretName: "db-creds"
			$dataKey:    "password"
		}
		host: res.#Secret & {
			$secretName: "db-creds"
			$dataKey:    "host"
		}
	}

	apiKey: res.#Secret & {
		$secretName: "api-keys"
		$dataKey:    "api-key"
	}
}

#components: {
	web: {
		res.#Container

		metadata: {
			name: "web"
			labels: "core.opmodel.dev/workload-type": "stateless"
		}

		spec: container: {
			name:  "web"
			image: #config.image
		}
	}
}

debugValues: {
	image: {
		repository: "nginx"
		tag:        "1.28"
		digest:     ""
	}
	db: {
		password: value: "super-secret"
		host: value:     "db.example.com"
	}
	apiKey: value: "my-api-key-123"
}
