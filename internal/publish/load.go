package publish

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/cue/parser"
)

// artifact is the loaded tree: the root package, the identity subpackage,
// and what cue.mod/module.cue states. cue.Values are retained (never bare
// decoded structs) so gate refusals can carry Pos().
type artifact struct {
	dir string

	// root is the artifact's root package value; rootPkgName its package
	// clause name and rootPkgPos that clause's position (file:line).
	root        cue.Value
	rootPkgName string
	rootPkgPos  string

	// identity is the ./identity package value.
	identity cue.Value

	// cueModPath is module.cue's module: line; cueModPos its position.
	cueModPath string
	cueModPos  string
	// sourceKind is module.cue's source.kind ("" when absent).
	sourceKind string
	// depVersions maps each cue.mod dependency path (major included) to its
	// pinned version — what a published artifact resolves.
	depVersions map[string]string
}

// loadArtifact loads the root package and the ./identity subpackage from dir
// with the resolved registry. A load failure is returned as the CUE error
// verbatim — an artifact that will not load has no readable identity, and
// reporting anything derived from it would blame the wrong thing.
func loadArtifact(opts Options) (*artifact, *Refusal) {
	absDir, err := filepath.Abs(opts.Dir)
	if err != nil {
		return nil, &Refusal{Headline: fmt.Sprintf("the artifact directory cannot be resolved: %v", err)}
	}
	a := &artifact{dir: absDir}

	root, pkgName, pkgPos, refusal := loadPackage(opts, absDir, ".")
	if refusal != nil {
		refusal.Headline = "the artifact does not load, so nothing about it can be judged"
		return nil, refusal
	}
	a.root, a.rootPkgName, a.rootPkgPos = root, pkgName, pkgPos

	identity, _, _, refusal := loadPackage(opts, absDir, "./identity")
	if refusal != nil {
		refusal.Headline = "the artifact's identity package does not load, so its identity cannot be read"
		return nil, refusal
	}
	a.identity = identity

	return a, nil
}

// loadPackage loads one package pattern from dir and returns its built value,
// package name, and package-clause position.
func loadPackage(opts Options, dir, pattern string) (val cue.Value, pkgName, pkgPos string, refusal *Refusal) {
	cfg := &load.Config{
		Dir: dir,
		Env: registryEnv(opts.Registry),
	}
	insts := load.Instances([]string{pattern}, cfg)
	if len(insts) == 0 {
		return cue.Value{}, "", "", &Refusal{Err: fmt.Errorf("no CUE instance found for %s in %s", pattern, dir)}
	}
	inst := insts[0]
	if inst.Err != nil {
		return cue.Value{}, "", "", &Refusal{Err: inst.Err}
	}
	v := opts.Context.BuildInstance(inst)
	if err := v.Err(); err != nil {
		return cue.Value{}, "", "", &Refusal{Err: err}
	}

	pkgName = inst.PkgName
	for _, f := range inst.Files {
		if f.PackageName() == pkgName {
			if pos := packageClausePos(f); pos != "" {
				pkgPos = relPos(dir, pos)
				break
			}
		}
	}
	return v, pkgName, pkgPos, nil
}

// packageClausePos returns the file:line of a file's package clause.
func packageClausePos(f *ast.File) string {
	for _, decl := range f.Decls {
		if pkg, ok := decl.(*ast.Package); ok {
			p := pkg.Name.Pos()
			if p.Filename() == "" {
				return ""
			}
			return fmt.Sprintf("%s:%d", p.Filename(), p.Line())
		}
	}
	return ""
}

// registryEnv returns the process environment with CUE_REGISTRY overridden
// when a registry is resolved — no os.Setenv, mirroring the library's schema
// OCILoader.
func registryEnv(registry string) []string {
	env := os.Environ()
	if registry == "" {
		return env
	}
	out := make([]string, 0, len(env)+1)
	for _, kv := range env {
		if strings.HasPrefix(kv, "CUE_REGISTRY=") {
			continue
		}
		out = append(out, kv)
	}
	return append(out, "CUE_REGISTRY="+registry)
}

// statArtifactDir resolves dir to an absolute path and requires it to be an
// existing directory. A typo'd path must surface as a plain not-found error —
// never as a gate refusal whose action ("opm mod init") cannot possibly help.
func statArtifactDir(dir string) (string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolving artifact directory: %w", err)
	}
	info, err := os.Stat(absDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("artifact directory %q does not exist", absDir)
		}
		return "", fmt.Errorf("accessing artifact directory %q: %w", absDir, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("artifact path %q is not a directory", absDir)
	}
	return absDir, nil
}

// hasCueMod reports whether dir/cue.mod/module.cue exists.
func hasCueMod(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "cue.mod", "module.cue"))
	return err == nil
}

// readCueMod reads cue.mod/module.cue's module: line (with its position),
// source.kind, and dependency pins into the artifact. Absence of individual
// fields is not an error here — the gates decide what each absence means.
func (a *artifact) readCueMod(ctx *cue.Context) error {
	path := filepath.Join(a.dir, "cue.mod", "module.cue")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	v := ctx.CompileBytes(data, cue.Filename(path))
	if err := v.Err(); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}

	if s, err := v.LookupPath(cue.ParsePath("module")).String(); err == nil {
		a.cueModPath = s
	}
	if s, err := v.LookupPath(cue.ParsePath("source.kind")).String(); err == nil {
		a.sourceKind = s
	}

	a.depVersions = map[string]string{}
	deps := v.LookupPath(cue.ParsePath("deps"))
	if deps.Exists() {
		iter, err := deps.Fields()
		if err == nil {
			for iter.Next() {
				depPath := iter.Selector().Unquoted()
				if ver, err := iter.Value().LookupPath(cue.ParsePath("v")).String(); err == nil {
					a.depVersions[depPath] = ver
				}
			}
		}
	}

	// The module: line's position, for msg 2's evidence column.
	a.cueModPos = relPos(a.dir, moduleLinePos(path, data))
	return nil
}

// moduleLinePos parses module.cue and returns the module: field's file:line.
func moduleLinePos(path string, data []byte) string {
	f, err := parser.ParseFile(path, data)
	if err != nil {
		return path
	}
	for _, decl := range f.Decls {
		field, ok := decl.(*ast.Field)
		if !ok {
			continue
		}
		if ident, ok := field.Label.(*ast.Ident); ok && ident.Name == "module" {
			return fmt.Sprintf("%s:%d", path, field.Pos().Line())
		}
	}
	return path
}

// relPos strips the artifact directory prefix from a file:line position so
// refusals name paths the way the author sees them.
func relPos(dir, pos string) string {
	if pos == "" {
		return ""
	}
	return strings.TrimPrefix(pos, dir+string(filepath.Separator))
}

// fieldPos returns a cue.Value's declaration position as a dir-relative
// file:line, or "".
func fieldPos(dir string, v cue.Value) string {
	p := v.Pos()
	if p.Filename() == "" {
		return ""
	}
	return relPos(dir, fmt.Sprintf("%s:%d", p.Filename(), p.Line()))
}
