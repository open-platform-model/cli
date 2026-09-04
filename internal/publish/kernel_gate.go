package publish

import (
	"errors"

	loaderfile "github.com/open-platform-model/library/opm/helper/loader/file"
)

// gateKernelLoad (msg 12) loads a module tree through the kernel's module
// loader — the same loader `opm module build` and the operator use — and
// turns a loader refusal into a publish refusal carrying the loader's own
// error. Publish must not judge a module loadable by any test the kernel
// does not apply (enhancement 0011: publish never judges an artifact
// differently from a consumer). Modules only: a catalog is acquired by a
// platform's import, not loaded as a module.
//
// Skipped when an identity field is open or absent: those trees are already
// refused (or filled by --version) by the identity gates, and the loader
// would name the same missing value a second time. One cause, one refusal.
func gateKernelLoad(p *Plan, opts Options) {
	if p.Kind != KindModule {
		return
	}
	for _, f := range p.Identity {
		if f.State != StateConcrete {
			return
		}
	}
	p.KernelChecked = true
	_, err := loaderfile.LoadModulePackage(opts.Context, p.Dir, loaderfile.LoadOptions{Registry: opts.Registry})
	if err == nil {
		p.KernelAccepted = true
		return
	}
	p.refuse(Refusal{
		Headline: "the kernel would refuse to load this module",
		Evidence: [][]string{{"loader", err.Error()}},
		Consequence: "opm module build and the operator load modules through this loader; a\n" +
			"published tag would be unloadable everywhere it is consumed.",
		Action: kernelLoadAction(err),
	})
}

// kernelLoadAction maps the loader's sentinel to the runnable fix.
func kernelLoadAction(err error) string {
	switch {
	case errors.Is(err, loaderfile.ErrMissingRequiredField):
		return "Make the field a concrete literal in identity/identity.cue and reference it\n" +
			"from metadata (see opm module vet)"
	case errors.Is(err, loaderfile.ErrWrongKind):
		return "Publish the artifact with its own command: opm catalog publish for a catalog"
	case errors.Is(err, loaderfile.ErrInvalidPackage):
		return "Fix the package layout: exactly one CUE package at the module root"
	default:
		return "Fix the loader error above and re-run: opm module publish --dry-run"
	}
}
