## Context

See proposal.md for why. The state this design works from (cli `main` at `025e1b6`, tag `v1.0.0-alpha.27`):

- `release-please-config.json`: one package `.`, `versioning: prerelease`, `prerelease: true`, `prerelease-type: alpha`. Manifest `1.0.0-alpha.27`. release-please's prerelease strategy only uses `prerelease-type` when the current version is stable; on `1.0.0-alpha.27` it proposes `1.0.0-alpha.28` whatever the type says.
- `.goreleaser.yml` has a `release:` block with only `extra_files`. goreleaser updates the GitHub Release release-please created and, with no `prerelease` key, forces it to non-prerelease (verified: `v1.0.0-alpha.27` has `isPrerelease=false`, notes authored by `opm-release-please[bot]`).
- `internal/inventory/gates.go` `GateOperatorVersionCeiling` refuses when `semver.Compare(op, cli) > 0`, prerelease counter included. It runs from `RunClusterGates` (`internal/workflow/apply/apply.go`) with `version.Version`, which only goreleaser's ldflags set; a `go install` build reports `dev` and skips the gate (so the operator's CI never exercises it).
- `internal/operator/manifest.go` pins `v1.0.0-alpha.19`; the operator is at `v1.0.0-alpha.22`, whose release carries `install.yaml`.
- `templates/{minimal,standard,advanced}`: identity `1.0.2`, `cue.mod` pins core `v2.0.0-alpha.13` and catalogs/opm `v4.4.3`; GHCR `v1.0.2` holds core `alpha.10` and opm `4.4.0` because the pin bumps never bumped identity and `publish-templates.sh` skips published versions. `opm module init` copies the template's pins verbatim.
- `go.mod` pins `github.com/open-platform-model/library v1.0.0-alpha.36`. Go `@latest` for the library resolves `v0.7.0` (the last stable tag), so every bump names the version explicitly.
- `openspec/specs/release-workflow/spec.md` still says "`alpha` prerelease version line" and "no `workflow_dispatch`", which the workflow has contradicted since the `v1.0.0-alpha.21` recovery.

Constraints from the cutover canon: the version crosses only through a `Release-As` footer in the final squash commit on main, written by the supervisor at merge time; workers never merge; root workspace tasks run only on the supervisor's main checkouts and reach this worktree as patch files; shipped bumps and fixture bumps never share a squash commit.

## Goals / Non-Goals

**Goals:**

- cli `v1.0.0-beta.1` links library `v1.0.0-beta.1`, ships templates `1.0.3` on core `v2.0.0-beta.1`, embeds the newest operator release that exists before it (`v1.0.0-alpha.22`), and is flagged Pre-release on GitHub.
- The operator can release beta patches and counters ahead of the cli without breaking `opm module apply`.
- Later releasable commits propose `1.0.0-beta.N+1` with no further intervention.

**Non-Goals:**

- Embedding the operator beta. It does not exist before this release, and it must not ship before a cli that tolerates it; the refresh is a later plain `fix(deps): embed opm-operator v1.0.0-beta.1` commit (gate G5), expected to cut the next cli beta (G6).
- Re-pinning the published test fixtures (`tests/fixtures`, `tests/e2e/testdata`, `tests/integration/.../testdata`). That is the supervisor's `deps:pins:fixtures` run, one separate `test(fixtures)` PR.
- Re-pinning downstream workflows to the cli beta tag (`deps:pins:opm-cli`, after G4).
- Changing `release.yml` itself. The recovery input already exists; only the spec catches up.
- GA. Dropping the suffix is `prerelease: false` plus a new carrier, later.

## Decisions

### D1. Ceiling compares MAJOR.MINOR with `semver.MajorMinor`

```go
// GateOperatorVersionCeiling ... compares MAJOR.MINOR only: the cli and the
// operator share a minor line and release patches and prerelease counters
// independently.
if semver.Compare(semver.MajorMinor(normalizedOp), semver.MajorMinor(normalizedCLI)) > 0 {
    return fmt.Errorf(
        "your CLI (%s) is older than the cluster operator (%s) — upgrade the CLI before applying against this cluster",
        cliVersion, opVersion,
    )
}
```

`golang.org/x/mod/semver.MajorMinor` returns `vX.Y` for any valid version, dropping patch, prerelease and build (`v1.1.0-beta.1` gives `v1.1`, `v1.0.0-beta.1+gabc.dirty` gives `v1.0`). The skip paths (dev build, invalid semver, absent Platform, RBAC denial) are unchanged, and so is the error text, which existing tests and users match on.

Options considered:
1. Keep the strict compare and have the supervisor enforce ordering (freeze operator main from G5 to G6, follow every operator release with a cli release). Fragile: operator `docs` commits cut releases, and counters restart at `beta.1` in both repos, so headroom is 0 to +1.
2. Give the cli a higher first beta counter (for example `beta.6`) as headroom. Buys time, fixes nothing, and makes the counters meaningless.
3. Compare MAJOR.MINOR.PATCH, ignoring only the prerelease. Still refuses operator patch releases the cli has not matched.
4. **Compare MAJOR.MINOR (chosen).** Owner decision 3; matches the 0021 OQ14 position that cli and operator share MAJOR.MINOR and release patches independently. The gate exists to stop a CLI that predates a CRD or protocol change on a new minor, which it still does.

Exit code and message: unchanged (non-zero, "your CLI (X) is older than the cluster operator (Y) — upgrade the CLI ..."). The refusal now fires only across a minor boundary.

### D2. The version line moves by a squash-message footer, never by config

`prerelease-type` becomes `beta` so that later releases count on the beta line and a future stable-then-prerelease step picks beta. The line change itself is `Release-As: 1.0.0-beta.1` as the last paragraph of PR-B's squash commit message, before the plain `Co-Authored-By` trailer. `release-as` in the config is sticky and produced the duplicate release PR #109 (reverted in ed9774e); `.release-please-manifest.json` is release-please's to write. The worker makes its final branch commit carry exactly the intended squash message so mention-guard scans it; the supervisor still writes the message at merge time.

### D3. Three PRs, one OpenSpec change

- **PR-A** (section 1): the ceiling gate and its spec delta, plus the change's planning artifacts. No gate; merges first so the relaxed gate is on main long before any operator beta. Squash type `fix(inventory)` makes it releasable: release-please opens `chore(main): release 1.0.0-alpha.28`, which the supervisor leaves open; PR-B's footer retitles it.
- **PR-B** (sections 2 to 5 and 7): every shipped change plus the archive; the carrier. Gate G2 (library on the Go proxy) and G3 (catalogs k8s and opm on GHCR, which implies G1 core).
- **PR-C** (section 6, own branch `beta/cli-fixture-pins`): `hack/platform`, `hack/kind-platform.yaml`, `examples/` from the supervisor's `deps:update` patch. `test(fixtures)`, no release. Gate G3.

Options considered: one PR (breaks the never-mix rule and holds the gate fix hostage to G2/G3); one PR per section (six release-please churns and five alpha release PR retitles for no benefit). The carrier carries the archive because it is the last PR of the change; PR-C's commit lands on its own branch but its checkbox is ticked in this tasks.md before the archive.

### D4. Operator embed stays on the newest alpha for beta.1

`task operator:sync VERSION=v1.0.0-alpha.22` refreshes the embed (manifest + digest-pinned image). Waiting for the operator beta would invert the release order the ceiling gate needs (cli before operator). With D1 the order matters less, but the operator beta is only built after the cli beta (G4 before G5), so it cannot be embedded here anyway.

### D5. Templates bump identity with the pins

After the supervisor's `deps:update` patch re-pins `templates/*/cue.mod` (core `v2.0.0-beta.1`, catalogs/opm `v4.4.4`), each template gets `opm module version set 1.0.3 templates/<name>` with an `opm` built from this worktree (`task build`, so `version set` behaves as released). `opm module tidy --check` and `publish-templates.sh --dry-run` prove the release job will publish all three.

### D6. goreleaser `release.prerelease: auto`

```yaml
release:
  prerelease: auto
  extra_files:
    - glob: LICENSE
```

`auto` marks a tag with a SemVer prerelease suffix as Pre-release. goreleaser's default `release.mode` (`keep-existing`) leaves release-please's notes in place, as it does today. No PR workflow runs goreleaser and the recovery run checks out the tag, so a broken `.goreleaser.yml` would be unfixable for `v1.0.0-beta.1`: section 5 runs `goreleaser check` and a local `goreleaser release --snapshot --clean` (writes only `dist/`, never publishes) before the change can merge. If goreleaser cannot be installed locally, the task records that and the supervisor decides.

## Research & Decisions

### release-please and the line change
**Context**: whether the config flip alone reaches `1.0.0-beta.1`.
**Explored**: upstream `src/versioning-strategies/prerelease.ts` (bump on an existing prerelease increments its trailing number); cutover review runs against release-please 17.3.0 and 17.6.0 (final-paragraph `Release-As` parsed as RELEASE AS; a body line starting with `word(` drops the commit); cli precedent PR #96 and #110.
**Options considered**:
1. Config flip only: proposes `alpha.28`.
2. Config `release-as`: sticky, duplicate release PRs.
3. Hand-edit the manifest: bypasses release tooling (forbidden by the canon).
4. Footer on the carrier squash commit plus the config flip.
**Decision**: 4.
**Rationale**: the only one-shot mechanism that leaves release tooling owning every version file.

### Ceiling gate semantics
**Context**: counter reset makes the strict compare fragile (see D1).
**Explored**: `gates.go`, apply call site, operator `status.operatorVersion` publisher, the `cli-shipped-coupling` research (no other cli/operator version comparison exists).
**Decision**: MAJOR.MINOR (D1). **Rationale**: owner decision 3.

## Risks / Trade-offs

- [The footer is lost at merge (multi-commit GitHub squash bullets, or a body line starting with `word(`)] → The carrier message is prepared as the branch's final commit, and the supervisor parses it with release-please's parser before merging; the release PR title must read `chore(main): release 1.0.0-beta.1`, else fall back to `BEGIN_COMMIT_OVERRIDE` on the merged PR.
- [The alpha.28 release PR opened by PR-A gets merged by habit] → tasks.md names it as do-not-merge; only the supervisor merges.
- [Library beta shifts `schema.DefaultSchemaVersion()` to core beta while `hack/platform` is still on alpha.13 until PR-C merges] → Tests read the default dynamically; a skew warning is expected, a refusal is not. Any test failure is investigated, and a library behavior gap is fixed upstream, never forked in the cli.
- [goreleaser `auto` misread, or the snapshot differs from the real run] → `goreleaser check` plus a snapshot catches config errors; a failed real run burns `beta.1` (no hand tagging) and the target moves to `beta.2`.
- [The operator `v1.0.0-alpha.22` embed changes CRDs or RBAC the cli's install plan filters] → Review the `install.yaml` diff and run `go test ./internal/operator/...`; the diff is one reviewable commit.
- [Templates fail `opm module tidy --check` after the patch] → Run `opm module tidy` on the template and include the result in the same `fix(deps)` commit.
- [MAJOR.MINOR ceiling lets an operator patch that needs a newer cli through] → Accepted: by the shared-minor rule, a patch never requires a newer cli; a change that does belongs on a new minor.

## Migration Plan

Merge order (supervisor): PR-A any time; PR-C after G3; PR-B after G2 and G3, merged last with the footer. Then merge the retitled release PR, confirm G4 (Pre-release flag, five archives, `checksums.txt`, templates `1.0.3` on GHCR). Rollback is not a revert: a released tag is immutable, so a bad `beta.1` is fixed forward in `beta.2`.
