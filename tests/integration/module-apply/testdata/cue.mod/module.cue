// Declared path equals metadata.modulePath verbatim (core v2 identity):
// kernel synthesis imports the module by that canonical path, and the import
// resolves locally only when it matches the declared module path.
module: "example.com/modules/module_apply_itest@v0"
language: {
	version: "v0.17.0"
}
source: {
	kind: "self"
}
deps: {
	"opmodel.dev/catalogs/opm@v2": {
		v: "v2.0.0-alpha.3"
	}
	"opmodel.dev/core@v2": {
		v: "v2.0.0-alpha.4"
	}
}
