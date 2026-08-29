## Context

See proposal.md. `tests/e2e/operator_test.go` has `requireKindCluster(t)` (skips without a kubeconfig or reachable `kind-opm-dev`) and shells out to `kubectl` for cluster assertions; the handoff tests in `instance_handoff_test.go` call it first. The operator's manager ClusterRole grants `impersonate` on ServiceAccounts, users and groups and nothing on workload kinds; `hack/kind-operator-rbac.yaml` adds a dev-only `opm-operator-dev-applier` ClusterRole and binding.

## Goals / Non-Goals

**Goals:** a wrong-prepared cluster fails in under a second with the remedy in the message.
**Non-Goals:** auto-applying the grant from the tests (a test that mutates cluster RBAC hides the preparation step it is meant to expose); probing every verb the operator might need (two representative verbs identify the missing grant).

## Decisions

**D1. Probe with `kubectl auth can-i <verb> <resource> -n <ns> --as=<sa>`.** It is the cluster's own SubjectAccessReview, needs no client-go plumbing in the test package, and matches how the rest of the e2e file talks to the cluster. Alternative (client-go `SelfSubjectAccessReview` with impersonation headers): more code for the same answer; rejected. `kubectl auth can-i` exits 1 both for a denial and for a failed check, so the helper reads stdout: `no` is a denial (fail, D2); `yes` passes; anything else (no output, an error on stderr) is "could not check" and follows the reachability rule (skip with the kubectl error in the message). Without that split, a user lacking `impersonate` would see a skip where a fail was meant, or a fail where a skip was meant.

**D1a. The identity probed is the operator's effective applier, not a constant.** The helper reads the controller Deployment's container args: with `--default-service-account=<sa>` present, it probes `system:serviceaccount:default:<sa>` (the flag resolves the SA in the instance's namespace, which for the fixtures is `default`); otherwise it probes the controller's own SA, `system:serviceaccount:opm-operator-system:opm-operator-controller-manager`. The failure message names whichever identity it probed. Hardcoding the controller SA would report a false denial the day the dev operator runs with the flag.

**D2. Fail, do not skip.** A missing grant on a reachable cluster is a preparation mistake, not an absent environment; skipping would hide it in CI-like runs. Unreachable cluster keeps the skip.

**D3. Call sites: the five handoff-family tests only, through `requireReconcilingOperator`.** `TestE2E_Handoff_Adoption`, `_DigestGate`, `_PreconditionRefusals`, `TestE2E_ThinEditor_ValuesRoundTrip`, `TestE2E_Delete_OperatorOwnedDelegates` already call `requireReconcilingOperator`; the probe is added inside it after the replica check rather than as a second helper each test must remember to call. The lifecycle test does not call it, installs its own operator, and applies no workloads.

## Risks / Trade-offs

- [The probe's verbs drift from what the operator needs] → the probe checks the two kinds every fixture renders; if a fixture adds a kind the dev grant lacks, the product error still appears, and the grant file is the place to extend.
- [The probe entrenches a dev grant no real cluster has, and the suite keeps bypassing impersonation] → accepted for this change and stated in the proposal; the enhancement that replaces the grant with the production pattern retires this probe.
- [The could-not-check branch (neither `yes` nor `no`) is unexercised by the suite] → it is three lines, its neighbours are exercised live, and exercising it automatically requires the test to create cluster RBAC (a non-goal); verified once by hand with a ServiceAccount token that lacks `impersonate`.
