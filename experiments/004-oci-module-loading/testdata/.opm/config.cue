package config

import (
	k8s "opm.dev/providers/kubernetes@v0"
)

config: {
	registry: "localhost:5001"
	providers: {
		kubernetes: k8s.#Provider
	}
}
