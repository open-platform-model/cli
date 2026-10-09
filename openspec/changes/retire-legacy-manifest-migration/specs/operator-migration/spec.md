## REMOVED Requirements

### Requirement: The proof list covers exactly the objects of earlier release manifests
**Reason**: The capability is retired. Nobody runs OPM yet, so no operator installed from a release manifest exists to take over (owner decision of 2026-10-05, confirmed 2026-10-09: "Remove both"). The enhancement withdraws 0012:D8:R6 and 0012:D8:R7. An existing object is judged by the ownership guard alone (capability `apply-pruning`, "Ownership guard judges every apply and dry run"; capability `operator-lifecycle`, "Install runs every refusing check before its first write").
**Migration**: None by the CLI. To let the operator instance take an existing object, annotate it `opmodel.dev/adopt=<instance UUID>` as the refusal prints, or delete the earlier install first.

### Requirement: An object is proven only by kind, name, labels and the absence of an instance identity
**Reason**: The capability is retired. Nobody runs OPM yet, so no operator installed from a release manifest exists to take over (owner decision of 2026-10-05, confirmed 2026-10-09: "Remove both"). The enhancement withdraws 0012:D8:R6 and 0012:D8:R7. An existing object is judged by the ownership guard alone (capability `apply-pruning`, "Ownership guard judges every apply and dry run"; capability `operator-lifecycle`, "Install runs every refusing check before its first write").
**Migration**: None by the CLI. To let the operator instance take an existing object, annotate it `opmodel.dev/adopt=<instance UUID>` as the refusal prints, or delete the earlier install first.

### Requirement: Install adopts the proven objects the module renders and no others
**Reason**: The capability is retired. Nobody runs OPM yet, so no operator installed from a release manifest exists to take over (owner decision of 2026-10-05, confirmed 2026-10-09: "Remove both"). The enhancement withdraws 0012:D8:R6 and 0012:D8:R7. An existing object is judged by the ownership guard alone (capability `apply-pruning`, "Ownership guard judges every apply and dry run"; capability `operator-lifecycle`, "Install runs every refusing check before its first write").
**Migration**: None by the CLI. To let the operator instance take an existing object, annotate it `opmodel.dev/adopt=<instance UUID>` as the refusal prints, or delete the earlier install first.

### Requirement: Labels and annotations of an earlier client-side or opm-cli apply do not survive install
**Reason**: The capability is retired. Nobody runs OPM yet, so no operator installed from a release manifest exists to take over (owner decision of 2026-10-05, confirmed 2026-10-09: "Remove both"). The enhancement withdraws 0012:D8:R6 and 0012:D8:R7. An existing object is judged by the ownership guard alone (capability `apply-pruning`, "Ownership guard judges every apply and dry run"; capability `operator-lifecycle`, "Install runs every refusing check before its first write").
**Migration**: None by the CLI. To let the operator instance take an existing object, annotate it `opmodel.dev/adopt=<instance UUID>` as the refusal prints, or delete the earlier install first.

### Requirement: Install recreates the earlier Deployment once
**Reason**: The capability is retired. Nobody runs OPM yet, so no operator installed from a release manifest exists to take over (owner decision of 2026-10-05, confirmed 2026-10-09: "Remove both"). The enhancement withdraws 0012:D8:R6 and 0012:D8:R7. An existing object is judged by the ownership guard alone (capability `apply-pruning`, "Ownership guard judges every apply and dry run"; capability `operator-lifecycle`, "Install runs every refusing check before its first write").
**Migration**: None by the CLI. To let the operator instance take an existing object, annotate it `opmodel.dev/adopt=<instance UUID>` as the refusal prints, or delete the earlier install first.

### Requirement: Install deletes the superseded role bindings
**Reason**: The capability is retired. Nobody runs OPM yet, so no operator installed from a release manifest exists to take over (owner decision of 2026-10-05, confirmed 2026-10-09: "Remove both"). The enhancement withdraws 0012:D8:R6 and 0012:D8:R7. An existing object is judged by the ownership guard alone (capability `apply-pruning`, "Ownership guard judges every apply and dry run"; capability `operator-lifecycle`, "Install runs every refusing check before its first write").
**Migration**: None by the CLI. To let the operator instance take an existing object, annotate it `opmodel.dev/adopt=<instance UUID>` as the refusal prints, or delete the earlier install first.

### Requirement: The migration deletes only proven objects, and refuses otherwise
**Reason**: The capability is retired. Nobody runs OPM yet, so no operator installed from a release manifest exists to take over (owner decision of 2026-10-05, confirmed 2026-10-09: "Remove both"). The enhancement withdraws 0012:D8:R6 and 0012:D8:R7. An existing object is judged by the ownership guard alone (capability `apply-pruning`, "Ownership guard judges every apply and dry run"; capability `operator-lifecycle`, "Install runs every refusing check before its first write").
**Migration**: None by the CLI. To let the operator instance take an existing object, annotate it `opmodel.dev/adopt=<instance UUID>` as the refusal prints, or delete the earlier install first.

### Requirement: Every refusing check runs before the migration's first write
**Reason**: The capability is retired. Nobody runs OPM yet, so no operator installed from a release manifest exists to take over (owner decision of 2026-10-05, confirmed 2026-10-09: "Remove both"). The enhancement withdraws 0012:D8:R6 and 0012:D8:R7. An existing object is judged by the ownership guard alone (capability `apply-pruning`, "Ownership guard judges every apply and dry run"; capability `operator-lifecycle`, "Install runs every refusing check before its first write").
**Migration**: None by the CLI. To let the operator instance take an existing object, annotate it `opmodel.dev/adopt=<instance UUID>` as the refusal prints, or delete the earlier install first.

### Requirement: Re-running install completes a migration that stopped partway
**Reason**: The capability is retired. Nobody runs OPM yet, so no operator installed from a release manifest exists to take over (owner decision of 2026-10-05, confirmed 2026-10-09: "Remove both"). The enhancement withdraws 0012:D8:R6 and 0012:D8:R7. An existing object is judged by the ownership guard alone (capability `apply-pruning`, "Ownership guard judges every apply and dry run"; capability `operator-lifecycle`, "Install runs every refusing check before its first write").
**Migration**: None by the CLI. To let the operator instance take an existing object, annotate it `opmodel.dev/adopt=<instance UUID>` as the refusal prints, or delete the earlier install first.

### Requirement: An operator applied from a module-rendered manifest needs no migration step
**Reason**: The capability is retired. Nobody runs OPM yet, so no operator installed from a release manifest exists to take over (owner decision of 2026-10-05, confirmed 2026-10-09: "Remove both"). The enhancement withdraws 0012:D8:R6 and 0012:D8:R7. An existing object is judged by the ownership guard alone (capability `apply-pruning`, "Ownership guard judges every apply and dry run"; capability `operator-lifecycle`, "Install runs every refusing check before its first write").
**Migration**: None by the CLI. To let the operator instance take an existing object, annotate it `opmodel.dev/adopt=<instance UUID>` as the refusal prints, or delete the earlier install first.

### Requirement: Install reports what the migration did and left
**Reason**: The capability is retired. Nobody runs OPM yet, so no operator installed from a release manifest exists to take over (owner decision of 2026-10-05, confirmed 2026-10-09: "Remove both"). The enhancement withdraws 0012:D8:R6 and 0012:D8:R7. An existing object is judged by the ownership guard alone (capability `apply-pruning`, "Ownership guard judges every apply and dry run"; capability `operator-lifecycle`, "Install runs every refusing check before its first write").
**Migration**: None by the CLI. To let the operator instance take an existing object, annotate it `opmodel.dev/adopt=<instance UUID>` as the refusal prints, or delete the earlier install first.

### Requirement: The CRDs-only form admits proven CRDs and makes no other migration write
**Reason**: The capability is retired. Nobody runs OPM yet, so no operator installed from a release manifest exists to take over (owner decision of 2026-10-05, confirmed 2026-10-09: "Remove both"). The enhancement withdraws 0012:D8:R6 and 0012:D8:R7. An existing object is judged by the ownership guard alone (capability `apply-pruning`, "Ownership guard judges every apply and dry run"; capability `operator-lifecycle`, "Install runs every refusing check before its first write").
**Migration**: None by the CLI. To let the operator instance take an existing object, annotate it `opmodel.dev/adopt=<instance UUID>` as the refusal prints, or delete the earlier install first.

### Requirement: The migration's deletes pass the shared delete verdict
**Reason**: The capability is retired. Nobody runs OPM yet, so no operator installed from a release manifest exists to take over (owner decision of 2026-10-05, confirmed 2026-10-09: "Remove both"). The enhancement withdraws 0012:D8:R6 and 0012:D8:R7. An existing object is judged by the ownership guard alone (capability `apply-pruning`, "Ownership guard judges every apply and dry run"; capability `operator-lifecycle`, "Install runs every refusing check before its first write").
**Migration**: None by the CLI. To let the operator instance take an existing object, annotate it `opmodel.dev/adopt=<instance UUID>` as the refusal prints, or delete the earlier install first.
