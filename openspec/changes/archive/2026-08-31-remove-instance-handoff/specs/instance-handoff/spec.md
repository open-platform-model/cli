## REMOVED Requirements

### Requirement: Handoff command exists and is forward-only
**Reason**: `opm instance handoff` is removed. The flip is irreversible and the command never verified that the operator's effective applier identity could apply the instance; the feature will be redesigned separately.
**Migration**: No CLI path from CLI ownership to operator ownership exists. Create operator-owned instances outside the CLI (kubectl, GitOps); the CLI edits them through the thin-editor apply path.

### Requirement: Precondition chain runs in order and aborts before the flip
**Reason**: Part of the removed command.
**Migration**: None; the gates guarded a flip that no longer exists.

### Requirement: Verification render is strict-registry and self-compared
**Reason**: Part of the removed command. `status.lastAppliedRenderDigest` keeps being recorded on apply so a future transfer has a value to verify against.
**Migration**: None.

### Requirement: Ownership flip changes only the owner
**Reason**: Part of the removed command.
**Migration**: An instance's owner is set by whoever creates it; the CLI writes `cli` on apply and never rewrites it.

### Requirement: Post-flip wait judges an inventory-stable reconcile
**Reason**: Part of the removed command.
**Migration**: None.
