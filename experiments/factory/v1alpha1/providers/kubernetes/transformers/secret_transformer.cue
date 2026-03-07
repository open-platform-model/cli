package transformers

import (
	transformer "opmodel.dev/core/transformer@v1"
	schemas "opmodel.dev/schemas@v1"
	config_resources "opmodel.dev/resources/config@v1"
	k8scorev1 "opmodel.dev/schemas/kubernetes/core/v1@v1"
)

// SecretTransformer converts Secrets resources to Kubernetes Secrets and ExternalSecrets.
//
// Variant dispatch per data entry:
//   #SecretLiteral -> include in K8s Secret stringData
//   #SecretK8sRef  -> skip (resource already exists in cluster)
//   #SecretEsoRef  -> emit ExternalSecret CR
//
// Mixed variants within a single secret group are supported: literal entries
// create a K8s Secret, ESO entries create ExternalSecret CRs, K8s refs are skipped.
#SecretTransformer: transformer.#Transformer & {
	metadata: {
		modulePath:  "opmodel.dev/providers/kubernetes/transformers"
		version:     "v1"
		name:        "secret-transformer"
		description: "Converts Secrets resources to Kubernetes Secrets and ExternalSecrets"

		labels: {
			"core.opmodel.dev/resource-category": "config"
			"core.opmodel.dev/resource-type":     "secret"
		}
	}

	requiredLabels: {}

	// Required resources - Secrets MUST be present
	requiredResources: {
		"opmodel.dev/resources/config/secrets@v1": config_resources.#SecretsResource
	}

	optionalResources: {}
	requiredTraits: {}
	optionalTraits: {}

	#transform: {
		#component: _
		#context:   transformer.#TransformerContext

		_secrets: #component.spec.secrets

		output: {
			for _secretName, secret in _secrets {
				// Compute the deterministic K8s resource name
				let _k8sName = (schemas.#SecretImmutableName & {
					baseName:  secret.name
					data:      secret.data
					immutable: secret.immutable
				}).out

				// Collect #SecretLiteral entries for K8s Secret stringData
				let _literals = {
					for _dk, _entry in secret.data
					if _entry.value != _|_
					if _entry.secretName == _|_
					if _entry.externalPath == _|_ {
						(_dk): _entry.value
					}
				}

				// Emit K8s Secret if there are any literal entries
				if len(_literals) > 0 {
					"\(_k8sName)": k8scorev1.#Secret & {
						apiVersion: "v1"
						kind:       "Secret"
						metadata: {
							name:      _k8sName
						namespace: #context.#moduleReleaseMetadata.namespace
						labels:    #context.labels
						if len(#context.componentAnnotations) > 0 {
							annotations: #context.componentAnnotations
						}
					}
					type: secret.type
					if secret.immutable == true {
						immutable: true
					}
					stringData: _literals
				}
			}

			// Emit ExternalSecret CRs for #SecretEsoRef entries
			for _dk, _entry in secret.data
			if _entry.externalPath != _|_ {
				"ExternalSecret/\(_k8sName)": {
					apiVersion: "external-secrets.io/v1beta1"
					kind:       "ExternalSecret"
					metadata: {
						name:      _k8sName
						namespace: #context.#moduleReleaseMetadata.namespace
							labels:    #context.labels
							if len(#context.componentAnnotations) > 0 {
								annotations: #context.componentAnnotations
							}
						}
						spec: {
							target: name: _k8sName
							data: [{
								secretKey: _dk
								remoteRef: {
									key:      _entry.externalPath
									property: _entry.remoteKey
								}
							}]
						}
					}
				}

				// #SecretK8sRef entries: nothing emitted (resource pre-exists)
			}
		}
	}
}
