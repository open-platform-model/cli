package main

import (
	"opmodel.dev/core@v1"
	schemas "opmodel.dev/schemas@v1"
	resources_workload "opmodel.dev/resources/workload@v1"
)

core.#Module

metadata: {
	modulePath:       "example.com/modules"
	name:             "secrets-module"
	version:          "0.1.0"
	defaultNamespace: "default"
}

#config: {
	image: schemas.#Image

	db: {
		password: schemas.#Secret & {
			$secretName: "db-creds"
			$dataKey:    "password"
		}
		host: schemas.#Secret & {
			$secretName: "db-creds"
			$dataKey:    "host"
		}
	}

	apiKey: schemas.#Secret & {
		$secretName: "api-keys"
		$dataKey:    "api-key"
	}
}

#components: {
	web: {
		resources_workload.#Container

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
