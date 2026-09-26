---
title: "Publish a module"
description: "Publish a module to an OCI registry so others can install it."
type: how-to
sidebar:
  order: 24
---

<!-- One sentence: `opm module publish` pushes a module to an OCI registry at the path and version the module itself declares, after a set of checks that refuse anything a consumer could not load. Use it when a module is ready for others to import or install.
Say early that there is no destination flag: the path in `identity/identity.cue` decides where it goes, and the version decides the tag. A published tag is permanent; publish refuses to overwrite it and the CLI has no command that removes one.
Check against: cli/internal/cmd/module/publish.go, cli/internal/publish/publish.go (package comment), cli/internal/publish/registry.go (gateAlreadyPublished) -->

## Before you begin

<!-- By title: a module that passes `opm module vet`, as in "Your first module" and "Define a module's configuration". An OCI registry you can push to. `cue.mod/module.cue` declaring `source: {kind: "self"}`, which modules scaffolded by `opm mod init` already carry. Verify: which registries have been exercised end to end (GHCR is the one the first-party pipeline uses).
Check against: opm/docs/site/authoring/your-first-module.md, core/docs/site/authoring/define-module-configuration.md, cli/internal/publish/gates.go (gateSourceSelf), cli/templates/minimal/cue.mod/module.cue -->

## Steps

1. Route the module's path to your registry.

   <!-- The module path is `ModulePath` in `identity/identity.cue`, equal to `module:` in `cue.mod/module.cue`. Map its domain to your registry in the registry setting, which the CLI reads from `--registry`, then `OPM_REGISTRY`, then `registry` in `~/.opm/config.cue`, in CUE registry syntax: `example.com=ghcr.io/<owner>,opmodel.dev=ghcr.io/open-platform-model,registry.cue.works`. Keep the `opmodel.dev` entry so core and the catalogs still resolve.
   - If the path is under `opmodel.dev` or `community.opmodel.dev`, it must fit the exact shape for that domain (`opmodel.dev/(modules|catalogs|platforms|templates)/<name>`, `community.opmodel.dev/(m|catalogs|p)/<owner>/<name>`) and sit under a module segment; any other domain is unconstrained. See "Registry namespaces".
   Check against: cli/internal/config/resolver.go (ResolveRegistry), cli/internal/config/templates.go (DefaultRegistry), cli/internal/publish/gates.go (firstPartyShape, communityShape, gateNamespace, gateKindSegment) -->

2. Log in to the registry.

   <!-- `opm registry login <host>`. It prompts for a username and secret, checks them against the registry, and only then writes the standard docker credential file; a rejected credential leaves the file untouched. Append `+insecure` to a host served over plain HTTP. Without a host it uses the host the registry setting resolves to, and refuses when that names several.
   - If publishing from CI, use `docker login` instead: the command is interactive, and both write the same file.
   Check against: cli/internal/cmd/registry/login.go -->

3. Set the version you are publishing.

   <!-- `opm module version set <version>`: bare SemVer, no `v`. It rewrites only the `Version` literal in `identity/identity.cue`, offline. Its major must match the path's: a path ending `@v0` publishes `0.x.y`. Commit the change, so the version sits in history before the artifact does. A module scaffolded by `opm mod init` starts at 0.1.0.
   - If `Version` has no value, `opm module publish --version <version>` fills it and writes `identity/identity.cue` before pushing. If it has one, `--version` only asserts it, and a different value is refused.
   Check against: cli/internal/cmd/module/version.go, cli/internal/publish/identity.go (resolveVersion), cli/internal/scaffold/scaffold.go (InitialVersion) -->

4. Remove local dependency overrides.

   <!-- If `cue.mod/local-module.cue` exists (`replaceWith` entries pointing at local checkouts), publish refuses: the published module resolves its dependencies from the registry whatever the file says, so what was tested is not what ships. Remove the file.
   - If the substitution does not matter, publish anyway with `--skip-override-check`; the replacements are ignored either way.
   Check against: cli/internal/publish/gates.go (gateOverride) -->

5. Run a dry run.

   <!-- `opm module publish --dry-run` in the module directory, or with its path. Every check runs, including the registry lookup for an existing tag; nothing is pushed.
   Output to describe: the plan rows (kind, declaredPath, cueModPath, registryRepo, major, tag, tag from, one identity row for ModulePath and one for Version, local override, kernel loader), then one verdict: `GO — pushing <registryRepo>:<tag>`, `REFUSED — <n> refusal(s)` with each refusal printed below, or `INCOMPLETE` when the registry could not be reached. Exit codes: 0 go, 2 refused, 3 registry unreachable.
   The checks, all run in one pass so one fix round covers them: cue.mod exists; the tree and its identity package load; identity conforms to core's `#IdentityPackage`; `Version` has a value; `--version` agrees with it; the kernel's module loader accepts the tree; cue.mod declares `source: {kind: "self"}`; cue.mod's `module:` equals `ModulePath`; `metadata.modulePath` and `metadata.version` derive from the identity package; the tag's major equals the path's; the namespace shape and module segment on owned domains; the root package name equals `metadata.name`; no `local-module.cue`; the tag is not already published. Each refusal shows its evidence and a runnable fix. Point at "Publish refusals" for each message.
   Check against: cli/internal/publish/gates.go (Run), cli/internal/publish/plan.go (Render), cli/internal/publish/kernel_gate.go, cli/internal/publish/identity.go, cli/internal/cmdutil/publish.go (RunPublish) -->

6. Publish.

   <!-- `opm module publish`. The same plan prints, then `Published <registryRepo>:<tag>`.
   What gets pushed is the module directory as it is on disk, zipped by CUE's module machinery. Verify: the command help says "the committed tree", but cli/internal/publish has no git check, so uncommitted edits would be published; say which is true.
   Check against: cli/internal/publish/registry.go (Push), cli/internal/cmdutil/publish.go -->

## Check that it worked

<!-- Command: `opm module publish --dry-run` again.
Success: the plan now refuses with `<declaredPath> already holds <tag>` and the action `Bump and retry:  opm module version set <next patch>`, which confirms the registry serves the tag. Verify whether the author prefers a consumer-side check instead, such as an instance file that imports the module path and `opm instance build`.
Check against: cli/internal/publish/registry.go (gateAlreadyPublished, nextPatch), cli/examples/instances/podinfo/instance.cue -->

## Related

<!-- By title: the reference entries "CLI reference" (`opm module publish`, `opm module version set`, `opm registry login`) and "Registry namespaces", the diagnostics entry "Publish refusals", and the concept page "Versions in OPM".
Check against: opmodel.dev/site/content/docs/reference/cli/index.md, cli/docs/site/reference/registry-namespaces.md, cli/docs/site/diagnostics/publish-refusals.md, core/docs/site/concepts/versions.md -->
