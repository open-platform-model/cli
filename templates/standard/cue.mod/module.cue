module: "opmodel.dev/templates/standard@v1"
language: {
	version: "v0.17.0"
}
source: {
	kind: "self"
}
deps: {
	"cue.dev/x/k8s.io@v0": {
		v: "v0.7.0"
	}
	"opmodel.dev/catalogs/opm@v2": {
		v: "v2.0.0-alpha.3"
	}
	"opmodel.dev/core@v2": {
		v: "v2.0.0-alpha.5"
	}
}
