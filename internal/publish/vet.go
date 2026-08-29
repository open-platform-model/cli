package publish

import (
	"fmt"
	"strings"

	"cuelang.org/go/cue"
)

// VetChecks runs the subset of the publish gates `opm module vet` shares with
// publish (D16, D18, D21): identity conformance against #IdentityPackage,
// metadata ↔ identity derivation, cue.mod ↔ declared-path agreement, and the
// version-major/path-major half of the tag rule. Concreteness is deliberately
// not enforced — an open Version is a valid authoring state; publish is where
// it must be filled.
//
// Returns the plan (check failures accumulate as refusals), and the loaded
// module root value so the caller can continue into values validation without
// a second load. A root that does not load is returned as an error — the
// existing vet load-failure class — while a missing or broken identity
// package is a check refusal.
func VetChecks(opts Options) (*Plan, cue.Value, error) {
	if opts.Context == nil {
		return nil, cue.Value{}, fmt.Errorf("publish: Options.Context is required")
	}
	if !opts.IdentitySchema.Exists() {
		return nil, cue.Value{}, fmt.Errorf("publish: Options.IdentitySchema is required")
	}
	absDir, err := statArtifactDir(opts.Dir)
	if err != nil {
		return nil, cue.Value{}, err
	}

	p := &Plan{Kind: opts.Kind, Dir: absDir}

	if !hasCueMod(absDir) {
		p.refuse(noCueModRefusal(opts, absDir))
		return p, cue.Value{}, nil
	}

	root, pkgName, pkgPos, refusal := loadPackage(opts, absDir, ".")
	if refusal != nil {
		return nil, cue.Value{}, fmt.Errorf("loading module package from %s: %w", absDir, refusal.Err)
	}
	a := &artifact{dir: absDir, root: root, rootPkgName: pkgName, rootPkgPos: pkgPos}

	identity, _, _, refusal := loadPackage(opts, absDir, "./identity")
	if refusal != nil {
		p.refuse(Refusal{
			Headline: "the module's identity package does not load, so its identity cannot be checked",
			Err:      refusal.Err,
		})
		return p, root, nil
	}
	a.identity = identity

	if err := a.readCueMod(opts.Context); err != nil {
		p.refuse(Refusal{Headline: "cue.mod/module.cue cannot be read", Err: err})
		return p, root, nil //nolint:nilerr // the read failure IS the refusal; the plan carries it
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

	// D18's evaluable half at vet: no tag argument exists, so the check is
	// that the declared version's major names the path's.
	if ver := p.identityField("Version"); ver != nil && ver.State == StateConcrete {
		p.Tag = "v" + ver.Value
		p.TagSource = "the artifact's own Version"
		gateTagMajor(p)
	}

	gateCueModAgreement(p, a)
	gateDerivation(p, a)
	gateKernelLoad(p, opts)

	return p, root, nil
}
