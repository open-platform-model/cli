## Why

`cue.mod/local-module.cue` is the sanctioned local-development mechanism (0006 D37), and every CLI load path honours it today because the developer's directory is the main module of that load. The render is the exception: the library kernel generates its own main module and, on library releases up to `v1.0.0-alpha.28`, promotes only each input's `cue.mod/module.cue`, so a developer's redirect of a shared module, of a never-published module, or of the catalog is silently dropped at render and the published pin resolves instead. The CLI's D19 warning ("demanded keys may not correspond to published bytes") therefore describes a render that does not happen.

The library change `render-local-replacements` (library repo, same name) promotes those redirects into the render module under the platform-wins precedence of 0019 D13, behind a per-render opt-in on `RenderInput`, refuses an input carrying the file when the opt-in is off, and reports every honoured replacement as a diagnostics row. This change makes the CLI opt in and word the rows.

## What Changes

- **Opt in.** The CLI's single `Kernel.Render` call (`internal/workflow/render/render.go`, `renderInstance`) sets the kernel's local-replacements opt-in for every render-bearing command: `module render/build/vet` paths that render, `instance build/apply/diff`. The operator is untouched and never opts in.
- **Word the rows.** The D19 warning is rebuilt from data instead of a file test. For each replacement row the kernel honoured: one warning naming the replaced path, the directory it was served from and which input (module or platform) supplied it, ending with the existing "may not correspond to any published build" caveat. For each `replaceWith` in the module context's own `cue.mod/local-module.cue` that no row honours: one warning saying the platform names that path and the redirect belongs in the platform module's `cue.mod/local-module.cue`. A clean context stays silent.
- **Provenance unchanged.** The `SourceLocal` signal and the `module-instance.opmodel.dev/source: local` annotation keep their present rule (file present with a local-path `replaceWith`), which the library's rows do not alter.
- **Reader shared.** `pkg/loader` gains a small reader of a module root's `local-module.cue` replacements; `HasLocalModuleReplacement` is reimplemented on it. The publish gate's own reader (which also needs the published pin) is left as is.
- **Library bump.** `go.mod` moves to the library release carrying the opt-in; a kernel refusal naming `cue.mod/local-module.cue` (malformed file, mismatched module path in the target directory) surfaces through the existing validation-error path.

**Not in this change:** the library behaviour itself; new flags (none: a developer who wrote the file wants it honoured, and the publish gates already refuse or warn before anything leaves the machine); public site documentation of the workflow (`opmodel.dev`, separate repo); any operator work.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `kernel-render`: the "Local-replacement renders warn (0010 D19)" requirement changes from a file-presence warning to: the render opts in, honoured replacements are warned from the kernel's rows, and an inert module-side replacement is warned as such.

## Impact

**SemVer:** MINOR. No flag, no command syntax change; a render now evaluates a developer's replaced bytes where it previously evaluated the published pin, and the warnings say which.

**Packages:** `internal/workflow/render` (one field at the render call, warning construction, tests), `pkg/loader` (replacement reader, provenance reimplemented on it), `go.mod`. Commands affected: every render-bearing command, no syntax change.

**Depends on:** the library release that ships `render-local-replacements` (after `v1.0.0-alpha.28`). Cannot land before the bump: with the new library and no opt-in, every render of a directory carrying the file refuses.

**Complexity justification (Principle VII):** one boolean at the call site and one warning builder over a slice the kernel already sorts; the reader replaces an ad-hoc `CompileBytes` parse that exists today.
