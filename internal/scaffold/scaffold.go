package scaffold

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/mod/modconfig"
	"cuelang.org/go/mod/module"

	"github.com/open-platform-model/library/opm/compat"
	loaderfile "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/kernel"

	"github.com/open-platform-model/cli/internal/cueedit"
	"github.com/open-platform-model/cli/internal/publish"
)

// InitialVersion is the version every scaffold starts at, written in the
// defaulted form so release automation owns it from the first commit (D12).
const InitialVersion = "0.1.0"

// modulePathRE is the shape a new module path must have: a dotted first
// segment (what makes it a module path rather than a bare word), at least one
// more segment, and a terminal major.
var modulePathRE = regexp.MustCompile(`^[a-z0-9._-]*\.[a-z0-9._-]+(/[a-z0-9._-]+)+@v\d+$`)

// packageNameRE is the subset of #SnakeNameType a CUE package clause can
// bind: snake_case starting with a letter.
var packageNameRE = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// ValidateNewModulePath checks the path a scaffold will be re-identified to:
// module-path shaped with an explicit major (init never invents one), and a
// leaf that is a valid CUE package name, because the leaf becomes the root
// package, metadata.name, and the identity's path leaf all at once (0010 D8).
func ValidateNewModulePath(path string) error {
	if !modulePathRE.MatchString(path) {
		return fmt.Errorf("module path %q must be <domain>/<...>/<name>@v<major> (e.g. example.com/modules/my_app@v0)", path)
	}
	if leaf := Leaf(path); !packageNameRE.MatchString(leaf) {
		return fmt.Errorf("module path leaf %q must be snake_case starting with a letter — it becomes the module's package name and metadata.name", leaf)
	}
	return nil
}

// Leaf returns the path's last segment, major stripped.
func Leaf(modulePath string) string {
	base, _, _ := strings.Cut(modulePath, "@")
	return base[strings.LastIndex(base, "/")+1:]
}

// ResolveTemplateVersion selects the version a reference resolves to over
// the published history: an exact reference must exist; a floating one takes
// the newest release within its constraint — stable preferred, prerelease
// fallback on a prerelease-only history (compat.HighestStable, its first true
// caller). Returns the `v`-prefixed tag.
//
// Failure modes keep D25's honesty contract: a transport failure is a
// *publish.ConnectivityError naming the lookup and registry; an empty or
// non-matching history is a *publish.Refusal naming the expanded path — a
// shortcut typo 404s inside the reserved segment, never falling back.
func ResolveTemplateVersion(ctx context.Context, registry string, ref TemplateRef) (string, error) {
	client, err := publish.NewRegistryClient(registry)
	if err != nil {
		return "", err
	}
	lookup := ref.LookupPath()
	versions, err := client.ModuleVersions(ctx, lookup)
	if err != nil {
		return "", &publish.ConnectivityError{
			Op:  fmt.Sprintf("listing published versions of %s (registry %s)", lookup, registry),
			Err: err,
		}
	}

	if ref.Exact != "" {
		want := "v" + ref.Exact
		for _, v := range versions {
			if v == want {
				return want, nil
			}
		}
		return "", &RefusalError{publish.Refusal{
			Headline:    fmt.Sprintf("%s has no published version %s", ref.Base, want),
			Evidence:    refEvidence(ref, registry),
			Consequence: "An exact reference pins one tag; nothing substitutes for it.",
			Action:      fmt.Sprintf("Pick a published version, or float within the major:  %s@%s", ref.Raw[:strings.LastIndex(ref.Raw, "@")], "v"+strings.SplitN(ref.Exact, ".", 2)[0]),
		}}
	}

	if len(versions) == 0 {
		headline := fmt.Sprintf("%s has no published versions", lookup)
		if ref.Shortcut {
			headline = fmt.Sprintf("no template %q — %s has no published versions", ref.Raw, lookup)
		}
		return "", &RefusalError{publish.Refusal{
			Headline:    headline,
			Evidence:    refEvidence(ref, registry),
			Consequence: "A template reference resolves inside the path it names; nothing falls\nback elsewhere.",
			Action:      "List the official templates:  opm module template list",
		}}
	}
	return compat.HighestStable(versions), nil
}

// refEvidence renders the resolution inputs a refusal reports: the reference,
// its expansion when one happened, and the registry consulted.
func refEvidence(ref TemplateRef, registry string) [][]string {
	ev := [][]string{{"reference", ref.Raw}}
	if ref.Shortcut {
		ev = append(ev, []string{"expanded to", ref.LookupPath()})
	}
	ev = append(ev, []string{"registry", registry})
	return ev
}

// RefusalError carries a publish.Refusal through an error return, so scaffold
// failures print through the house refusal funnel with exit 2.
type RefusalError struct {
	Refusal publish.Refusal
}

func (e *RefusalError) Error() string { return e.Refusal.Headline }

// Result describes a completed scaffold.
type Result struct {
	// Dir is the created module directory.
	Dir string
	// Files are the dir-relative paths written, sorted.
	Files []string
	// TemplatePath and TemplateVersion name the fetched source.
	TemplatePath    string
	TemplateVersion string
}

// Run scaffolds newPath from the template at ref: resolve the version,
// acquire the module (the kernel's acquire runs the identity verification —
// a tampered artifact refuses at fetch), copy the fetched tree into dest,
// re-identify it wholesale, and assert the result still parses and derives.
// dest must not exist; on any failure after creation it is removed.
func Run(ctx context.Context, k *kernel.Kernel, registry, newPath string, ref TemplateRef, dest string) (*Result, error) {
	if err := ValidateNewModulePath(newPath); err != nil {
		return nil, err
	}
	version, err := ResolveTemplateVersion(ctx, registry, ref)
	if err != nil {
		return nil, err
	}
	templatePath := ref.Base + "@" + majorOf(version)

	// The acquire is the trust gate: same fetch path as the whole CLI, same
	// CUE disk cache, and the identity verification a raw copy would skip.
	if _, err := k.AcquireModuleFromRegistry(ctx, templatePath, version); err != nil {
		return nil, &publish.ConnectivityError{
			Op:  fmt.Sprintf("acquiring %s %s (registry %s)", templatePath, version, registry),
			Err: err,
		}
	}

	if _, err := os.Stat(dest); err == nil {
		return nil, fmt.Errorf("directory already exists: %s", dest)
	}
	files, err := copyFetched(ctx, registry, templatePath, version, dest)
	if err != nil {
		_ = os.RemoveAll(dest)
		return nil, err
	}

	if err := Reidentify(dest, templatePath, newPath); err != nil {
		_ = os.RemoveAll(dest)
		return nil, err
	}

	if err := assertDerives(ctx, k, registry, dest, newPath); err != nil {
		_ = os.RemoveAll(dest)
		return nil, err
	}

	sort.Strings(files)
	return &Result{Dir: dest, Files: files, TemplatePath: templatePath, TemplateVersion: version}, nil
}

// Reidentify rewrites a freshly cloned tree from oldPath's identity to
// newPath's: the cue.mod `module:` line, the identity package's ModulePath
// and Version (set to the initial version as a plain literal), every literal
// self-import, and every root-package clause (old leaf → new leaf). This is
// D16's statement set plus the package clause, applied wholesale — correct
// exactly because the user owns nothing in the tree yet; repair mode refuses
// the same rewrite for the opposite reason. Metadata is untouched: it
// derives (D12).
func Reidentify(dir, oldPath, newPath string) error {
	if _, err := cueedit.SetCueModModule(dir, newPath); err != nil {
		return fmt.Errorf("re-identifying cue.mod: %w", err)
	}
	if _, err := cueedit.SetIdentityModulePath(dir, newPath); err != nil {
		return fmt.Errorf("re-identifying identity ModulePath: %w", err)
	}
	if _, err := cueedit.SetIdentityVersion(dir, InitialVersion); err != nil {
		return fmt.Errorf("resetting identity Version: %w", err)
	}
	if _, err := cueedit.RewriteSelfImports(dir, oldPath, newPath); err != nil {
		return fmt.Errorf("rewriting self-imports: %w", err)
	}
	if _, err := cueedit.RenamePackageClauses(dir, Leaf(oldPath), Leaf(newPath)); err != nil {
		return fmt.Errorf("renaming package clauses: %w", err)
	}
	return nil
}

// copyFetched copies the fetched module's source tree into dest. The fetch
// resolves from the same CUE module cache the acquire just warmed, so no
// second network round-trip happens. Returns the dir-relative files written.
func copyFetched(ctx context.Context, registry, modulePath, version, dest string) ([]string, error) {
	reg, err := modconfig.NewRegistry(&modconfig.Config{CUERegistry: registry})
	if err != nil {
		return nil, fmt.Errorf("building registry resolver: %w", err)
	}
	mv, err := module.NewVersion(modulePath, version)
	if err != nil {
		return nil, fmt.Errorf("parsing %s@%s: %w", modulePath, version, err)
	}
	loc, err := reg.Fetch(ctx, mv)
	if err != nil {
		return nil, &publish.ConnectivityError{
			Op:  fmt.Sprintf("fetching %s %s (registry %s)", modulePath, version, registry),
			Err: err,
		}
	}

	root := loc.Dir
	if root == "" {
		root = "."
	}
	var files []string
	err = fs.WalkDir(loc.FS, root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		data, err := fs.ReadFile(loc.FS, p)
		if err != nil {
			return fmt.Errorf("reading %s: %w", p, err)
		}
		rel := p
		if root != "." {
			rel = strings.TrimPrefix(strings.TrimPrefix(p, root), "/")
		}
		target := filepath.Join(dest, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		//nolint:gosec // G306: scaffolded source files are project files, not secrets
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return err
		}
		files = append(files, filepath.FromSlash(rel))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("copying %s %s: %w", modulePath, version, err)
	}
	return files, nil
}

// assertDerives is the post-rewrite assertion: the scaffolded tree loads and
// its metadata derives the new identity — modulePath the new path, version
// the initial default. A load failure is an internal error (the writers left
// an inconsistent tree). A value MISMATCH is a property of the clone source:
// re-identification rewrote the identity package, so metadata still stating
// the old values means the source carries literals instead of the D12
// derivation — official templates cannot hit this (their derivation is
// gate-enforced at publish), an arbitrary `--from` donor can, and it earns a
// refusal naming the donor's defect rather than a blamed-wrong internal
// error.
func assertDerives(ctx context.Context, k *kernel.Kernel, registry, dir, newPath string) error {
	val, err := k.LoadModulePackage(ctx, dir, loaderfile.LoadOptions{Registry: registry})
	if err != nil {
		return fmt.Errorf("internal error: scaffolded tree does not load: %w", err)
	}
	for field, want := range map[string]string{
		"modulePath": newPath,
		"version":    InitialVersion,
	} {
		got, err := val.LookupPath(cue.ParsePath("metadata." + field)).String()
		if err != nil {
			return fmt.Errorf("internal error: scaffolded metadata.%s does not evaluate: %w", field, err)
		}
		if got != want {
			return &RefusalError{publish.Refusal{
				Headline: fmt.Sprintf("the clone source does not derive metadata.%s from its identity package", field),
				Evidence: [][]string{
					{"metadata." + field, got},
					{"expected", want, "derived from identity/identity.cue after re-identification"},
				},
				Consequence: "Re-identification rewrites the identity package and everything that\nderives from it; metadata carrying literals stays stamped with the\nsource's old identity (D12).",
				Action:      fmt.Sprintf("Clone a module whose metadata derives (%s: id.%s), or start\nfrom an official template:  opm module template list", field, deriveField(field)),
			}}
		}
	}
	return nil
}

// deriveField maps a metadata field to the identity export it derives from.
func deriveField(metaField string) string {
	if metaField == "version" {
		return "Version"
	}
	return "ModulePath"
}

// majorOf returns the major a v-prefixed tag names ("v1.2.3" → "v1").
func majorOf(tag string) string {
	return "v" + strings.SplitN(strings.TrimPrefix(tag, "v"), ".", 2)[0]
}
