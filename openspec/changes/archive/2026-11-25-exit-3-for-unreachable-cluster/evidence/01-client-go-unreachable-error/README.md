# client-go error for an unreachable cluster

## Hypothesis

When the API server cannot be reached, client-go returns a transport error (`*url.Error`, a `net.Error`), not an API status, so `cmdutil.ExitCodeFromK8sError` returns 1 for it today.

## Setup

A standalone Go module, with its own `go.mod` so the cli's `go test ./...`, `go vet ./...` and `golangci-lint run ./...` do not load it. It pins `k8s.io/client-go v0.37.1` and the `k8s.io/apimachinery` version of the cli's `go.mod` at cli commit da27f30d. It imports nothing from the cli. It sends a dynamic-client GET of a `ModuleInstance` to three hosts: a refused port, a host name that does not resolve, and an address that does not answer (3 s timeout).

## Run

```bash
cd openspec/changes/exit-3-for-unreachable-cluster/evidence/01-client-go-unreachable-error
GOFLAGS=-mod=mod go run .
```

## Outcome

```text
https://127.0.0.1:1                 dial tcp 127.0.0.1:1: connect: connection refused
  type=*url.Error apiStatus=false timeout=false unavailable=false netErr=true
https://no-such-host.invalid:6443   dial tcp: lookup no-such-host.invalid: no such host
  type=*url.Error apiStatus=false timeout=false unavailable=false netErr=true
https://10.255.255.1:6443           context deadline exceeded
  type=*url.Error apiStatus=false timeout=false unavailable=false netErr=true
```

`timeout` and `unavailable` are `apierrors.IsServerTimeout` and `apierrors.IsServiceUnavailable`, the only branches that give exit 3 today. Hypothesis held. Every case is a `*url.Error` that matches `errors.As(err, &net.Error)` and no API status check.
