// Fixture for `opm instance tree <file>`: cmdutil.ResolveInstanceArg acquires
// this package through the kernel and reads the instance name and namespace
// from its metadata. Import-free by design: the kernel's shape gate needs the
// kind, a concrete metadata.name and metadata.namespace, and a #module of
// kind Module, none of which needs a registry (core@v2 line).
package insttreetest

kind: "ModuleInstance"

metadata: {
	name:      "tree-test-rel"
	namespace: "opm-tree-test"
}

#module: {
	kind: "Module"
	metadata: {
		name:       "rel-tree"
		modulePath: "opmodel.dev/tests/rel-tree@v1"
		version:    "0.0.1"
	}
}
