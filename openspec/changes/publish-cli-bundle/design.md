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

`linkedVersion` reads `debug.ReadBuildInfo()` and returns the `Version` of the dependency with that path (following `Replace` when set). A `replace` to a directory yields `(devel)`, which `WritePins` refuses as not an exact version (and docs-kit would too, C15), so a dev tree cannot publish pins by accident. `cobradump.Write` adds cobra's completion command and replaces the home directory with `~` in flag defaults, as `hack/cmdref` does today.

Errors and exits: `0` with one JSON document on stdout; `1` with `docskit-dump: <message>` on stderr for a wrong argument (`usage: docskit-dump [pins]`), a missing build info ("no build info: run it with go run or go build"), a library dependency absent from the build, or a `DefaultSchemaModule` that names only a major.

Example output, `go run ./hack/docskit-dump pins`:

```json
{"schema": "docs.opmodel.dev/pins/v1", "pins": {"core": "2.0.0-beta.1", "library": "1.0.0-beta.1", "opm-operator": "1.0.0-beta.4"}}
```

The test, `hack/docskit-dump/main_test.go`, checks `pins()` against the sources `resolve-versions.sh` reads: the library `require` in `go.mod` (parsed with `golang.org/x/mod/modfile`, already a dependency), the `DefaultSchemaModule` line in the library's `opm/schema/loader.go` at that version (read from the module cache with `go list -m -json` for its directory), and `PinnedOperatorVersion`. A second test runs the dump twice in process and compares the bytes (C14's determinism, caught here before docs-kit's `check` catches it).

**As built (section 1).** `linkedVersion(info, path)` takes the build info as a parameter, so its replace handling is table-tested, and `coreRelease` refuses a `DefaultSchemaModule` without the `opmodel.dev/core@v` prefix or naming only a major. `debug.ReadBuildInfo` reports the library dependency in the test binary exactly as under `go run` (`TestPins_TestBinaryLinksTheLibrary` asserts it equals `go.mod`'s require). A directory `replace` of the library was tried with a scratch `-modfile`: `go run ./hack/docskit-dump pins` exits 1 with `docskit-dump: cobradump: pin library is "(devel)", not an exact version such as 1.0.0-beta.1 (no v)`; the program refuses it before docs-kit would. The test's file reading matches `resolve-versions.sh`'s: a `replace` of the library in `go.mod` fails it, and each constant is matched by its text, exactly once. `go test ./hack/...` was added to `task test:unit` and to the `unit` jobs of `pr.yml` and `ci.yml` (`go test ./internal/... ./hack/...`), which ran only `./internal/...`, so the test runs in CI and inside `task test`.

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
		// still reads the cli from git; both go at G2-switch.
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

A re-run of the failed run (the same `main` commit) for a tag whose bundle already exists rebuilds the same digest, which `push` treats as a no-op (C4 rule 2). A fresh `release` dispatch after `main` renamed or deleted an authored page builds a different `edit`, so a different digest, and `push` refuses it; a fix to a published bundle is a docs revision. `publish-docs` also requires `github.ref == 'refs/heads/main'`, the one ref `publish.yml` accepts, so a manual release run from another branch skips it. The Lint jobs of `pr.yml` and `ci.yml` run `.tasks/opm-docs.sh pin-check` (`task docs:pins:check`), and `.github/dependabot.yml` ignores `open-platform-model/docs-kit*` among the actions, since `publish.yml@` moves only with `.opm-docs-version`. The `cobradump` require moves by hand: Dependabot ignores `github.com/open-platform-model/*`. Tasks, `.tasks/opm-docs.sh` and `.gitignore` (`/out/`, `/.bin/`) as catalog_opm; `task check` runs `docs:bundle:check` after `docs:reference:check`.

### D5. Gate G2-pins and the release order

The cli's bundle is an anchor: `opm-docs pull` refuses a site version whose anchor pins a version without a bundle (C16 D2), and that would break every site build. The owner decided (2026-10-03) how G2-pins is met: release-mode backfills of exactly the versions the cli's `main` pins today, core `v2.0.0-beta.1`, library `v1.0.0-beta.1` and opm-operator `v1.0.0-beta.4` (each sibling's own change dispatches its backfill), then cli `v1.0.0-beta.6`, the first release with `hack/docskit-dump`. No pin moves and no cascade bump is needed. The open release PR cli#276 (beta.6) is held until adoption is merged and the three backfills verify; if the library (library#155) or operator (opm-operator#178) releases merge first and the release cascade bumps the cli's pins, those versions need bundles before cli#276 merges. Before the release PR merges, a local check proves it: build the release PR head's bundle (`task docs:bundle`), then `opm-docs pull --local cli@v1.0=out/cli` with a scratch `bundles.cue` holding opmodel.dev's planned `docs` and `versions."v1.0"` (C16 D1, D6); this needs docs-kit's `pull-docs-placement` released (gate G2-site) in the local `opm-docs`. After the release, the same pull without `--local`, anonymously, is G2-pins.

**The pre-merge check, prepared in section 1** (run it on the release PR's head; it cannot pass until the three backfills exist). The scratch `bundles.cue`, outside the repository:

```cue
registry: "ghcr.io/open-platform-model/docs"
signer: {
	issuer:   "https://token.actions.githubusercontent.com"
	workflow: "https://github.com/open-platform-model/docs-kit/.github/workflows/publish.yml"
	refs: ["refs/tags/v[0-9]*"]
}
docs: {
	cli:            {repo: "open-platform-model/cli"}
	core:           {repo: "open-platform-model/core"}
	library:        {repo: "open-platform-model/library"}
	"opm-operator": {repo: "open-platform-model/opm-operator"}
}
versions: "v1.0": {
	anchor: {project: "cli", tag: "1.0"}
	pinned: ["library", "core", "opm-operator"]
}
```

```sh
task docs:bundle
.bin/opm-docs pull --config <scratch>/bundles.cue --out <scratch>/pull --local cli@v1.0=out/cli
```

It passes when it exits 0 and `<scratch>/pull/lock.json` has `docs` entries for `library` `1.0.0-beta.1`, `core` `2.0.0-beta.1` and `opm-operator` `1.0.0-beta.4` (or whatever the head's `manifest.json` `pins` names). After the release, drop `--local` for G2-pins.

**Automatic in the release gate (review of cli#290).** `.github/scripts/release-pin-check.sh` (`task deps:release-check`, the `Release-pin gate (G1)` step of the Lint jobs on release-please branches) also runs `go run ./hack/docskit-dump pins` and, for each pin, an anonymous manifest HEAD of `ghcr.io/open-platform-model/docs/<project>:<pin>` (an anonymous token first, as the site pulls); a 401, 403 or 404 counts as missing (C16) and fails the gate naming the project, the version and the dispatch that publishes it. So cli#276's Lint job is the G2-pins gate: it fails until the three backfills exist, and passes only when every pin has a bundle. The manual `--local` pull above stays the fuller pre-merge check (signatures, lint, cross-bundle checks), and its anonymous form without `--local` the post-release confirmation. The same review found the gate looked up the nested module `github.com/open-platform-model/docs-kit/cobradump` as a repository; it now takes the repository from the first three path segments and the tag as `<subdir>/<version>` (any `/vN` suffix dropped).

Dry run on 2026-10-03 with `opm-docs` 0.4.0 (which carries `pull-docs-placement`) and the network blocked by a refusing proxy, so nothing reached GHCR: the config validated, the local anchor loaded with its pins, and the pull stopped at the first pin's registry request (`v1.0 library 1.0.0-beta.1: resolving ghcr.io/open-platform-model/docs/library:1.0.0-beta.1`), as expected.

`v1.0.0-beta.5` (today's newest release) pins library `1.0.0-beta.1`, core `2.0.0-beta.1` and the operator `1.0.0-beta.4`, but it has no hook and can never have a bundle; nothing is lost, because the site keeps reading the cli from git until G2-switch.

**Published (section 2 record, owner steps).** G2-pins held before cli#276 merged: the release gate's anonymous lookups found all three backfilled bundles, and the owner's anonymous pull resolved them (core `2.0.0-beta.1`, library `1.0.0-beta.1`, opm-operator `1.0.0-beta.4`), with the cli `1.0.0-beta.6` bundle as anchor.

| Release | Run | Digest | Pins |
| --- | --- | --- | --- |
| `v1.0.0-beta.6` | `https://github.com/open-platform-model/cli/actions/runs/37135949909` | `sha256:a1ee7187d68b5a09a56c74b2a4aa3b6534c62ad6abeb3deb05e105e5aeded775` | core `2.0.0-beta.1`, library `1.0.0-beta.1`, opm-operator `1.0.0-beta.4` |
| `v1.0.0-beta.7` | `https://github.com/open-platform-model/cli/actions/runs/37153556651` | `sha256:a15c7aa08d53142609a06ea23d25b0ab4c2f6840b7b00d962d3da6ec5268cb9d` | core `2.0.0-beta.2`, library `1.0.0-beta.3`, opm-operator `1.0.0-beta.5` |

`v1.0.0-beta.7` (the cascade bump of cli#291) is the release the site's v1.0 anchors on (opmodel.dev#38); the owner added `e2e-verified` to it after `task test:e2e` passed 65/0. Verified 2026-10-04: `ghcr.io/open-platform-model/docs/cli` is public and linked to `open-platform-model/cli`; the full tags (`1.0.0-beta.6.0`, `1.0.0-beta.7.0`) resolve to the release tags' digests, and `1.0` and `1` to beta.7's; `cosign verify` with C9's flags (`--certificate-identity-regexp '^https://github\.com/open-platform-model/docs-kit/\.github/workflows/publish\.yml@refs/tags/v[0-9]'`, `--certificate-github-workflow-repository open-platform-model/cli`, `--certificate-github-workflow-ref refs/heads/main`) passes for both digests; each bundle's `manifest.json` `pins` are the ones in the table.

### Parity record (section 1)

At cli commit `f954083a` (main `1d9f475b` plus the dump program and `docs-kit.cue`), `task docs:bundle` with `opm-docs` 0.4.0 wrote 16 pages, linted green: the six authored pages of `docs/site/` and the ten generated pages under `reference/cli/`. Each of the ten equals the committed `docs/site/reference/cli/` page of the same name once cmdref's two marker comments are removed (the begin comment with its following blank line, and the end comment with its preceding newline, as docs-kit's `compareWithCmdref` strips them), and the page sets are equal: no difference at all, so nothing beyond C19's parity record. docs-kit's `TestCLICommandParityLive` with `OPM_CLI_CHECKOUT` naming this tree passes too. The manifest's `pins` are `core 2.0.0-beta.1`, `library 1.0.0-beta.1`, `opm-operator 1.0.0-beta.4`.

### D6. Retiring cmdref at G2-switch and G2-edge

**Gate.** G2-switch holds (opmodel.dev#38, 2026-10-04: v1.0 anchors on the cli 1.0.0-beta.7 bundle). G2-edge (docs-kit orchestration; defined in docs-kit#43, which merges first) holds once opmodel.dev's `add-edge-build` switches the site's `sources-main` job: which checks every repository's `main` together, then reads the cli's `main` from its `edge` docs bundle instead of a `main` checkout. Retiring earlier would leave that job reading a `docs/site/` without `reference/cli/`, failing every `main` page that links a command page (owner decision 2026-10-04 on `pull-reference-bundles` OQ1).

The same deletion breaks opmodel.dev's local explicit-mode builds (`OPM_VERSIONS=v1.0=/src task build|serve`), which read the cli's `docs/site/` in place; after retirement a local check of every `main` is opmodel.dev's `task build:edge` (with `OPM_BUNDLES_LOCAL` for an unmerged cli tree), or `OPM_DOCS_BUNDLES=1` after a pull, as `add-edge-build` documents. The retirement archives this change only once section 2 is checked and D5 records the runs.

Delete `internal/cmdref/`, `hack/cmdref/`, `docs/site/reference/cli/`, the `docs:reference` and `docs:reference:check` tasks and their comment, their line in `task check`, and the `command-reference` jobs of `ci.yml` and `pr.yml`; the `markdown` source loses its `exclude` and the comment about it. `AGENTS.md` loses the `internal/cmdref/` layout entry (section 1 already added the `hack/docskit-dump/` entry; it gains "help text and flags are the reference; check with `task docs:bundle:check`"), the `docs:reference` line of "Core commands", and the "Docs bundles" paragraph's sentence about the excluded committed pages ("The committed `docs/site/reference/cli/` pages stay excluded ..."), keeping its second half: help text reaches the site only through a release.

**The required check.** "Command Reference (current)" is required nowhere. `gh api repos/open-platform-model/cli/rules/branches/main` (2026-10-04) lists every rule that applies to `main` from any ruleset, organization ones included: a single `workflows` rule (the organization `mention-guard` ruleset, requiring `mention-guard.yml`) and no `required_status_checks`; `main` has no classic branch protection; the owner confirmed the same day that no organization ruleset requires it. Deleting the job blocks nothing. If the planned PR-only ruleset ever lists required checks, it must not list this one.

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
- After G2-switch, a help-text fix reaches the site only through a cli release: a docs revision applies only Markdown or comment changes (C3), and help text is Go strings (C19 risks). Revisions of authored pages are dispatched by hand for now (cli#282, tracked in docs-kit#16).
- `go run` in the docs build downloads the cli's modules in docs-kit's build job; a slow proxy lengthens it, within C14's 10-minute limit.
- "Command Reference (current)" is required by no ruleset (D6, read 2026-10-04 and confirmed by the owner); a future ruleset that lists it would block every PR once the job is gone.
