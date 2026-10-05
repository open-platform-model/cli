package publish

import (
	"context"
	"fmt"
	"strings"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/module"
)

// VetChecks runs the subset of the publish gates `opm module vet` shares with
// publish (0011:D16, D18, D21): identity conformance against #IdentityPackage,
// metadata ↔ identity derivation, cue.mod ↔ declared-path agreement, and the
// version-major/path-major half of the tag rule. Concreteness is deliberately
// not enforced — an open Version is a valid authoring state; publish is where
// it must be filled.
//
// Returns the plan (check failures accumulate as refusals), and a module over
// the loaded root so the caller can continue into values validation without a
// second load. The module wraps the root as loaded here, in the caller's CUE
// runtime: it is not a kernel-acquired module, and its Metadata is nil when an
// open identity field (an authoring state vet allows) keeps the metadata from
// decoding. The plan's ModuleName carries the authored metadata.name either
// way. A root that does not load is returned as an error — the existing vet
// load-failure class — while a missing or broken identity package is a check
// refusal. The module is nil wherever no root loaded.
func VetChecks(ctx context.Context, opts Options) (*Plan, *module.Module, error) {
	if err := opts.requireInputs(); err != nil {
		return nil, nil, err
	}
	absDir, err := statArtifactDir(opts.Dir)
	if err != nil {
		return nil, nil, err
	}

	p := &Plan{Kind: opts.Kind, Dir: absDir}

	if !hasCueMod(absDir) {
		p.refuse(noCueModRefusal(opts, absDir))
		return p, nil, nil
	}

	root, pkgName, pkgPos, refusal := loadPackage(opts, absDir, ".")
	if refusal != nil {
		return nil, nil, fmt.Errorf("loading module package from %s: %w", absDir, refusal.Err)
	}
	mod := vettedModule(root)
	p.ModuleName = authoredName(root)
	a := &artifact{dir: absDir, root: root, rootPkgName: pkgName, rootPkgPos: pkgPos}

	identity, _, _, refusal := loadPackage(opts, absDir, "./identity")
	if refusal != nil {
		p.refuse(Refusal{
			Headline: "the module's identity package does not load, so its identity cannot be checked",
			Err:      refusal.Err,
		})
		return p, mod, nil
	}
	a.identity = identity

	if err := a.readCueMod(opts.Context); err != nil {
		p.refuse(Refusal{Headline: "cue.mod/module.cue cannot be read", Err: err})
		return p, mod, nil //nolint:nilerr // the read failure IS the refusal; the plan carries it
	}
	p.CueModPath, p.CueModPos = a.cueModPath, a.cueModPos

	if r := conformIdentity(a, opts.IdentitySchema); r != nil {
		p.refuse(*r)
	}
	p.Identity = identityStates(a)
	p.DeclaredPath = requireConcreteModulePath(p)
	if before, after, ok := strings.Cut(p.DeclaredPath, "@"); ok {
		p.RegistryRepo, p.Major = before, after
	}

	// 0011:D18's evaluable half at vet: no tag argument exists, so the check is
	// that the declared version's major names the path's.
	if ver := p.identityField("Version"); ver != nil && ver.State == StateConcrete {
		p.Tag = "v" + ver.Value
		p.TagSource = "the artifact's own Version"
		gateTagMajor(p)
	}

	gateCueModAgreement(p, a)
	gateDerivation(p, a)
	gateKernelLoad(ctx, p, opts)

	return p, mod, nil
}

// vettedModule wraps the loaded root as a module. An open identity field
// leaves the metadata undecodable; the package is still what vet validates,
// so the module then carries the package with nil Metadata.
func vettedModule(root cue.Value) *module.Module {
	mod, err := module.NewModuleFromValue(root)
	if err != nil {
		return &module.Module{Package: root}
	}
	return mod
}

// authoredName is the root's metadata.name when it is a concrete string, else
// "". It is read from the authored tree, so it survives an open version that
// keeps the module's metadata from decoding as a whole.
func authoredName(root cue.Value) string {
	name, err := root.LookupPath(cue.ParsePath("metadata.name")).String()
	if err != nil {
		return ""
	}
	return name
}
