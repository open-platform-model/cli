---
title: Registry Namespaces
description: "Every module path prefix OPM owns, who may publish there, and where each rule is enforced."
type: reference
sidebar:
  order: 6
---

OPM artifacts live at CUE module paths, and the path's prefix decides who publishes there and what the tooling asserts about it. This page is the reference for every prefix OPM owns: what each means, who may publish to it, and where the rule is enforced. Rules are cited to the enhancement decisions that define them.

## The principle: hosts, not indexes

The registry model asserts things **only about domains OPM owns** (`0011 D13`). Your artifacts under your own domain (`example.com/modules/billing@v1`, `my.corp/catalogs/internal@v2`) are entirely yours: `opm module publish` and `opm catalog publish` run their identity and content gates but assert nothing about your namespace's layout. There is no central index to register with; a registry is a host, and the declared module path *is* the address.

## Owned prefixes

| Prefix | What it is | Published by | Enforcement |
| --- | --- | --- | --- |
| `opmodel.dev/core` | The OPM core schema module (current line: major v2) | `core` repo release CI | Release automation; the module path is the address (`0010 D1`) |
| `opmodel.dev/catalogs/<name>` | First-party catalogs, currently the single consolidated `opm` catalog (`0010 D47`) | `catalog_opm` repo release CI via `opm catalog publish` | The full catalog gate set on every publish ([The Catalog Contract](/docs/reference/catalog-contract/)) |
| `opmodel.dev/modules/<name>` | First-party modules. **Flat**: one snake-case leaf, no nesting | Module release pipelines via `opm module publish` | The publish namespace gate refuses nested paths and non-snake leaves (`0011 D13`) |
| `opmodel.dev/templates/<name>` | Official module templates, the modules `opm mod init` clones | The `cli` repo's release CI, exclusively | Reserved segment (`0011 D25`); the name `index` is reserved within it; publish gates admit the segment as module-kind |
| `opmodel.dev/platforms/…` | **Reserved, unpublished.** No platform is published or fetched from a registry today | Nobody | A namespace claim, not a publisher (`0011 D14`) |
| `testing.opmodel.dev/*` | Fixtures and experiments, never production artifacts. Current occupants include the operator's test-module fleet (`testing.opmodel.dev/modules/operator/*`) | Owning repos' CI | Convention plus the gates' identity checks; nothing here carries a compatibility promise |
| `community.opmodel.dev/m/<owner>/<name>` | Community modules under per-owner paths | The owning community publisher | Identity gates; the owner segment is the namespace (`0011 D13`) |
| Anything else | Yours | You | Identity and content gates only, no namespace assertions |

## Why bare-word template shortcuts are safe

`opm mod init standard` expands the bare word into `opmodel.dev/templates/standard`, and only there (`0011 D25`). The expansion is syntactic (a bare word can never be a valid module path, which requires a dotted first segment) and its target is the one segment nothing but the cli's own release pipeline can publish to, through the full gate set. A typo fails inside the reserved segment with the expansion named; it can never resolve to something somebody squatted.

## Two prefixes that are deliberately *not* what they look like

- **`testing.opmodel.dev` is not a staging environment.** Artifacts there are test inputs for OPM's own repositories; they may be deleted, renamed, or republished without notice.
- **`opmodel.dev/platforms` is not coming soon.** It is reserved precisely so that nobody claims it informally; whether platforms are ever registry-published is an open design question, not a roadmap item.

## See also

- [The Catalog Contract](/docs/reference/catalog-contract/): what a catalog publish promises and where it is enforced.
