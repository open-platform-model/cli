// Package operator implements the opm-operator lifecycle surface: the pinned
// operator module and the operator release it deploys, planning and
// performing the install of that module as the CLI-owned instance
// opm-operator, the record-driven uninstall and its safety checks, and the
// running-operator check.
package operator

import (
	"strings"

	"github.com/open-platform-model/cli/internal/inventory"
)

// The operator's fixed names. The running-operator check (CheckReady) locates
// the operator by these names alone, so it finds an operator applied from a
// release manifest, by kubectl or by 'opm operator install', and an operator
// installed as the operator module, whose instance keeps every one of them.
// They are the contract that check relies on once the CLI no longer embeds
// the operator manifest. A module version that renames one updates it here.
const (
	// OperatorInstanceName and OperatorNamespace are the coordinates of the
	// ModuleInstance that deploys the operator when it is installed as a module.
	OperatorInstanceName = "opm-operator"
	OperatorNamespace    = "opm-operator-system"

	// ControllerDeploymentName is the operator's controller Deployment, in
	// OperatorNamespace.
	ControllerDeploymentName = "opm-operator-controller-manager"

	// OperatorAPIGroup is the API group the operator's CRDs serve.
	OperatorAPIGroup = inventory.GroupOpmodel

	// OperatorModulePath is the operator module's path without its @<major>
	// suffix.
	OperatorModulePath = "opmodel.dev/modules/opm_operator"
)

// crdNames are the CRDs every operator release since v1.0.0-alpha.18 serves.
var crdNames = [...]string{
	inventory.CRDNameModuleInstances,
	"modulepackages." + OperatorAPIGroup,
	"platforms." + OperatorAPIGroup,
	"transformerregistrations." + OperatorAPIGroup,
}

// CRDNames returns the CRDs every operator release since v1.0.0-alpha.18
// serves, as a fresh slice the caller may keep or change.
func CRDNames() []string {
	return append([]string(nil), crdNames[:]...)
}

// The signals DeploysOperator reports, naming which part of the record matched.
const (
	SignalCoordinates  = "coordinates"
	SignalModulePath   = "module path"
	SignalInventoryCRD = "inventory CRD"
)

const crdGroup = "apiextensions.k8s.io"

// DeploysOperator reports whether rec is an instance that deploys the
// operator, and which signal matched. It reads only the record, never a
// render. Any one of three signals is enough, checked in this order:
//
//   - coordinates: the instance is OperatorInstanceName in OperatorNamespace;
//   - module path: rec.ModulePath, with any @<major> suffix removed, is
//     OperatorModulePath;
//   - inventory CRD: the recorded inventory holds a CustomResourceDefinition
//     whose name, after its first ".", is exactly OperatorAPIGroup.
//
// The operator recognizes the instance that deploys it by the same three
// signals, so the CLI and the operator agree on which instance that is.
func DeploysOperator(rec *inventory.Record) (signal string, ok bool) {
	if rec == nil {
		return "", false
	}
	if rec.Name == OperatorInstanceName && rec.Namespace == OperatorNamespace {
		return SignalCoordinates, true
	}
	if path, _, _ := strings.Cut(rec.ModulePath, "@"); path == OperatorModulePath {
		return SignalModulePath, true
	}
	for _, e := range rec.Inventory.Entries {
		if e.Group != crdGroup || e.Kind != kindCustomResourceDefinition {
			continue
		}
		if _, group, found := strings.Cut(e.Name, "."); found && group == OperatorAPIGroup {
			return SignalInventoryCRD, true
		}
	}
	return "", false
}
