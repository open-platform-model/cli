package hybrid

// transformer_helpers.cue
// Shared helper definitions for transformers to reduce code duplication.

// #K8sPorts converts component port specifications to Kubernetes port format.
// Usage: let _ports = (#K8sPorts & {_component: #component}).out
#K8sPorts: {
	_component: _
	out: [
		if _component.spec.container.ports != _|_
		for _, port in _component.spec.container.ports {
			name:          port.name
			containerPort: port.targetPort
			protocol:      *port.protocol | "TCP"
		},
	]
}

// #K8sEnv converts component environment variables to Kubernetes env format.
// Usage: let _env = (#K8sEnv & {_component: #component}).out
#K8sEnv: {
	_component: _
	out: [
		if _component.spec.container.env != _|_
		for _, env in _component.spec.container.env {
			name:  env.name
			value: env.value
		},
	]
}
