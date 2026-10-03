## Context

The instance render commands resolve the Kubernetes namespace once, through `config.ResolveKubernetes` with the command's `-n` value as `NamespaceFlag`: flag, then `OPM_NAMESPACE`, then `kubernetes.namespace` in `~/.opm/config.cue`, then `default`. The result is a `config.ResolvedField` whose `Source` records which one won (`internal/config/resolver.go:138-152`). All four commands then call `render.FromInstanceFile` (`internal/cmd/instance/{apply,build,diff,vet}.go`), which acquires the instance package through the kernel (`internal/workflow/render/render.go:72`), resolves the platform, renders, and finally overwrites the namespace on the result when the source is flag or env (`render.go:259-266`).

The module path (`render.FromModule`, `internal/workflow/render/module.go`) synthesizes the instance with `syntheticIdentity`, which already takes the namespace from flag or env (`module.go:206-225`). There the override and `metadata.namespace` agree by construction.

## Goals / Non-Goals

**Goals:**

- Refuse an instance-file render whose flag or env namespace disagrees with the file's `metadata.namespace`, before anything reaches the cluster or the registry beyond the instance acquire.
- Say in the `-n` help text of the instance render commands that the flag must match the file.

**Non-Goals:**

- Re-synthesizing the instance in the override namespace (the "honour" alternative, rejected below).
- Changing the by-file query commands (`status`, `tree`, `events`, `delete`), where `-n` still overrides the file's namespace for the lookup.
- Changing the module path.
- Removing the override assignment at `render.go:259-266`. After this change it writes the value the instance already has on both paths, but deleting it is not part of the decision.

## Decisions

### Where the guard runs

The guard SHALL run in `FromInstanceFile`, immediately after `AcquireInstanceFromDir` returns, and before `instanceDepsOf` and `resolvePlatformEnv`. At that point `inst.Metadata.Namespace` is the kernel's decode of the file, and the existing comment "cheap failures never hit the cluster or registry" holds: platform resolution is the next step. `instance apply` and `instance diff` construct their Kubernetes client before calling `FromInstanceFile`; the guard does not move that, so on those two commands a missing kubeconfig is still reported before the mismatch.

Putting the guard in `FromInstanceFile` rather than in `renderInstance` keeps the module path out of it structurally: `FromModule` never calls it.

```go
// refuseNamespaceOverride returns a validation error when the namespace was
// set by --namespace or OPM_NAMESPACE and differs from the namespace the
// instance file declares. The file owns the namespace: it is part of the
// instance's identity (fqn and uuid), so an override would put the record
// in one namespace and the resources in another. A namespace from the
// config file or the default is not an override and is never compared.
func refuseNamespaceOverride(instancePath, declared string, ns config.ResolvedField) error
```

Called as:

```go
if inst.Metadata != nil {
    if err := refuseNamespaceOverride(opts.InstanceFilePath, inst.Metadata.Namespace, opts.K8sConfig.Namespace); err != nil {
        printValidationError(err)
        return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
    }
}
```

`inst.Metadata` is nil only when the decode failed, and the acquire already refuses a package that is not a valid instance; with no declared namespace there is nothing to compare.

### Exit code and message

Exit code 2 (`ExitValidationError`), printed through `printValidationError` like the other input refusals in `FromInstanceFile`. The input is well-formed but inconsistent, which is a validation failure, not a usage error.

`refuseNamespaceOverride` returns a `*pkgerrors.ValidationError` whose `Message` is the one-line disagreement (the override's source, both namespaces, the instance argument as given) and whose `Details` is the guidance to edit `metadata.namespace`. `printValidationError` falls through to `cmdutil.PrintValidationError`, which prints a `ValidationError` with details as `render failed: <message>` followed by a details block, the same header every other `FromInstanceFile` refusal uses. A plain error would instead print as an escaped `error=` field under the header. The message carries no enhancement or ADR reference (CLI output reaches people without the enhancements repo).

```
$ opm instance apply ./jellyfin -n staging
ERROR render failed: --namespace "staging" disagrees with metadata.namespace "media" in ./jellyfin
  the namespace is part of the instance's identity: to deploy to "staging", set metadata.namespace: "staging" in the instance file (that makes a new instance; delete the one in "media" if it is deployed); otherwise drop the override
exit 2

$ OPM_NAMESPACE=staging opm instance build ./jellyfin
ERROR render failed: OPM_NAMESPACE "staging" disagrees with metadata.namespace "media" in ./jellyfin
  ...same guidance...
exit 2
```

The exact wording is fixed in implementation; the spec pins the parts (source, both values, edit `metadata.namespace`).

When both `-n` and `OPM_NAMESPACE` are set, the flag wins resolution and the message names `--namespace`; the env value is shadowed and not compared.

### Help text

`opm instance apply`, `build`, `diff` and `vet` each register their own `-n` with the text "Target namespace". It becomes:

```
Namespace; must equal the instance file's metadata.namespace
```

`instance vet`'s example `opm instance vet ./jellyfin_instance.cue -n production` is dropped, since it would now be refused for any file not in `production`. `task docs:reference` regenerates `docs/site/reference/cli/opm-instance.md`; the cmdref check fails CI otherwise.

`instance vet` is not named in the owner decision, but it calls the same `FromInstanceFile`, so it refuses too. Exempting it would take extra code and would make vet pass an invocation that apply refuses. Its help text changes with its behaviour.

## Research & Decisions

### Refuse or honour

**Context**: The override and the file disagree; one must win consistently.
**Explored**: `research.json` task i5 (2026-10-02), verified against `render.go`, `module.go`, `instance_arg.go`, `instance_target.go` and core `module_instance.cue`.
**Options considered**:
1. Refuse - the file stays the single owner of the namespace (library ADR-001: `#ModuleInstance`, named `#ModuleRelease` in the ADR, owns the authoritative deployed namespace); one guard and a message; by-file query commands keep finding what apply wrote.
2. Honour - re-synthesize the instance in the override namespace so resources, fqn, uuid and record move together. The same file would then name different instances depending on the shell environment, and `instance status f` would still look in the file's namespace.
**Decision**: Refuse (owner, 2026-10-03, i5).
**Rationale**: Namespace is identity. A file that means one instance regardless of environment is the simpler contract.

### Fixture that relied on the split

**Context**: `tests/integration/skip-unprovided/main.go:181` runs `instance apply` on `internal/workflow/render/testdata/skip-unprovided/instance` with `-n opm-skip-unprovided-itest`, while the fixture declares `namespace: "default"`. The program then reads the record from the test namespace (`:206`, `:212`) and deletes that namespace on cleanup.
**Options considered**:
1. Set the fixture's `metadata.namespace` to `opm-skip-unprovided-itest` and keep `-n` - the program exercises the matching-flag case against a real cluster, and its resources finally land in the namespace it cleans up.
2. Drop `-n` and use `default` - the cleanup would then delete the `default` namespace.
**Decision**: Option 1. The only other users of that instance directory are `TestSkipUnprovided_BothEntryPoints` and the new tests here, which pass an empty or explicit namespace and do not depend on `default`.

## Risks / Trade-offs

- [A script that set `OPM_NAMESPACE` globally and ran `instance apply` across files in other namespaces now fails] -> Intended: each of those applies wrote a split instance. The message says how to fix the file or drop the override.
- [The integration program is not in CI] -> Section 1 runs it locally against `kind-opm-dev`.
