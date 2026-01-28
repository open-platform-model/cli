package main

// Define a simple module
module: #Module & {
	metadata: {
		apiVersion:  "opm.dev@v0"
		name:        "simple-app"
		version:     "0.1.0"
		description: "Simple test application"
	}

	#components: {
		web: #Component & {
			metadata: {
				name: "web"
				labels: {
					"core.opm.dev/workload-type": "stateless"
				}
			}
			spec: {}
		}
		api: #Component & {
			metadata: {
				name: "api"
				labels: {
					"core.opm.dev/workload-type": "stateless"
				}
			}
			spec: {}
		}
		worker: #Component & {
			metadata: {
				name: "worker"
				labels: {
					"core.opm.dev/workload-type": "stateless"
				}
			}
			spec: {}
		}
		database: #Component & {
			metadata: {
				name: "database"
				labels: {
					"core.opm.dev/workload-type": "stateful"
				}
			}
			spec: {}
		}
	}

	config: {}
	values: {}
}

// Create a module release
moduleRelease: #ModuleRelease & {
	metadata: {
		name:      "simple-app-release"
		namespace: "default"
	}
	#module: module
	values:  {}
}

// Compute matching plan using the provider (inline computation)
matchingPlan: {
	for tID, t in provider.transformers {
		let matches = [
			for _, c in moduleRelease.components
			if (#Matches & {transformer: t, component: c}).result {
				c
			},
		]

		if len(matches) > 0 {
			(tID): {
				transformer: t
				components:  matches
			}
		}
	}
}
