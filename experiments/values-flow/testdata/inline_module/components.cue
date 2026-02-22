// inline_module/components.cue — second package file.
// Proves that Approach A loads ALL non-values files correctly.
// When no values*.cue files are present, no filtering occurs but
// the enumeration logic must still handle a multi-file package correctly.
package inlinemodule

#components: {
	web: {
		metadata: name: "web"
		spec: {
			image:    #config.image
			replicas: #config.replicas
		}
	}
	sidecar: {
		metadata: name: "sidecar"
		spec: {
			// Fixed values — not driven by #config
			port:  9090
			image: "prom/prometheus:latest"
		}
	}
}
