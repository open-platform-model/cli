package main

// Minimal core definitions inlined for testing

#NameType: string
#FQNType:  string

#LabelsAnnotationsType: [string]: string

#Component: {
	apiVersion: "core.opm.dev/v0"
	kind:       "Component"
	metadata: {
		name!: #NameType
		labels?: #LabelsAnnotationsType
		annotations?: #LabelsAnnotationsType
	}
	spec: {...}
	#resources?: [string]: _
	#traits?: [string]: _
}

#TransformerContext: close({
	#moduleMetadata:    _
	#componentMetadata: _
	name:               string
	namespace:          string

	moduleLabels: {
		if #moduleMetadata.labels != _|_ {#moduleMetadata.labels}
	}

	componentLabels: {
		"app.kubernetes.io/instance": "\(name)-\(namespace)"
		if #componentMetadata.labels != _|_ {#componentMetadata.labels}
	}

	controllerLabels: {
		"app.kubernetes.io/managed-by": "open-platform-model"
		"app.kubernetes.io/name":       #componentMetadata.name
		"app.kubernetes.io/version":    #moduleMetadata.version
	}

	labels: {[string]: string}
	labels: {
		for k, v in moduleLabels {
			(k): "\(v)"
		}
		for k, v in componentLabels {
			(k): "\(v)"
		}
		for k, v in controllerLabels {
			(k): "\(v)"
		}
		...
	}
})

#Transformer: {
	apiVersion: "opm.dev/core/v0"
	kind:       "Transformer"

	metadata: {
		apiVersion!:  #NameType
		name!:        #NameType
		fqn:          #FQNType & "\(apiVersion)#\(name)"
		description!: string
		labels?:      #LabelsAnnotationsType
		annotations?: #LabelsAnnotationsType
	}

	requiredLabels?:    #LabelsAnnotationsType
	optionalLabels?:    #LabelsAnnotationsType
	requiredResources:  [string]: _
	optionalResources:  [string]: _
	requiredTraits:     [string]: _
	optionalTraits:     [string]: _

	#transform: {
		#component: _
		context:    #TransformerContext
		output:     {...}
	}
}

#TransformerMap: [string]: #Transformer

#Matches: {
	transformer: #Transformer
	component:   #Component

	_reqLabels: *transformer.requiredLabels | {}
	_missingLabels: [
		for k, v in _reqLabels
		if len([for lk, lv in component.metadata.labels if lk == k && (lv & v) != _|_ {true}]) == 0 {
			k
		},
	]

	_reqResources: *transformer.requiredResources | {}
	_missingResources: [
		for k, v in _reqResources
		if len([for rk, rv in component.#resources if rk == k && (rv & v) != _|_ {true}]) == 0 {
			k
		},
	]

	_reqTraits: *transformer.requiredTraits | {}
	_missingTraits: [
		for k, v in _reqTraits
		if component.#traits == _|_ || len([for tk, tv in component.#traits if tk == k && (tv & v) != _|_ {true}]) == 0 {
			k
		},
	]

	result: len(_missingLabels) == 0 && len(_missingResources) == 0 && len(_missingTraits) == 0
}

#Provider: {
	apiVersion: "core.opm.dev/v0"
	kind:       "Provider"
	metadata: {
		name:        string
		description: string
		version:     string
		minVersion:  string
		labels?:     #LabelsAnnotationsType
	}
	transformers: #TransformerMap
	...
}

#Module: {
	apiVersion: "core.opm.dev/v0"
	kind:       "Module"
	metadata: {
		apiVersion!:  string
		name!:        string
		version!:     string
		description?: string
		labels?:      #LabelsAnnotationsType
	}
	#components: [string]: #Component
	config:      _
	values:      _
}

#ModuleRelease: {
	apiVersion: "core.opm.dev/v0"
	kind:       "ModuleRelease"
	metadata: {
		name!:      string
		namespace!: string
		labels?:    #LabelsAnnotationsType
	}
	#module!: #Module
	values:   _

	// Flattened components
	components: {
		for name, comp in #module.#components {
			(name): comp & {
				metadata: {
					if #module.metadata.labels != _|_ {
						labels: {
							for k, v in #module.metadata.labels {
								(k): v
							}
						}
					}
				}
			}
		}
	}
}

#MatchTransformers: {
	provider: #Provider
	module:   #ModuleRelease

	out: {
		for tID, t in provider.transformers {
			let matches = [
				for _, c in module.components
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
}
