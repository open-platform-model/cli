package core

// #BundleRelease: The concrete deployment instance
// Contains: Reference to Bundle, concrete values (closed)
// Users/deployment systems create this to deploy a specific version
#BundleRelease: close({
	apiVersion: "opm.dev/core/v0"
	kind:       "BundleRelease"

	metadata: {
		name!:        string
		labels?:      #LabelsAnnotationsType
		annotations?: #LabelsAnnotationsType
	}

	// Reference to the Bundle to deploy
	#bundle!: #CompiledBundle | #Bundle

	// Concrete values (everything closed/concrete)
	// Must satisfy the value schema from #bundle.spec
	values!: close(#bundle.#spec)

	if #bundle.#status != _|_ {status: #bundle.#status}
	status?: {
		// Deployment lifecycle phase
		phase: "pending" | "deployed" | "failed" | "unknown" | *"pending"

		// Human-readable status message
		message?: string
	}
})

#BundleReleaseMap: [string]: #BundleRelease
