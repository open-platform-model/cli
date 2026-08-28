## 1. templates/

- [x] 1.1 In `templates/{minimal,standard,advanced}/identity/identity.cue`: `Version: "1.0.0"`, remove the `#VersionType` declaration and its comment and the `// x-release-please-version` marker.
- [x] 1.2 In each template's `module.cue`: `version: id.Version`, remove the interpolation comment.
- [x] 1.3 `opm module version set 1.0.1 ./templates/<t>` for all three (proves the literal writer; bumps for republish).

## 2. internal/scaffold

- [x] 2.1 `repair.go` `identityFileContent`: emit `Version: %q`; drop the `#VersionType` block and comment.
- [x] 2.2 `scaffold.go` `Reidentify`: call `cueedit.SetIdentityVersion(dir, InitialVersion)`.
- [x] 2.3 Flip `repair_test.go` and `internal/cmd/module/init_test.go` expectations to the literal form.

## 3. internal/cueedit

- [x] 3.1 Delete `ResetIdentityVersion` and `spliceVersionReset`; delete their tests; keep every `SetIdentityVersion` tolerance test (literal, `&` chain, defaulted).

## 4. tests/e2e

- [x] 4.1 `mod_init_test.go`: assert `Version: "0.1.0"` and absence of `*"` / `#VersionType` in the scaffold's identity file.
- [x] 4.2 Add an assertion that `opm module build` on the scaffold passes the loader shape gate (renders or fails past the gate).

## 5. Validation gates

- [x] 5.1 `task fmt`, `task lint`
- [x] 5.2 `task test`
- [x] 5.3 `.github/scripts/publish-templates.sh --dry-run` reports GO for all three templates at `v1.0.1`.
