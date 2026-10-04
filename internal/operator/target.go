package operator

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/mod/semver"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/output"
)

// managerContainer is the controller Deployment's container that runs the
// operator.
const managerContainer = "manager"

// Target is the operator module version an install deploys and the
// operator release that version states it deploys.
type Target struct {
	// ModuleVersion is the module version, "v"-prefixed ("v0.1.0").
	ModuleVersion string
	// OperatorVersion is the operator release, "v"-prefixed.
	OperatorVersion string
	// Default reports that ModuleVersion is the cli's pin.
	Default bool
}

// Module returns the module version bare, as a user writes it ("0.1.0").
func (t Target) Module() string { return strings.TrimPrefix(t.ModuleVersion, "v") }

// TargetError is a refusal of the target module version (exit 2): it names
// the module version and the rule it failed.
type TargetError struct {
	ModuleVersion string
	Rule          string
}

func (e *TargetError) Error() string {
	return fmt.Sprintf("refusing %s %s: %s", OperatorModulePath, strings.TrimPrefix(e.ModuleVersion, "v"), e.Rule)
}

// RenderedOperatorImage returns the image tag container "manager" of the
// controller Deployment runs in a render: the part after the last ":" of the
// image reference's last path element, without any "@<digest>".
func RenderedOperatorImage(objs []*unstructured.Unstructured) (string, error) {
	for _, obj := range objs {
		if obj.GetKind() != kindDeployment || obj.GetName() != ControllerDeploymentName || obj.GetNamespace() != OperatorNamespace {
			continue
		}
		containers, _, _ := unstructured.NestedSlice(obj.Object, "spec", "template", "spec", "containers") //nolint:errcheck // a wrong type reads as no containers
		for _, c := range containers {
			cm, ok := c.(map[string]any)
			if !ok || cm["name"] != managerContainer {
				continue
			}
			image, isString := cm["image"].(string)
			if !isString {
				image = ""
			}
			if tag := imageTag(image); tag != "" {
				return tag, nil
			}
			return "", fmt.Errorf("container %q of Deployment %s/%s runs %q, which carries no tag",
				managerContainer, OperatorNamespace, ControllerDeploymentName, image)
		}
		return "", fmt.Errorf("the Deployment %s/%s has no container %q", OperatorNamespace, ControllerDeploymentName, managerContainer)
	}
	return "", fmt.Errorf("the render has no Deployment %s/%s", OperatorNamespace, ControllerDeploymentName)
}

// imageTag returns the tag of an image reference, "" when it has none.
func imageTag(image string) string {
	ref, _, _ := strings.Cut(image, "@")
	name := ref[strings.LastIndex(ref, "/")+1:]
	_, tag, ok := strings.Cut(name, ":")
	if !ok {
		return ""
	}
	return tag
}

// CheckTarget applies install's target rules to a rendered module version
// before anything is written (0021:D9:R4):
//
//   - V1: the operator's MAJOR.MINOR is not above the cli's; a cli whose own
//     version is not a released SemVer skips the rule with a warning;
//   - the rendered controller image is tagged with the operator version the
//     module states;
//   - V5: the rendered ModuleInstance CRD carries the cli's field floor.
//
// Every refusal is a *TargetError.
func CheckTarget(t Target, objs []*unstructured.Unstructured, cliVersion string) error {
	if err := checkOperatorNotNewer(t, cliVersion); err != nil {
		return err
	}

	tag, err := RenderedOperatorImage(objs)
	if err != nil {
		return &TargetError{ModuleVersion: t.ModuleVersion, Rule: err.Error()}
	}
	if tag != t.OperatorVersion {
		return &TargetError{ModuleVersion: t.ModuleVersion, Rule: fmt.Sprintf(
			"it states operator %s but its controller image is tagged %s", t.OperatorVersion, tag)}
	}

	crd := renderedCRD(objs, inventory.CRDNameModuleInstances)
	if crd == nil {
		return &TargetError{ModuleVersion: t.ModuleVersion, Rule: "its render has no CustomResourceDefinition " + inventory.CRDNameModuleInstances}
	}
	if err := inventory.CheckCRDFieldFloor(crd); err != nil {
		return &TargetError{ModuleVersion: t.ModuleVersion, Rule: fmt.Sprintf(
			"its ModuleInstance CRD does not carry the fields this cli requires (%s): %v", inventory.CRDFieldFloor, err)}
	}
	return nil
}

// checkOperatorNotNewer is V1.
func checkOperatorNotNewer(t Target, cliVersion string) error {
	cli := cliVersion
	if cli != "" && cli[0] != 'v' {
		cli = "v" + cli
	}
	if !semver.IsValid(cli) {
		output.Warn("skipping the operator-version check of the install target: CLI version is not a released semver", "version", cliVersion)
		return nil
	}
	if semver.Compare(semver.MajorMinor(t.OperatorVersion), semver.MajorMinor(cli)) > 0 {
		return &TargetError{ModuleVersion: t.ModuleVersion, Rule: fmt.Sprintf(
			"it deploys opm-operator %s, newer than this CLI (%s) - upgrade the CLI first", t.OperatorVersion, cliVersion)}
	}
	return nil
}

// renderedCRD returns the rendered CustomResourceDefinition named name.
func renderedCRD(objs []*unstructured.Unstructured, name string) *unstructured.Unstructured {
	for _, obj := range objs {
		if obj.GetKind() == kindCustomResourceDefinition && obj.GetName() == name {
			return obj
		}
	}
	return nil
}

// IsRefusal reports whether err is one of install's refusals of the target
// (exit 2), as opposed to a failure to reach or read something.
func IsRefusal(err error) bool {
	var te *TargetError
	var ve *VersionError
	return errors.As(err, &te) || errors.As(err, &ve)
}
