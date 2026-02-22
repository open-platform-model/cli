## ADDED Requirements

### Requirement: Shared inventory resolution helper in cmdutil

The `cmdutil` package SHALL provide a `ResolveInventory` function that encapsulates
the full inventory lookup-and-discover flow used by `mod delete` and `mod status`.

The function SHALL accept:
- A context
- A Kubernetes client
- A `*ReleaseSelectorFlags` (carrying release name and/or release ID)
- A namespace string
- An `ignoreNotFound bool` flag
- A structured logger scoped to the release

The function SHALL return the resolved `*inventory.InventorySecret`, the discovered
live `[]*core.Resource`, and an error.

The function MUST implement the following resolution logic:
- If `rsf.ReleaseID` is non-empty: resolve via `inventory.GetInventory` using the
  release ID. If `rsf.ReleaseName` is also set, use it as the display name; otherwise
  use the release ID as the display name.
- If `rsf.ReleaseName` is non-empty (and no ReleaseID): resolve via
  `inventory.FindInventoryByReleaseName`.
- If inventory lookup fails: log the error and return an `*ExitError` with code
  `ExitGeneralError`.
- If the inventory Secret is not found and `ignoreNotFound` is true: log info and
  return `nil, nil, nil`.
- If the inventory Secret is not found and `ignoreNotFound` is false: log the error
  and return an `*ExitError` with code `ExitNotFound`.
- After resolving the Secret: call `inventory.DiscoverResourcesFromInventory` to fetch
  live resources. If this fails: log the error and return an `*ExitError` with code
  `ExitGeneralError`.

#### Scenario: Resolution by release name succeeds

- **WHEN** `ReleaseSelectorFlags.ReleaseName` is set and the inventory Secret exists
- **THEN** `ResolveInventory` returns the Secret and its discovered live resources with no error

#### Scenario: Resolution by release ID succeeds

- **WHEN** `ReleaseSelectorFlags.ReleaseID` is set and the inventory Secret exists
- **THEN** `ResolveInventory` returns the Secret and its discovered live resources with no error

#### Scenario: Release not found with ignoreNotFound false

- **WHEN** the inventory Secret does not exist and `ignoreNotFound` is false
- **THEN** `ResolveInventory` returns an `*ExitError` with code `ExitNotFound`

#### Scenario: Release not found with ignoreNotFound true

- **WHEN** the inventory Secret does not exist and `ignoreNotFound` is true
- **THEN** `ResolveInventory` returns `nil, nil, nil` (callers treat this as a no-op)

#### Scenario: Kubernetes error during inventory lookup

- **WHEN** `inventory.GetInventory` or `inventory.FindInventoryByReleaseName` returns
  a non-nil error
- **THEN** `ResolveInventory` logs the error and returns an `*ExitError` with code
  `ExitGeneralError`

#### Scenario: Resource discovery fails

- **WHEN** the inventory Secret is found but `DiscoverResourcesFromInventory` returns
  an error
- **THEN** `ResolveInventory` logs the error and returns an `*ExitError` with code
  `ExitGeneralError`
