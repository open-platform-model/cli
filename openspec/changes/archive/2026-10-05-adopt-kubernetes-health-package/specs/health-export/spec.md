## REMOVED Requirements

### Requirement: Health status type is exported
**Reason**: The cli no longer declares a health status type. Readiness is the library's `opm/k8s/health`, whose `Status` type and constants carry the same strings, and a frontend that adopts a tier package keeps no copy and no alias (0012:D3:R6).
**Migration**: Use `health.Status` and `health.Ready`, `health.NotReady`, `health.Complete`, `health.Unknown`, `health.Missing`, `health.Applied`, `health.Bound` from `github.com/open-platform-model/library/opm/k8s/health`. The cli's rule is `health-evaluation`.

### Requirement: EvaluateHealth is exported
**Reason**: The evaluator is deleted; `health.Evaluate` in the library holds the same rules.
**Migration**: Call `health.Evaluate`.

### Requirement: QuickInstanceHealth aggregates health from pre-fetched resources
**Reason**: The fold is the library's `health.Aggregate`, which takes the evaluated statuses and the count of tracked objects that have none.
**Migration**: Call `health.Aggregate` over `health.Evaluate` of each live object, with the missing plus unreadable count.

### Requirement: IsHealthy helper function
**Reason**: The healthy set is the library's `health.IsHealthy`, with the same four statuses.
**Migration**: Call `health.IsHealthy`.
