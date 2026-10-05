// Command controller-pin writes and checks the cli's controller module pin,
// internal/controller/pin.go: the controller module version `opm controller
// install` installs by default and the controller release that version
// deploys, read from the module's own controller package without a render.
// It is not linked into the opm binary.
//
//	go run ./hack/controller-pin <version>        # task controller:pin VERSION=<version>
//	go run ./hack/controller-pin --check          # G1: the pin is served and consistent
//	go run ./hack/controller-pin select <max> <cli-version>
//
// select prints the newest published module version at or below <max>
// (bare SemVer, the cascade resolver's answer) whose controller MAJOR.MINOR is
// not above <cli-version>'s, the rule install applies to a target, and exits
// 3 when no version qualifies.
//
// The registry mapping is OPM_REGISTRY, else CUE_REGISTRY, else the cli's
// default mapping.
package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/format"
	"io"
	"os"
	"regexp"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/controller"
	"github.com/open-platform-model/cli/internal/modref"
	"github.com/open-platform-model/cli/internal/publish"
)

// pinFile is where the pin lives, relative to the repository root (the
// directory task runs in).
const pinFile = "internal/controller/pin.go"

// controllerConst is the constant the pin records the controller version in.
const controllerConst = "PinnedControllerVersion"

// exitNone is select's "no version qualifies", the cascade's stay code.
const exitNone = 3

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr, registryMapping(), pinFile))
}

func registryMapping() string {
	for _, key := range []string{"OPM_REGISTRY", "CUE_REGISTRY"} {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	return config.DefaultRegistry
}

// registry is what the pin tool reads from the registry.
type registry interface {
	modref.Source
	controller.ModuleFetcher
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer, mapping, file string) int {
	fail := func(err error) int {
		fmt.Fprintln(stderr, "controller-pin:", err)
		return 1
	}
	src, err := modref.NewSource(mapping)
	if err != nil {
		return fail(err)
	}
	reg, ok := src.(registry)
	if !ok {
		return fail(errors.New("the registry client cannot fetch module sources"))
	}

	switch {
	case len(args) == 1 && args[0] == "--check":
		return runCheck(ctx, reg, stdout, stderr, mapping, file)
	case len(args) == 3 && args[0] == "select":
		return runSelect(ctx, reg, stdout, stderr, args[1], args[2])
	case len(args) == 1 && !strings.HasPrefix(args[0], "-"):
		return runPin(ctx, reg, stdout, stderr, mapping, file, args[0])
	default:
		return fail(errors.New("usage: controller-pin <version> | --check | select <max-version> <cli-version>"))
	}
}

func runCheck(ctx context.Context, reg registry, stdout, stderr io.Writer, mapping, file string) int {
	problems, err := check(ctx, reg, mapping, file)
	if err != nil {
		fmt.Fprintln(stderr, "controller-pin:", err)
		return 1
	}
	for _, p := range problems {
		fmt.Fprintln(stderr, "controller-pin:", p)
	}
	if len(problems) > 0 {
		return 1
	}
	fmt.Fprintf(stdout, "%s: pin is served and consistent\n", file)
	return 0
}

func runSelect(ctx context.Context, reg registry, stdout, stderr io.Writer, maxVersion, cliVersion string) int {
	v, err := selectVersion(ctx, reg, maxVersion, cliVersion)
	if err != nil {
		fmt.Fprintln(stderr, "controller-pin:", err)
		return 1
	}
	if v == "" {
		fmt.Fprintf(stderr, "controller-pin: no published %s version at or below %s deploys a controller the cli %s can drive\n",
			controller.ControllerModulePath, maxVersion, cliVersion)
		return exitNone
	}
	fmt.Fprintln(stdout, v)
	return 0
}

func runPin(ctx context.Context, reg registry, stdout, stderr io.Writer, mapping, file, version string) int {
	moduleVersion, ctrlVersion, err := resolve(ctx, reg, mapping, version)
	if err == nil {
		err = writePin(file, moduleVersion, ctrlVersion)
	}
	if err != nil {
		fmt.Fprintln(stderr, "controller-pin:", err)
		return 1
	}
	fmt.Fprintf(stdout, "pinned %s %s (deploys opm-controller %s)\n", controller.ControllerModulePath, moduleVersion, ctrlVersion)
	return 0
}

// resolve resolves an exact module version (bare or "v"-prefixed) through
// the registry and reads the controller version it deploys. It returns the
// module version bare and the controller version "v"-prefixed.
func resolve(ctx context.Context, reg registry, mapping, version string) (moduleVersion, ctrlVersion string, err error) {
	sel, err := modref.ParseSelector(strings.TrimPrefix(version, "v"))
	if err != nil || sel.Exact == "" {
		return "", "", fmt.Errorf("%q is not a module version such as 0.1.0", version)
	}
	res, err := modref.Resolve(ctx, reg, modref.Request{Path: controller.ControllerModulePath, Selector: sel, Registry: mapping})
	if err != nil {
		return "", "", err
	}
	ctrlVersion, err = controller.ReadControllerVersion(ctx, reg, res.Version)
	if err != nil {
		return "", "", err
	}
	return strings.TrimPrefix(res.Version, "v"), ctrlVersion, nil
}

// renderPin is the pin file's content.
func renderPin(moduleVersion, ctrlVersion string) ([]byte, error) {
	var b bytes.Buffer
	fmt.Fprintf(&b, `// Code generated by task controller:pin; DO NOT EDIT.

package controller

// PinnedModuleVersion is the controller module version (ControllerModulePath)
// 'opm controller install' installs when --version is not given. Refresh it
// only with 'task controller:pin VERSION=<version>'.
const PinnedModuleVersion = %q

// %s is the controller release PinnedModuleVersion deploys, as the
// module's controller package states it, "v"-prefixed.
const %s = %q
`, moduleVersion, controllerConst, controllerConst, ctrlVersion)
	return format.Source(b.Bytes())
}

func writePin(file, moduleVersion, ctrlVersion string) error {
	data, err := renderPin(moduleVersion, ctrlVersion)
	if err != nil {
		return err
	}
	return os.WriteFile(file, data, 0o644) //nolint:gosec // a source file
}

var (
	moduleConstRE     = regexp.MustCompile(`(?m)^const PinnedModuleVersion = "([^"]*)"$`)
	controllerConstRE = regexp.MustCompile(`(?m)^const ` + controllerConst + ` = "([^"]*)"$`)
)

// readPin returns the two pinned values from the pin file.
func readPin(file string) (moduleVersion, ctrlVersion string, err error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", "", fmt.Errorf("reading the pin: %w", err)
	}
	m := moduleConstRE.FindSubmatch(data)
	o := controllerConstRE.FindSubmatch(data)
	if m == nil || o == nil {
		return "", "", fmt.Errorf("%s does not declare PinnedModuleVersion and %s", file, controllerConst)
	}
	return string(m[1]), string(o[1]), nil
}

// check re-resolves the pinned module version and reports every way the pin
// disagrees with what the registry serves. An error is a failure to check
// (an unreadable pin, a registry that cannot be reached), never a pass.
func check(ctx context.Context, reg registry, mapping, file string) ([]string, error) {
	moduleVersion, pinnedCtrl, err := readPin(file)
	if err != nil {
		return nil, err
	}
	_, ctrlVersion, err := resolve(ctx, reg, mapping, moduleVersion)
	if err != nil {
		var refusal *modref.RefusalError
		if errors.As(err, &refusal) {
			return []string{fmt.Sprintf("%s: %s %s is not served by the registry (%s)",
				file, controller.ControllerModulePath, moduleVersion, refusal.Refusal.Headline)}, nil
		}
		var opErr *controller.VersionError
		if errors.As(err, &opErr) {
			return []string{fmt.Sprintf("%s: %v", file, err)}, nil
		}
		return nil, fmt.Errorf("looking up %s %s: %w", controller.ControllerModulePath, moduleVersion, err)
	}
	if ctrlVersion != pinnedCtrl {
		return []string{fmt.Sprintf("%s: %s is %s, but %s %s deploys opm-controller %s; run 'task controller:pin VERSION=%s'",
			file, controllerConst, pinnedCtrl, controller.ControllerModulePath, moduleVersion, ctrlVersion, moduleVersion)}, nil
	}
	return nil, nil
}

// selectVersion walks the published module versions in max's major, at or
// below max, newest first, and returns the first (bare) whose controller
// MAJOR.MINOR is not above cliVersion's. Development builds are skipped.
func selectVersion(ctx context.Context, reg registry, maxVersion, cliVersion string) (string, error) {
	maxV := "v" + strings.TrimPrefix(maxVersion, "v")
	cli := "v" + strings.TrimPrefix(cliVersion, "v")
	if !semver.IsValid(maxV) || !semver.IsValid(cli) {
		return "", fmt.Errorf("select needs two SemVer versions, got %q and %q", maxVersion, cliVersion)
	}
	versions, err := reg.ModuleVersions(ctx, controller.ControllerModulePath)
	if err != nil {
		return "", fmt.Errorf("listing %s versions: %w", controller.ControllerModulePath, err)
	}
	semver.Sort(versions)
	for i := len(versions) - 1; i >= 0; i-- {
		v := versions[i]
		if semver.Compare(v, maxV) > 0 || semver.Major(v) != semver.Major(maxV) || publish.IsDevTag(v) {
			continue
		}
		op, err := controller.ReadControllerVersion(ctx, reg, v)
		if err != nil {
			return "", err
		}
		if semver.Compare(semver.MajorMinor(op), semver.MajorMinor(cli)) <= 0 {
			return strings.TrimPrefix(v, "v"), nil
		}
	}
	return "", nil
}
