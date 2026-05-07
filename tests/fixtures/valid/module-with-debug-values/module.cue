// Test fixture: minimal stateless module with debugValues so that the
// `opm module build` synthesis path can be exercised by the registry-backed
// tests in cli/pkg/loader, cli/internal/workflow/render, and cli/tests/e2e.
package fixture

import (
	m "opmodel.dev/core/v1alpha1/module@v1"
	schemas "opmodel.dev/opm/v1alpha1/schemas@v1"
)

m.#Module

metadata: {
	modulePath:       "example.com/fixtures"
	name:             "module-with-debug-values"
	version:          "0.1.0"
	description:      "Minimal synthable module used as a test fixture for opm module build."
	defaultNamespace: "default"
}

#config: {
	image: schemas.#Image & {
		repository: string | *"nginx"
		tag:        string | *"1.27"
		digest:     string | *""
	}
	port:        int & >0 & <=65535 | *80
	replicas:    uint & >0 | *1
	serviceType: "ClusterIP" | "NodePort" | "LoadBalancer" | *"ClusterIP"
	resources?:  schemas.#ResourceRequirementsSchema
}

debugValues: {
	port:        80
	replicas:    1
	serviceType: "ClusterIP"
	resources: {
		requests: {cpu: "10m", memory: "32Mi"}
		limits: {cpu: "100m", memory: "64Mi"}
	}
}
