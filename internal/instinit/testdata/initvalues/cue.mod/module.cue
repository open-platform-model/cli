module: "test.example/modules/initvalues@v0"
language: {
	version: "v0.17.0"
}
source: {
	kind: "self"
}
deps: {
	"opmodel.dev/catalogs/opm@v4": {
		v:       "v4.4.1"
		default: true
	}
	"opmodel.dev/core@v2": {
		v: "v2.0.0-alpha.11"
	}
}
