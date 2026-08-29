## Context

See proposal.md. `tests/e2e/operator_test.go` has `requireKindCluster(t)` (skips without a kubeconfig or reachable `kind-opm-dev`) and shells out to `kubectl` for cluster assertions; the handoff tests in `instance_handoff_test.go` call it first. The operator's manager ClusterRole grants `impersonate` on ServiceAccounts, users and groups and nothing on workload kinds; `hack/kind-operator-rbac.yaml` adds a dev-only `opm-operator-dev-applier` ClusterRole and binding.

## Goals / Non-Goals

**Goals:** a wrong-prepared cluster fails in under a second with the remedy in the message.
**Non-Goals:** auto-applying the grant from the tests (a test that mutates cluster RBAC hides the preparation step it is meant to expose); probing every verb the operator might need (two representative verbs identify the missing grant).

## Decisions

**D1. Probe with `kubectl auth can-i <verb> <resource> -n <ns> --as=<sa>`.** It is the cluster's own SubjectAccessReview, needs no client-go plumbing in the test package, and matches how the rest of the e2e file talks to the cluster. Alternative (client-go `SelfSubjectAccessReview` with impersonation headers): more code for the same answer; rejected.

**D2. Fail, do not skip.** A missing grant on a reachable cluster is a preparation mistake, not an absent environment; skipping would hide it in CI-like runs. Unreachable cluster keeps the skip.

**D3. Call sites: the four handoff-family tests only.** `TestE2E_Handoff_Adoption`, `_DigestGate`, `_PreconditionRefusals`, `TestE2E_ThinEditor_ValuesRoundTrip`, `TestE2E_Delete_OperatorOwnedDelegates`; the lifecycle test installs its own operator and applies no workloads.

## Risks / Trade-offs

- [The probe's verbs drift from what the operator needs] → the probe checks the two kinds every fixture renders; if a fixture adds a kind the dev grant lacks, the product error still appears, and the grant file is the place to extend.
