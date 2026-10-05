## MODIFIED Requirements

### Requirement: Render digests are kernel-derived and operator-parity

`status.lastAppliedRenderDigest` SHALL be computed over the kernel's rendered objects with the library's shared render digest, `opm/k8s/inventory.RenderDigest`, which the operator also uses. The digest leaves the value of the `app.kubernetes.io/managed-by` label out, so the CLI (`opm-cli`) and the operator (`opm-controller`) digest the same render to the same value, while any other difference in an object, another label included, changes it (0012:D6:R2/R3). A registry-gated integration check SHALL verify that the CLI's local-dir staging path and the operator's registry-acquisition call sequence, both ending in `Kernel.Render` against the same platform module, produce identical render digests for the same instance (0006 D30 gate). Evaluator-version skew reporting applies to a future cross-binary comparison, where the CLI and operator binaries embed separate CUE evaluators; the in-binary check renders both paths with one evaluator and cannot exhibit skew.

#### Scenario: Parity for the same inputs

- **WHEN** the parity check renders a fixture instance via the CLI kernel path and via the operator's call sequence against the same platform module
- **THEN** the two render digests SHALL be identical

#### Scenario: Skew reported explicitly (cross-binary check, slice C3)

- **WHEN** the future cross-binary parity comparison runs while the `cli` and `opm-operator` binaries embed different `cuelang.org/go` minor versions
- **THEN** the check SHALL fail with a message naming the evaluator-version skew as the suspected cause

#### Scenario: The runtime name does not move the digest

- **WHEN** two renders produce the same objects except that one stamps `app.kubernetes.io/managed-by: opm-cli` and the other `opm-controller`
- **THEN** their render digests SHALL be identical
