// Package controller implements the opm-controller lifecycle surface: the pinned
// controller module and the controller release it deploys, planning and
// performing the install of that module as the CLI-owned instance
// opm-controller, the record-driven uninstall and its safety checks, and the
// running-controller check.
package controller

import (
	"strings"

	"github.com/open-platform-model/cli/internal/inventory"
)

// The controller's fixed names. The running-controller check (CheckReady) locates
// the controller by these names alone, never by its instance record. They are
// the contract between the CLI and the controller module; a module version that
// renames one updates it here.
const (
	// ControllerInstanceName and ControllerNamespace are the coordinates of the
	// ModuleInstance that deploys the controller when it is installed as a module.
	ControllerInstanceName = "opm-controller"
	ControllerNamespace    = "opm-controller-system"

	// ControllerDeploymentName is the controller's Deployment, in
	// ControllerNamespace.
	ControllerDeploymentName = "opm-controller-manager"

	// ControllerAPIGroup is the API group the controller's CRDs serve.
	ControllerAPIGroup = inventory.GroupOpmodel

	// ControllerModulePath is the controller module's path without its @<major>
	// suffix.
	ControllerModulePath = "opmodel.dev/modules/opm_controller"
)

// crdNames are the CRDs every controller release since v1.0.0-alpha.18 serves.
var crdNames = [...]string{
	inventory.CRDNameModuleInstances,
	"modulepackages." + ControllerAPIGroup,
	"platforms." + ControllerAPIGroup,
	"transformerregistrations." + ControllerAPIGroup,
}

// CRDNames returns the CRDs every controller release since v1.0.0-alpha.18
// serves, as a fresh slice the caller may keep or change.
func CRDNames() []string {
	return append([]string(nil), crdNames[:]...)
}

// The signals DeploysController reports, naming which part of the record matched.
const (
	SignalCoordinates  = "coordinates"
	SignalModulePath   = "module path"
	SignalInventoryCRD = "inventory CRD"
)

const crdGroup = "apiextensions.k8s.io"

// DeploysController reports whether rec is an instance that deploys the
// controller, and which signal matched. It reads only the record, never a
// render. Any one of three signals is enough, checked in this order:
//
//   - coordinates: the instance is ControllerInstanceName in ControllerNamespace;
//   - module path: rec.ModulePath, with any @<major> suffix removed, is
//     ControllerModulePath;
//   - inventory CRD: the recorded inventory holds a CustomResourceDefinition
//     whose name, after its first ".", is exactly ControllerAPIGroup.
//
// The controller recognizes the instance that deploys it by the same three
// signals, so the CLI and the controller agree on which instance that is.
func DeploysController(rec *inventory.Record) (signal string, ok bool) {
	if rec == nil {
		return "", false
	}
	if rec.Name == ControllerInstanceName && rec.Namespace == ControllerNamespace {
		return SignalCoordinates, true
	}
	if path, _, _ := strings.Cut(rec.ModulePath, "@"); path == ControllerModulePath {
		return SignalModulePath, true
	}
	for _, e := range rec.Inventory.Entries {
		if e.Group != crdGroup || e.Kind != kindCustomResourceDefinition {
			continue
		}
		if _, group, found := strings.Cut(e.Name, "."); found && group == ControllerAPIGroup {
			return SignalInventoryCRD, true
		}
	}
	return "", false
}
