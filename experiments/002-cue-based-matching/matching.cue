package experiment

import (
	core "test.com/experiment/pkg/core"
)

// #Matches checks if a component satisfies a transformer's requirements
#Matches: {
	transformer: core.#Transformer
	component:   core.#Component

	// 1. Check Required Labels
	// Logic: All labels in transformer.requiredLabels must exist in component.metadata.labels with same value
	_reqLabels: *transformer.requiredLabels | {}
	_missingLabels: [
		for k, v in _reqLabels
		if len([for lk, lv in component.metadata.labels if lk == k && (lv & v) != _|_ {true}]) == 0 {
			k
		},
	]

	// 2. Check Required Resources
	// Logic: All keys in transformer.requiredResources must exist in component.#resources
	_reqResources: *transformer.requiredResources | {}
	_missingResources: [
		for k, v in _reqResources
		if len([for rk, rv in component.#resources if rk == k && (rv & v) != _|_ {true}]) == 0 {
			k
		},
	]

	// 3. Check Required Traits
	// Logic: All keys in transformer.requiredTraits must exist in component.#traits
	_reqTraits: *transformer.requiredTraits | {}
	_missingTraits: [
		for k, v in _reqTraits
		if component.#traits == _|_ || len([for tk, tv in component.#traits if tk == k && (tv & v) != _|_ {true}]) == 0 {
			k
		},
	]

	// Result: true if no requirements are missing
	result: len(_missingLabels) == 0 && len(_missingResources) == 0 && len(_missingTraits) == 0
}
