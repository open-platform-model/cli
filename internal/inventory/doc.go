// Package inventory is the CLI's inventory record (the ModuleInstance CR and
// its status.inventory block), the explicit mapping of that block to the CRD's
// wire shape, and the cluster operations over it. The inventory entry, its
// identity, the stale set and both digests are the library's
// opm/k8s/inventory; this package keeps no copy of them (0012:D1, 0012:D7).
package inventory
