@if(test)

package extension

// =============================================================================
// CRDs Resource Tests
// =============================================================================

// Test: CRDsResource definition structure
_testCRDsResourceDef: #CRDsResource & {
	metadata: {
		modulePath: "opmodel.dev/resources/extension"
		version:    "v1"
		name:       "crds"
		fqn:        "opmodel.dev/resources/extension/crds@v1"
	}
}

// Test: Single CRD component
_testSingleCRDComponent: #CRDs & {
	metadata: name: "grafana-crds"
	spec: crds: {
		"grafanas.grafana.integreatly.org": {
			group: "grafana.integreatly.org"
			names: {
				kind:   "Grafana"
				plural: "grafanas"
			}
			scope: "Namespaced"
			versions: [{
				name:    "v1beta1"
				served:  true
				storage: true
			}]
		}
	}
}

// Test: Multiple CRDs in a single component
_testMultipleCRDsComponent: #CRDs & {
	metadata: name: "cert-manager-crds"
	spec: crds: {
		"certificates.cert-manager.io": {
			group: "cert-manager.io"
			names: {
				kind:     "Certificate"
				plural:   "certificates"
				singular: "certificate"
				shortNames: ["cert", "certs"]
				categories: ["cert-manager"]
			}
			scope: "Namespaced"
			versions: [{
				name:    "v1"
				served:  true
				storage: true
				schema: openAPIV3Schema: {
					type: "object"
					properties: spec: type: "object"
				}
				subresources: status: {}
			}]
		}
		"clusterissuers.cert-manager.io": {
			group: "cert-manager.io"
			names: {
				kind:   "ClusterIssuer"
				plural: "clusterissuers"
			}
			scope: "Cluster"
			versions: [{
				name:    "v1"
				served:  true
				storage: true
			}]
		}
	}
}
