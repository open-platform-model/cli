---
title: "Contracts with no provider"
description: "Why a render refused a provider-fulfilled trait or resource, and how --skip-unprovided renders the rest."
type: how-to
weight: 27
---

<!-- Diagnostics entry for the render refusal a provider-fulfilled contract with no provider on the platform raises, in every render-bearing command (`opm module build`, `vet`, `apply`; `opm instance build`, `vet`, `diff`, `apply`), and for the `--skip-unprovided` flag that renders the rest. The rule is the kernel's (core SPEC §2.1 and §3.1); the CLI passes the switch and words the rows. Check against: cli/internal/workflow/render/validation.go, cli/internal/workflow/render/skipped.go, cli/internal/cmdutil/flags.go -->

## The message

<!-- The kernel's refusal names the component, the demanded contract key and, for a provider-fulfilled contract nothing on the platform provides, the tail "provider-fulfilled, no provider on this platform". The CLI prints the diagnostics rows and then the hint below, whatever the platform source (`--platform`, the cluster Platform, or the render's own deps). Exit code 2. Check against: library/opm/errors/match.go (UnresolvedDemand.describe), cli/internal/workflow/render/validation.go (unprovidedHint) -->

```text
ERRO render failed: 1 unresolved demand(s):
  component "db": unresolved trait demand "opmodel.dev/catalogs/opm/traits/backup@v1alpha1": defined by "opmodel.dev/catalogs/opm@v4" and nothing on this platform implements it; provider-fulfilled, no provider on this platform

Hint: a provider-fulfilled contract has no provider on this platform: install one, pass --platform <dir> with a platform that carries one, or pass --skip-unprovided to render the rest
```

## What it means

<!-- Two sentences at most. A provider-fulfilled contract (`fulfilment: "provider"`, such as the `backup` trait of opmodel.dev/catalogs/opm@v4) is implemented by a platform's provider, never by the catalog that defines it, so it renders only on a platform that carries one; a platform generated from a module's own deps never does, and a cluster without the OPM operator registers none. Link Publish a module for the deps platform. -->

## Causes and fixes

### Install a provider, or render against a platform that has one

<!-- The fix that renders the contract. On a cluster with the OPM operator, a provider registers through a TransformerRegistration; `kubectl get transformerregistrations -A` lists them. Off the cluster, pass `--platform <dir>` with a platform module that carries a provider. Check against: cli/internal/platform/resolve.go -->

### Render the rest with --skip-unprovided

<!-- Pass `--skip-unprovided` (bool, default false) to render everything else. Only demands the kernel marks unprovided are skipped: a catalog-fulfilled gap, a provider that exists but did not match, and an over-subscribed contract still refuse. A skipped trait produces nothing and its component renders every other object; a skipped resource drops its whole component, which is not reported as unmatched. Every skip is printed as a warning on standard error, so `build` output on standard output stays a clean manifest. A row the kernel reports same-base alternatives for appends "; implemented at: <keys>". Check against: cli/internal/workflow/render/skipped.go -->

```text
WARN component "db": skipped provider-fulfilled trait "opmodel.dev/catalogs/opm/traits/backup@v1alpha1" (no provider on this platform)
WARN component "archive" not rendered: provider-fulfilled resource "example.dev/catalogs/k8up/resources/backup-store@v1alpha1" has no provider on this platform
```

### The skipped-contracts annotation

<!-- `opm instance apply` and `opm module apply` record every skip on the ModuleInstance as `module-instance.opmodel.dev/skipped-contracts`: sorted, deduplicated `<component>=<fqn>` pairs joined with commas. An apply that skips nothing, with or without the flag, removes it. It is information for whoever inspects the instance; no gate reads it. Check against: cli/internal/inventory/store.go (specAnnotations), cli/internal/workflow/apply/apply.go (SkippedContracts) -->

```text
module-instance.opmodel.dev/skipped-contracts: db=opmodel.dev/catalogs/opm/traits/backup@v1alpha1
```

### An operator-managed instance refuses the flag

<!-- The operator renders an instance it manages and never skips, so an apply with `--skip-unprovided` to such an instance is refused before any spec is written, exit code 2. Install a provider instead. Check against: cli/internal/workflow/apply/thineditor.go -->

```text
--skip-unprovided has no effect on instance "hello": the opm-operator renders it and does not skip provider-fulfilled contracts. Install a provider for the contract instead
```
