## REMOVED Requirements

### Requirement: A moved operator module pin on a release PR needs e2e evidence
**Reason**: Interim gate. The cluster-backed CI check "E2E (kind, embedded operator)" runs the whole e2e suite against the pinned operator module on every release-please PR, has passed on release PRs and is required by the `main` ruleset, which are the three retirement conditions in workspace RELEASING.md, section "Gates".
**Migration**: None. Release PRs no longer need the `e2e-verified` label; the required CI check gates them.
