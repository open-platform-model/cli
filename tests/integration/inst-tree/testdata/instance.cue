// Identity-only fixture: `opm instance tree <file>` resolves the instance
// name/namespace from this file (cmdutil.ResolveInstanceArg). Data-only —
// no schema import needed for identity extraction (core@v1 line).
package insttreetest

kind: "ModuleInstance"

metadata: {
	name:      "tree-test-rel"
	namespace: "opm-tree-test"
}
