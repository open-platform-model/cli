package core

// #Scope: Defines cross-cutting concerns and shared contexts
// that span multiple components within a system.
// Scopes encapsulate policies and configurations that apply
// to a group of components, enabling consistent governance
// and operational behavior across those components.
#Scope: close({
	apiVersion: "opm.dev/core/v0"
	kind:       "Scope"

	metadata: {
		name!: string

		labels?:      #LabelsAnnotationsType
		annotations?: #LabelsAnnotationsType
	}

	// Policies applied by this scope
	// Only policies with target "scope" can be applied here
	#policies: [PolicyFQN=string]: #Policy & {
		metadata: {
			name: string | *PolicyFQN
			// Validation: target must be "scope"
			target: "scope"
		}
	}

	// Policy applicability
	// Which components this scope applies to
	appliesTo: {
		// Component label selectors
		componentLabels?: [string]: #LabelsAnnotationsType

		// Specific component names
		// Either a list components or pointing to #allComponentsList in the module
		components: [...#Component]
	}

	_allFields: {
		if #policies != _|_ {
			for _, policy in #policies {
				if policy.#spec != _|_ {
					for k, v in policy.#spec {
						(k): v
					}
				}
			}
		}
	}

	// Fields exposed by this scope
	// Automatically turned into a spec
	// Must be made concrete by the user
	spec: close(_allFields)
})

#ScopeMap: [string]: #Scope
