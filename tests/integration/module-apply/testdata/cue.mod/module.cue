// Declared path follows the nameSnakeCase publishing convention
// (metadata.modulePath + "/" + nameSnakeCase): kernel synthesis imports the
// module by that canonical path, and the import resolves locally only when
// it matches the declared module path.
module: "example.com/modules/module_apply_itest@v0"
language: {
	version: "v0.17.0"
}
source: {
	kind: "self"
}
deps: {
	"opmodel.dev/catalogs/opm@v1": {
		v: "v1.0.0-alpha.2"
	}
	"opmodel.dev/core@v1": {
		v: "v1.0.0-alpha.1"
	}
}
