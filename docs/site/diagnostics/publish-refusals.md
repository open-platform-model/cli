---
title: "Publish refusals"
description: "Why publishing a module or catalog was refused, and how to fix it."
type: how-to
sidebar:
  order: 26
---

<!-- Diagnostics entry for every refusal the publish gates raise in `opm module publish` and `opm catalog publish`. Each gate below is one cause; the headline the CLI prints after "refused: " is quoted in each section so a reader can search for it. Every gate that can run does run, so one publish attempt lists every refusal at once; only a missing cue.mod file and a tree that does not load stop the run early. Check against: cli/internal/publish/gates.go, cli/internal/cmdutil/publish.go -->

## The message

<!-- The shape every refusal prints in. The plan goes to standard output first and ends in the verdict line "REFUSED — <count> refusal(s)". Each refusal then prints on standard error in one of two forms: a drafted refusal as "refused: <headline>" followed by aligned evidence rows (label, value, declaring file:line), a consequence paragraph and a runnable action; or a refusal carrying CUE's own error as "<headline>: <count> issue(s)" followed by CUE's grouped errors with their positions. Exit code 2 for refusals, 3 when the registry could not be reached (the plan then reads "INCOMPLETE", and nothing was judged). Verify: the log-level prefix the CLI's logger adds to each "refused:" line. Check against: cli/internal/publish/plan.go, cli/internal/publish/refusal.go, cli/internal/cmdutil/publish.go, cli/internal/cmdutil/output.go, cli/internal/exit/exit.go -->

```text
  REFUSED — <count> refusals
refused: <headline>

  <label>  <value>  <file:line>

  <consequence>

  <action>
```

## What it means

<!-- Two sentences at most. Publish reads the artifact's identity from `identity/identity.cue` and `cue.mod/module.cue`, derives the registry repository, major and tag from them, and refuses rather than push an artifact a consumer could not resolve or would resolve differently. Nothing is pushed while any refusal stands, and `--dry-run` runs every gate, including the registry lookups. Link the pages Publish a module, Publish a catalog and Registry namespaces. Check against: cli/internal/publish/gates.go, cli/internal/cmdutil/publish.go -->

## Causes and fixes

### There is no cue.mod file

<!-- Headline: "no cue.mod/module.cue under <dir> — this is not a CUE module". Both kinds; stops the run. Publish reads the module path from cue.mod and never invents one. Fix: the action names the command, `opm mod init <module-path>`, with the path read from the identity package when it loads. Check against: cli/internal/publish/gates.go -->

### The tree or its identity package does not load

<!-- Headlines: "the artifact does not load, so nothing about it can be judged" and "the artifact's identity package does not load, so its identity cannot be read". Both kinds; stops the run; the body is CUE's own error with positions. Fix: make the root package and the `./identity` package evaluate, then run `opm module vet` or re-run the publish with `--dry-run`. Check against: cli/internal/publish/load.go -->

### The cue.mod file cannot be read

<!-- Headline: "cue.mod/module.cue cannot be read". Both kinds; stops the run; carries the read or parse error. Fix: repair the file's syntax. Check against: cli/internal/publish/gates.go, cli/internal/publish/load.go -->

### The identity package does not conform to the schema

<!-- Headline: "the identity package does not conform to core's #IdentityPackage". Both kinds; CUE's error names the field. An open (unfilled) field is not this gate's finding; that is the empty identity field cause. Fix: correct the fields CUE names in `identity/identity.cue`. Check against: cli/internal/publish/identity.go, core/src/identity_package.cue -->

### The version flag disagrees with the declared version

<!-- Headline: "--version disagrees with the version this artifact declares". Evidence rows "--version" and "declared". `--version` fills an open `Version` or asserts a concrete one and never overwrites it. Fix: drop or correct the flag, or set the version first with `opm module version set <version>` (or `opm catalog version set`). Check against: cli/internal/publish/identity.go, cli/internal/cmdutil/publish.go -->

### An identity field has no value

<!-- Headline: "<path-or-dir> declares an identity field that has no value", with the evidence row naming `Version` or `ModulePath` and "declared, never filled". Version: supply it with `opm <kind> version set <version>` or `opm <kind> publish --version <version>` (the flag writes the value into `identity/identity.cue` before the push). ModulePath: declare it in `identity/identity.cue` as `"<host>/<path>@v<major>"`; `--version` does not fill it. Check against: cli/internal/publish/identity.go -->

### The kernel would refuse to load the module

<!-- Headline: "the kernel would refuse to load this module", evidence row "loader <error>". Modules only; skipped while an identity field is open or absent. Publish loads the tree through the same kernel acquire that `opm module build` and the operator use. The action depends on the loader's error: a missing required field ("Make the field a concrete literal in identity/identity.cue and reference it from metadata"), the wrong kind ("Publish the artifact with its own command: opm catalog publish for a catalog"), an invalid package ("exactly one CUE package at the module root"), otherwise "Fix the loader error above and re-run: opm module publish --dry-run". Check against: cli/internal/publish/kernel_gate.go, library/opm/errors/sentinels.go -->

### The cue.mod file does not declare its source

<!-- Headline: "cue.mod/module.cue declares no `source: {kind: \"self\"}`". Both kinds. CUE's module machinery reads `source` to decide which bytes form the artifact. Fix: `cue mod edit --source self`. Check against: cli/internal/publish/gates.go -->

### The declared path disagrees with cue.mod

<!-- Headline: "<declared-path> disagrees with itself about where it lives", evidence rows "declared" and "cue.mod" with file positions. Both kinds. Publish will not choose between the two. Fix: correct whichever is wrong, `ModulePath` in `identity/identity.cue` or the `module:` line in `cue.mod/module.cue`. Check against: cli/internal/publish/gates.go -->

### Metadata does not derive from the identity package

<!-- Headline: "<file> states a <modulePath-or-version> its identity package does not". Both kinds. `metadata.modulePath` and `metadata.version` must be the identity package's values; a literal in their place is refused even when it matches today. Fix: the action gives the line, `modulePath: id.ModulePath` or `version: id.Version`. Check against: cli/internal/publish/gates.go -->

### The tag falls outside the path's major

<!-- Headline: "the tag would not name a version within the major this artifact's path declares", evidence rows "tag" and "path major". Both kinds. Fix: publish a version within the path's major, or move `ModulePath` to the next major first. Link the concept page Versions in OPM. Check against: cli/internal/publish/gates.go -->

### The path does not fit the OPM namespace

<!-- Headline: "<repository> does not fit the namespace this domain publishes". Both kinds; applies only under opmodel.dev and community.opmodel.dev (opmodel.dev/core is exempt, testing.opmodel.dev is a separate domain, and any other domain is unconstrained). Allowed shapes: "opmodel.dev/(modules|catalogs|platforms|templates)/<name>" and "community.opmodel.dev/(m|catalogs|p)/<owner>/<name>". Fix: declare a path of that shape, or publish under a domain of your own. Link the reference page Registry namespaces. Check against: cli/internal/publish/gates.go -->

### The path's kind segment does not match the artifact

<!-- Headline: "a <kind> cannot publish under the <segment>/ segment". Both kinds, inside the OPM-owned domains only. A module publishes under modules/, m/ or templates/; a catalog under catalogs/. Fix: move the path under the right segment, or publish with the matching command. Check against: cli/internal/publish/gates.go -->

### The module's package name differs from its name

<!-- Headline: "<directory>'s package does not bind to its name", evidence rows "package" and "name". Modules only. A consumer's bare import binds the package name, so every consumer would have to alias the import. Fix: rename the root package to `metadata.name`, as the action shows (`package <name>`). Check against: cli/internal/publish/gates.go -->

### A local override file is present

<!-- Headlines: module, "cue.mod/local-module.cue is present; this module was validated against local checkouts, not against what a consumer will resolve"; catalog, "cue.mod/local-module.cue is present; a catalog cannot be published from a tree configured for local development". Evidence lists each replaced dependency and the version it would resolve to when published. CUE strips the file either way; the question is whether what was tested is what ships. Fix: remove the file and re-vet. Modules only: `--skip-override-check` publishes anyway; catalogs have no waiver. Check against: cli/internal/publish/gates.go, cli/internal/cmd/module/publish.go, cli/internal/cmd/catalog/publish.go -->

### A catalog member package does not load

<!-- Headline: "the <package-path> package does not load, so its members cannot be gated". Catalogs only; carries CUE's error. Fix: make the member package evaluate. Check against: cli/internal/publish/members.go -->

### A catalog member is off the catalog's key space

<!-- Headline: "<definition> (<package-path>) is off its catalog's key space". Catalogs only; every member of every kind, alpha included; CUE's error from unifying the member with core's #CatalogMemberFQNGate, checked concretely. Recognise the common cases: a member whose `metadata.fqn`, `metadata.modulePath` or `metadata.catalogVersion` does not follow from the catalog's identity and the member's filing (for example a member filed one directory too deep), or a resource, trait or blueprint that omits `metadata.apiVersion`. Fix: correct the member's metadata as the CUE error names. Link the reference page The catalog contract. Check against: cli/internal/publish/catalog_gates.go, core/src/identity_package.cue -->

### A trait does not state an overridable posture

<!-- Headline: "<definition> (<package-path>) does not state an overridable optional posture". Catalogs only; every trait. The trait's `optional` must be stated as a default (`bool | *true` or `bool | *false`): a trait that never mentions it fails as "incomplete value bool", and a fixed `true` or `false` fails because no module could narrow it. Fix: state the posture as a default. Link the page Write a trait. Check against: cli/internal/publish/catalog_gates.go, core/src/trait.cue -->

### The tag is already published

<!-- Headline: "<declared-path> already holds <tag>". Both kinds; the first registry lookup. A published tag names fixed bytes permanently and no flag overrides this. Fix: the action names the next patch, `opm <kind> version set <next-version>`. Check against: cli/internal/publish/registry.go -->

### The catalog would break a published contract

<!-- Headline: "<repository> would break a contract it already published", evidence "<definition> <apiVersion> compared against <repository>@<version>", then one row per violation: "field removed", "field added without optional or default", "default changed", "default removed", "domain narrowed" or "field made required". Catalogs only. At beta and GA a member may gain fields and options but never lose them or move a default. Not judged: alpha apiVersions, transformers, dev tags, and beta or GA members while the catalog's own version is a release prerelease. Fix: make the change additive, or ship the changed member alongside at the next apiVersion the action names. `opm catalog registry check <path@version> --compat` runs the same comparison on a published build. Check against: cli/internal/publish/compat.go, cli/internal/compat/compat.go, cli/internal/publish/check.go -->

## Where it is raised

<!-- Badge: publish. `opm module publish [path]` and `opm catalog publish [path]`, with `--dry-run` to run every gate without pushing. `opm module vet` runs a subset (identity load and conformance, an empty ModulePath, the tag's major, cue.mod agreement, derivation and the kernel load), prints the same refusals and exits with code 2; it does not refuse an open Version. `opm catalog registry check` reports the identity and compatibility findings on a published build in the same shape, framed as findings, not refusals. Check against: cli/internal/cmd/module/publish.go, cli/internal/cmd/catalog/publish.go, cli/internal/publish/vet.go, cli/internal/cmd/module/vet.go, cli/internal/cmd/catalog/registry.go -->
