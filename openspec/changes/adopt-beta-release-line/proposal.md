## Why

OPM is cutting its prerelease lines from alpha to beta (core `2.0.0-beta.1`; library, cli, opm-operator and catalogs/k8s `1.0.0-beta.1`). The cli must move with them: it links the library (whose embedded core decides the default schema), ships the official templates, embeds an opm-operator release, and publishes GitHub releases that catalog_opm and the module fleets install. Flipping `prerelease-type` alone does nothing, because release-please increments an existing `-alpha.N` and would propose `1.0.0-alpha.28`; the cli needs a one-shot `Release-As: 1.0.0-beta.1` carrier plus the config flip so later cuts count `beta.2`, `beta.3`, and so on.

The cutover also exposes a real defect. The operator-version ceiling gate compares full semver, prerelease counter included. Counters restart at `beta.1` in both repos, so any operator release that lands before the next cli release (docs commits cut operator releases) would make `opm module apply` refuse against the cluster. The owner's decision is that cli and operator share MAJOR.MINOR and release patches and prerelease counters independently, so the gate compares MAJOR.MINOR only.

The owner should see the consequence plainly: release-please keeps every beta release at `1.0.0-beta.N` until GA, so for the whole beta the relaxed ceiling refuses nothing between a 1.0 cli and a 1.0 operator. It revises enhancement 0006 D24's full-semver ceiling. During beta, skew safety rests on the CRD field floor plus a process rule (design.md D1): an operator `feat!` that needs a newer cli ships only after that cli release, or moves both to a new minor. Final ceiling semantics are the canon GA-exit item "ceiling gate semantics final (OQ14)".

## What Changes

- **Operator-version ceiling compares MAJOR.MINOR only.** `GateOperatorVersionCeiling` refuses only when the operator's MAJOR.MINOR is above the cli's; patch and prerelease are ignored (operator `1.0.0-beta.3` against cli `1.0.0-beta.2` passes; operator `1.1.0` against cli `1.0.0` still refuses). Relaxes a refusal, never adds one.
- **Library `v1.0.0-beta.1`.** `go.mod` pins the first library beta explicitly, so the cli's default core (`schema.DefaultSchemaVersion()`) becomes the core beta.
- **Embedded operator refreshed to `v1.0.0-alpha.22`** (three releases stale today). The operator beta embed follows in a later plain `fix(deps)` commit once the operator beta exists; the cli beta must ship first because of the ceiling gate.
- **Templates re-pinned to core `v2.0.0-beta.1` and catalogs/opm `v4.4.4`, identity bumped to `1.0.3`** in all three templates, so the release job actually publishes them (it skips versions already on GHCR, and `opm module init` scaffolds exactly what the template pins).
- **Release line flipped to beta.** `release-please-config.json` `prerelease-type: beta`; `.goreleaser.yml` `release.prerelease: auto`, so the GitHub release is flagged Pre-release like core, library and the operator, instead of goreleaser clearing the flag release-please set. Latest stays `v0.6.0` until GA (a GA-exit item); this change does not move it.
- **Help and doc examples** that name alpha versions move to beta or current tags (`operator:sync` usage, `opm operator install --version`, `opm catalog registry check`, the publish-a-catalog check note, AGENTS.md).
- **Beta promise and template rule written down.** AGENTS.md states the cli's beta promise (a break during beta is a `feat!` with a `BREAKING CHANGE:` migration footer that advances `1.0.0-beta.N`; GA drops the suffix) and that re-pinning `templates/*` needs `opm module version set` on each touched template; `openspec/config.yaml` gets a matching beta note under Principle VI so proposals stop declaring a MAJOR release-please will never cut during beta.
- **Fixture pins** (`hack/platform`, `hack/kind-platform.yaml`, `examples/`) follow in a companion `test(fixtures)` PR, never mixed with the shipped bumps.

SemVer class: PATCH. The version itself is forced by the carrier footer (`1.0.0-alpha.27` to `1.0.0-beta.1`).

Delivery: one PR per section (PR-A = section 1; PR-C = section 6 on `beta/cli-fixture-pins`; PR-B = sections 2 to 5 and 7, the carrier), because the relaxed ceiling must be on main before any operator beta, the carrier needs the published library and catalog betas, and fixture pins never share a squash commit with shipped bumps. Spec sync waits for the archive in PR-B.

Section count: seven, against the proposal rule's "at most about five" (`openspec/config.yaml` also allows "about seven" in the tasks rule, so the config conflicts with itself). Seven is kept because each section is a distinct gate or PR boundary (G2 library, the operator embed, G1+G3 templates, release config, the separate fixture branch, the archive and carrier); folding section 3 into section 2 would save one heading but mix an ungated embed into the G2-gated go.mod section.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `apply-preflight-gates`: the operator-version ceiling compares MAJOR.MINOR only.
- `release-workflow`: the release line is `beta`, a forced line change travels as a one-shot `Release-As` footer, the `workflow_dispatch` recovery input is specified (the spec still says there is none), and goreleaser flags prerelease tags as GitHub Pre-releases.

## Impact

- Commands: `opm module apply` and `opm instance apply` (ceiling gate), `opm operator install` (embedded operator version), `opm module init` (templates it fetches), `opm catalog registry check` and `opm operator install` help text.
- Packages: `internal/inventory` (gate + tests), `internal/operator` (`manifest.go`, `dist/install.yaml`), `templates/*`, `go.mod`/`go.sum`.
- Release tooling: `release-please-config.json`, `.goreleaser.yml`; no workflow file changes.
- Downstream: catalog_opm, modules, opm-modules and opm-operator CI re-pin the cli beta tag (supervisor task after G4, outside this change). The published templates move to `1.0.3`.
- No enhancement decision is implemented (the cutover implements no 0021 decision), so there is no `enhancement.yaml`.
