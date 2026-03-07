package policy

import (
	t "opmodel.dev/core/types@v1"
	prim "opmodel.dev/core/primitives@v1"
	component "opmodel.dev/core/component@v1"
)

// #Policy: Groups PolicyRules and targets them to a set of
// components via label matching or explicit references.
// Policies enable cross-cutting governance without coupling
// rules to individual components.
#Policy: {
	apiVersion: "opmodel.dev/core/v0"
	kind:       "Policy"

	metadata: {
		name!: t.#NameType

		labels?:      t.#LabelsAnnotationsType
		annotations?: t.#LabelsAnnotationsType
	}

	// PolicyRules grouped by this policy
	#rules: [RuleFQN=string]: prim.#PolicyRule & {
		metadata: {
			name: string | *RuleFQN
		}
	}

	// Which components this policy applies to
	// At least one of matchLabels or components must be specified
	appliesTo: {
		// Label-based matching — select components whose labels are a superset
		matchLabels?: t.#LabelsAnnotationsType

		// Explicit component references
		components?: [...component.#Component]
	}

	_allFields: {
		if #rules != _|_ {
			for _, rule in #rules {
				if rule.#spec != _|_ {
					for k, v in rule.#spec {
						(k): v
					}
				}
			}
		}
	}

	// Fields exposed by this policy
	// Automatically turned into a spec
	// Must be made concrete by the user
	spec: close(_allFields)
}

#PolicyMap: [string]: #Policy
