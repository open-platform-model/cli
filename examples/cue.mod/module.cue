module: "opmodel.dev/examples@v1"
language: {
	version: "v0.17.0"
}
source: {
	kind: "self"
}
deps: {
	"opmodel.dev/catalogs/opm@v4": {
		v: "v4.4.0"
	}
	"opmodel.dev/core@v2": {
		v: "v2.0.0-alpha.10"
	}
	"testing.opmodel.dev/modules/cli/podinfo@v0": {
		v: "v0.1.10"
	}
}
