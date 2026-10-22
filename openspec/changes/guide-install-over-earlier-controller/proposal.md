## Why

A cluster can still hold the install from before the operator was renamed to the controller. The new `opm controller install` does not migrate it. It refuses with exit 2, because the four CustomResourceDefinitions (`moduleinstances.opmodel.dev`, `modulepackages.opmodel.dev`, `platforms.opmodel.dev`, `transformerregistrations.opmodel.dev`) keep their names and the earlier install owns them. The earlier controller keeps running. The refusal names the `opmodel.dev/adopt` annotation, and nothing tells the user why the objects exist or that a second controller would run beside the new one.

This change adds one hint line to that refusal and one docs page. The page says how to remove the earlier controller by hand before `opm controller install`.

## What Changes

- `opm controller install` prints one `hint:` line after a refusal of the apply guard when the cluster also holds the earlier controller Deployment. The line points to the docs page. The refusal, its exit code 2 and the objects it refuses do not change.
- A new page `docs/site/diagnostics/install-over-the-earlier-controller.md` says what happened, how to remove the earlier controller, what to do with the four CRDs, and that `opm controller uninstall` deletes nothing over a manifest install.

Affected commands and packages: `opm controller install` (both the full form and `--crds-only`), `internal/cmd/controller`, `internal/controller`, `docs/site/diagnostics`.

Release class: PATCH after GA, and `beta.N+1` in beta. It adds one line to an existing error and one page. Nothing is breaking: no flag, exit code, output field or refusal changes.

Commit type and PR title for the implementing PR: `fix(controller): point an install refused over the earlier controller to its docs page`.

Delivery: one PR per change, the default.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `controller-lifecycle`: adds the requirements for the hint line and for the docs page.

## Enhancement

None.

## Impact

- Code: one read of one Deployment on the refusal path, one wrapping error type, tests on the fake cluster.
- Docs: one page. Its front matter and structure follow `legacy-inventory-secret.md` in the same folder.
- Old names: the hint and the page name the earlier objects by their old names, so the old names appear in two files (listed in `design.md`).
