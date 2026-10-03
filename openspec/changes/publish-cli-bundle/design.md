## Context

The cli releases with release-please and goreleaser (`release.yml`: `release-please` creates the tag and a draft; `publish-templates`; `goreleaser` uploads assets and publishes the draft last; a `workflow_dispatch` with `tag` finishes a draft by hand). `internal/cmdref` and `hack/cmdref` write `docs/site/reference/cli/` (`_index.md` with the global flags, one `opm-<command>.md` per top-level command); `task docs:reference:check` runs in `task check` and in the "Command Reference (current)" jobs of `ci.yml` and `pr.yml`. The site reads the cli's pins from git: the library version from `go.mod` (`github.com/open-platform-model/library v1.0.0-beta.1`), core from the library's `opm/schema/loader.go` `DefaultSchemaModule` (`opmodel.dev/core@v2.0.0-beta.1`), the operator from `internal/operator/manifest.go` `PinnedOperatorVersion` (`v1.0.0-beta.4`) (`opmodel.dev/site/scripts/resolve-versions.sh` lines 222-294).

docs-kit contracts read: C5 (`publish.yml`, the `setup-go` input), C6, C9, C12, C14 (repository commands: argv run in the source tree, one JSON document on stdout, `check` runs it twice and refuses differing output, never run in `push`/`promote`/`pull`), C15 (docs placement, `pins` in `manifest.json`), C16 (a site version's anchor and pinned projects, refused when a pin has no bundle), C19 (`cobradump` API and dump format, `cobra` config and pages). Reference adopter: catalog_opm.

## Goals / Non-Goals

**Goals:** the cli's docs bundle with its authored pages, its command reference at today's URLs and its pins; publishing on every release; a first release whose pins resolve (G2-pins); deleting cmdref once the site reads the bundle.

**Non-Goals:** help-text changes; a hidden `opm` command; reading pins any other way than the site does today; the site's switch.

## Decisions

### D1. `hack/docskit-dump`

Syntax: `go run ./hack/docskit-dump [pins]`. No flags. Run by docs-kit in the source tree (C14) and by `task docs:bundle`; not built into `opm`.

```go
// Command docskit-dump prints the opm command tree, or with "pins" the
// versions this cli build documents against, as the JSON documents docs-kit
// reads (docs-kit C14, C19).
package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/open-platform-model/docs-kit/cobradump"
	"github.com/open-platform-model/library/opm/schema"

	"github.com/open-platform-model/cli/internal/cmd"
	"github.com/open-platform-model/cli/internal/operator"
)

const libraryModule = "github.com/open-platform-model/library"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "docskit-dump:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	switch {
	case len(args) == 0:
		return cobradump.Write(cmd.NewRootCmd(), os.Stdout, cobradump.Options{})
	case len(args) == 1 && args[0] == "pins":
		p, err := pins()
		if err != nil {
			return err
		}
		return cobradump.WritePins(os.Stdout, p)
	default:
		return fmt.Errorf("usage: docskit-dump [pins]")
	}
}

// pins returns the versions this build compiles in: the library module the
// binary links, the core release that library's schema loader pins, and the
// operator release the cli installs. Each is bare SemVer, without "v".
func pins() (map[string]string, error) {
	lib, err := linkedVersion(libraryModule)
	if err != nil {
		return nil, err
	}
	core, ok := strings.CutPrefix(schema.DefaultSchemaModule, "opmodel.dev/core@v")
	if !ok {
		return nil, fmt.Errorf("library DefaultSchemaModule %q pins no exact core release", schema.DefaultSchemaModule)
	}
	return map[string]string{
		"library":      strings.TrimPrefix(lib, "v"),
		"core":         core,
		"opm-operator": strings.TrimPrefix(operator.PinnedOperatorVersion, "v"),
	}, nil
}
```

`linkedVersion` reads `debug.ReadBuildInfo()` and returns the `Version` of the dependency with that path (following `Replace` when set). A `replace` to a directory yields `(devel)`, which `WritePins` would print and docs-kit refuses as not SemVer (C15 D7: exit 2 naming the value), so a dev tree cannot publish pins by accident. `cobradump.Write` adds cobra's completion command and replaces the home directory with `~` in flag defaults, as `hack/cmdref` does today.

Errors and exits: `0` with one JSON document on stdout; `1` with `docskit-dump: <message>` on stderr for a wrong argument (`usage: docskit-dump [pins]`), a missing build info ("no build info: run it with go run or go build"), a library dependency absent from the build, or a `DefaultSchemaModule` that names only a major.

Example output, `go run ./hack/docskit-dump pins`:

```json
{"schema": "docs.opmodel.dev/pins/v1", "pins": {"core": "2.0.0-beta.1", "library": "1.0.0-beta.1", "opm-operator": "1.0.0-beta.4"}}
```

The test, `hack/docskit-dump/main_test.go`, checks `pins()` against the sources `resolve-versions.sh` reads: the library `require` in `go.mod` (parsed with `golang.org/x/mod/modfile`, already a dependency), the `DefaultSchemaModule` line in the library's `opm/schema/loader.go` at that version (read from the module cache with `go list -m -json` for its directory), and `PinnedOperatorVersion`. A second test runs the dump twice in process and compares the bytes (C14's determinism, caught here before docs-kit's `check` catches it).

### D2. `go.mod`

`require github.com/open-platform-model/docs-kit/cobradump v0.1.0` (or the newest `cobradump/v*`). cobradump depends only on `github.com/spf13/cobra` and `github.com/spf13/pflag`, which the cli already requires (`cobra v1.10.2`); minimal version selection keeps the cli's versions when they are newer. Section 1 verifies `go mod graph` gains exactly the cobradump node and `go.sum` gains only its lines. The root `docs-kit` module (CUE, oras-go, sigstore-go) is never required.

### D3. `docs-kit.cue`

```cue
bundles: cli: {
	placement: {kind: "docs", root: "/docs/", owns: ["reference/cli/"]}
	version: {from: "tag", prefix: "v"}
	pins: {command: ["go", "run", "./hack/docskit-dump", "pins"], projects: ["library", "core", "opm-operator"]}
	sources: [{
		kind:        "cobra"
		command:     ["go", "run", "./hack/docskit-dump"]
		section:     "reference/cli/"
		title:       "CLI Reference"
		description: "Every opm command and flag, generated from the CLI's cobra commands."
		weight:      2
	}, {
		// The authored pages ship in the same bundle (docs-kit DESIGN decision
		// 20). The exclude keeps cmdref's committed pages out while the site
		// still reads the cli from git; section 3 deletes both.
		kind: "markdown", dir: "docs/site", exclude: ["reference/cli/"]
	}]
}
```

The title, description and weight are what `command-reference` requires of `_index.md` today. `docs/site/reference/registry-namespaces.md` is authored and outside `reference/cli/`, so it ships as a normal page. `citations` stays `strip`: help text carries no citations since cli PR 275.

### D4. Publishing

`docs.yml` is catalog_opm's with `project: cli`, tags `vX.Y.Z`, and `setup-go: true` on every job (the dump and pins run `go run`). Its dispatch comment states the backfill floor: release mode needs `hack/docskit-dump` in the tag's tree, so `v1.0.0-beta.5` and earlier can never get a bundle; the first bundled release is the one after section 1.

`release.yml` gains:

```yaml
  publish-docs:
    name: Publish the cli docs bundle
    needs: [release-please, goreleaser]
    # always(): on a manual run release-please is skipped. The bundle follows
    # the published release, so a draft that never published has none.
    if: always() && needs.goreleaser.result == 'success'
    permissions:
      contents: read
      packages: write
      id-token: write
    # Pinned by docs-kit release tag, not a SHA: the signing certificate names
    # publish.yml at this ref, and the site trusts only docs-kit's v* tags
    # (docs-kit C5, C9). Moves with .opm-docs-version in one PR.
    uses: open-platform-model/docs-kit/.github/workflows/publish.yml@vX.Y.Z
    with:
      project: cli
      mode: release
      tag: ${{ needs.release-please.outputs.tag_name || inputs.tag }}
      setup-go: true
```

A manual re-run for a tag whose bundle already exists rebuilds the same digest, which `push` treats as a no-op (C4 rule 2). Tasks, `.tasks/opm-docs.sh` and `.gitignore` (`/out/`, `/.bin/`) as catalog_opm; `task check` runs `docs:bundle:check` after `docs:reference:check`.

### D5. Gate G2-pins and the release order

The cli's bundle is an anchor: `opm-docs pull` refuses a site version whose anchor pins a version without a bundle (C16 D2), and that would break every site build. So section 2 runs only after the release cascade has moved the cli's pins to bundled versions: `go.mod` to the library release of `publish-go-api-bundle` section 3 (whose `DefaultSchemaModule` names core's bundled release) and `PinnedOperatorVersion` to a release of `publish-crd-bundle` section 2. Before the release PR merges, a local check proves it: build the release PR head's bundle (`task docs:bundle`), then `opm-docs pull --local cli@v1.0=out/cli` with a scratch `bundles.cue` holding opmodel.dev's planned `docs` and `versions."v1.0"` (C16 D1, D6). After the release, the same pull without `--local`, anonymously, is G2-pins.

`v1.0.0-beta.5` (today's newest release) pins library `1.0.0-beta.1`, core `2.0.0-beta.1` and the operator `1.0.0-beta.4`, but it has no hook and can never have a bundle; nothing is lost, because the site keeps reading the cli from git until G2-switch.

### D6. Section 3 at G2-switch

Delete `internal/cmdref/`, `hack/cmdref/`, `docs/site/reference/cli/`, the `docs:reference` and `docs:reference:check` tasks and their comment, their line in `task check`, and the `command-reference` jobs of `ci.yml` and `pr.yml`; the `markdown` source loses its `exclude`. `AGENTS.md`'s `internal/cmdref/` entry becomes a `hack/docskit-dump/` entry ("help text and flags are the reference; check with `task docs:bundle:check`").

## Research & Decisions

### Where pins come from

**Context**: orchestration asks that each pin be read where `resolve-versions.sh` reads it today.
**Options considered**:
1. Read the files (`go.mod`, the library's `loader.go` from the module cache, `manifest.go`) in the program - the same text the site reads, but the program would parse Go source and need the module cache path.
2. Use what the build compiles in (`debug.ReadBuildInfo`, `schema.DefaultSchemaModule`, `operator.PinnedOperatorVersion`), and test that it equals those files - the values are the ones the shipped binary uses; the test keeps the site's reading and the program's answer equal.
**Decision**: option 2.
**Rationale**: what the cli pins is what it links and installs; reading constants directly is simpler than parsing source, and the test still binds the result to the files the site read before.

### A nested module of the cli for the dump program

**Context**: the owner asked to keep `go.mod` clean.
**Options considered**: 1. `hack/docskit-dump` in its own nested module - `go.mod` untouched, but a nested module cannot import the cli's `internal/` packages (`internal/cmd`, `internal/operator`), so it cannot build the command tree; 2. the program in the cli module, requiring only the nested `cobradump` module - one new `require` line.
**Decision**: option 2; "clean" is met by cobradump being a nested docs-kit module with no dependency the cli lacks (D2).

### `publish-docs` after `goreleaser`

**Decision**: gate on `needs.goreleaser.result == 'success'` with `always()`, as `goreleaser` itself gates on its needs.
**Rationale**: goreleaser publishes the release last; a bundle for a release that stays a draft would let a site version anchor on a cli nobody can download.

## Risks / Trade-offs

- G2-pins is a cross-repository ordering gate: a cli release cut before its pins have bundles breaks every site pull until a later cli release fixes it (release tags are immutable). Section 2's pre-merge check is the guard; the owner merges the release PR.
- After G2-switch, a help-text fix reaches the site only through a cli release: a docs revision applies only Markdown or comment changes (C3), and help text is Go strings (C19 risks).
- `go run` in the docs build downloads the cli's modules in docs-kit's build job; a slow proxy lengthens it, within C14's 10-minute limit.
- "Command Reference (current)" may be a required check in the ruleset; deleting the job in section 3 would block every PR until the owner removes it from the list.
