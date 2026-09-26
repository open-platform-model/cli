---
title: "Publish a catalog"
description: "Publish a catalog so platforms can subscribe to it."
type: how-to
sidebar:
  order: 22
---

<!-- One sentence: opm catalog publish pushes a catalog to an OCI registry at the path and version the catalog itself declares, after running every publish gate, so that a platform can import it. The reader needs it once their catalog's resources, traits and transformers pass their own checks. Say that publish pushes the committed directory exactly, with no build step and no version rewriting, and that a published tag can never be replaced. Check against: cli/internal/cmd/catalog/publish.go, cli/internal/publish/publish.go, cli/internal/publish/registry.go -->

## Before you begin

<!-- The reader must already have: a catalog module whose members vet cleanly (see "Write a trait" and "Write a transformer"); the opm CLI installed (latest cli release is v1.0.0-alpha.21; Verify at writing); push access to the OCI registry their module path maps to. No cluster is needed. Check against: cli/.release-please-manifest.json, cli/internal/cmd/catalog/catalog.go -->

## Steps

1. Check the catalog's identity.
   <!-- Three files must agree. cue.mod/module.cue: module "<host>/catalogs/<name>@v<major>" and source: {kind: "self"}. identity/identity.cue: ModulePath equal to that module line, Version as bare SemVer whose major matches the path's, plus RegistryPath and kindPrefix as core's #IdentityPackage defines them. catalog.cue: metadata.modulePath: id.ModulePath and metadata.version: id.Version, derived and never literal. If the path is under opmodel.dev it must be opmodel.dev/catalogs/<name>; under community.opmodel.dev it must be community.opmodel.dev/catalogs/<owner>/<name>; any other domain is unconstrained. Check against: core/src/identity_package.cue, catalog_opm/opm/identity/identity.cue, catalog_opm/opm/cue.mod/module.cue, cli/internal/publish/gates.go (gateNamespace, gateKindSegment, gateDerivation) -->
2. Point opm at your registry.
   <!-- The registry mapping uses CUE_REGISTRY syntax (<module-prefix>=<host>/<repo>, comma-separated, +insecure for plain HTTP). Precedence: the --registry flag, then OPM_REGISTRY, then config.registry in ~/.opm/config.cue (opm config init writes a default). Verify: when none of the three is set, the registry client receives an empty mapping and CUE's own CUE_REGISTRY handling applies. Check against: cli/internal/config/resolver.go, cli/internal/cmd/root.go, cli/internal/config/templates.go, cli/internal/publish/registry.go (NewRegistryClient) -->
3. Log in to the registry.
   <!-- opm registry login [host] checks the credential against the registry and stores it in the Docker credential file that push and pull both read. It is interactive; in CI, use docker login, which writes the same file. Check against: cli/README.md (Registry Operations), cli/internal/cmd/registry/login.go -->
4. Set the version to publish.
   <!-- opm catalog version set <version> [path] rewrites only the Version value in identity/identity.cue and works offline; commit the change, so a commit sits between choosing a version and pushing it. If release automation writes the version, pass --version <version> to publish instead: it fills an open Version or asserts the declared one, and never overwrites it. Check against: cli/internal/cmd/catalog/version.go, cli/internal/cmdutil/publish.go (PublishFlags), cli/internal/publish/identity.go (resolveVersion) -->
5. Remove local overrides.
   <!-- Delete cue.mod/local-module.cue if it exists. Catalog publish always refuses a tree that has one, and unlike module publish it has no --skip-override-check: a catalog's dependency choice reaches every module built against it. Check against: cli/internal/publish/gates.go (gateOverride), cli/internal/cmd/catalog/publish.go -->
6. Run the gates without pushing.
   <!-- opm catalog publish <path> --dry-run. Show the plan block: kind, declaredPath, cueModPath, registryRepo, major, tag, tag from, one identity row per field, local override, member gate, posture gate, compat gate, then the verdict (GO, REFUSED with a count, or INCOMPLETE when the registry could not be reached). Exit codes: 0 for GO, 2 when refused, 3 when the registry is unreachable. List what the gates check, in plain words: the tree is a CUE module, identity conforms, cue.mod and identity agree, metadata derives from identity, the tag's major matches the path, namespace and kind segment, no local override, every member's fqn and modulePath sit in the catalog's key space, every trait states an overridable posture, the tag is not already published, and no beta or GA member changed incompatibly since the last published build. Check against: cli/internal/publish/gates.go (Run), cli/internal/publish/plan.go (Render), cli/internal/publish/catalog_gates.go, cli/internal/publish/compat.go -->
7. Fix every refusal.
   <!-- Each refusal prints its evidence and a runnable action. Name the common ones by headline: "<path> already holds <tag>" (run opm catalog version set <next>); "<file> states a modulePath its identity package does not" (derive it: modulePath: id.ModulePath); "cue.mod/module.cue declares no source: {kind: "self"}" (cue mod edit --source self); "<Definition> (<package>) does not state an overridable optional posture"; "<Definition> (<package>) is off its catalog's key space"; "<path> would break a contract it already published" (make the change additive, or ship it at the next apiVersion). Alpha members and -dev builds are exempt from the compatibility gate. Point to the diagnostics entry "Publish refusals" for the full list rather than repeating it. Check against: cli/internal/publish/registry.go, cli/internal/publish/gates.go, cli/internal/publish/catalog_gates.go, cli/internal/publish/compat.go (compatRefusal, isDevTag) -->
8. Publish the catalog.
   <!-- opm catalog publish <path>. The plan prints again, then "Published <registryRepo>:<tag>". If --version filled an open Version, the fill is written into identity/identity.cue before the push. If the catalog is one of the first-party catalogs in catalog_opm, never publish by hand: its CI publishes releases and branch builds. Check against: cli/internal/cmdutil/publish.go (RunPublish), cli/internal/publish/registry.go (Push), catalog_opm/AGENTS.md (Release & publishing) -->

## Check that it worked

<!-- One command: opm catalog registry check <registryRepo>@<version>, for example example.com/catalogs/demo@v1.2.0 (the path without its major, and the "v" is optional). It pulls the build and checks it the way a platform does when it imports it: the declared identity is concrete and agrees with the coordinate. It lists the catalog's members by kind and apiVersion. Exit 0 means clean. --compat also reruns the compatibility comparison. Say that this check is an aid; publish is where the rules are enforced. Check against: cli/internal/cmd/catalog/registry.go, cli/internal/publish/check.go -->

## Related

<!-- Reference: the "CLI Reference" entries for opm catalog publish, opm catalog version set and opm catalog registry check, and "Registry Namespaces". Concept: "Platforms and catalogs", which covers how a platform subscribes to the published catalog, and "Versions in OPM". Diagnostics: "Publish refusals". -->
