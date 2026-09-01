## 1. Baseline layer A: CLI stderr

Runs before any code changes. The point is a recorded, re-runnable snapshot of
what the CLI prints today for bad `#config` values and a bad `#components`
block, so the same script re-run after implementation shows exactly which
diagnostics moved. Expected diff after implementation: the values-file cases
gain per-file positions; the `#components` cases are unchanged byte-for-byte
(the change does not touch that path, and a diff there is a regression).

- [ ] 1.1 Add the layered-values fixtures under `tests/e2e/testdata/vet-errors/layered/`,
      reusing the existing `module/` fixture's `#config` (`media?: [Name=string]: {type: "pvc" | *"emptyDir"}`):
      `base.cue` (valid), `conflict.cue` (sets `media.cache.type: "pvc"` where `base.cue`
      sets `"emptyDir"`: a cross-file concrete conflict), `disallowed.cue` (a top-level
      field `#config` does not declare), `abstract.cue` (leaves a field non-concrete).
      These fixtures are shipped: section 5's e2e assertions consume them.
- [ ] 1.2 Add a broken-`#components` fixture module at `tests/e2e/testdata/vet-errors/components/`
      (own `cue.mod`, identity package, valid `#config`) with three variants in separate
      files so each can be selected on its own: an unknown trait/resource reference, a
      wrong-typed field against the catalog schema, and a component missing a required field.
- [ ] 1.3 Add `openspec/changes/cli-layered-values/baseline/capture.sh`: builds `opm` via
      `task build`, then runs the matrix below with `HOME` pointed at a throwaway dir
      holding the dummy `config.cue` (same setup as `tests/e2e/mod_init_test.go`'s
      `TestMain`), writing each case's stderr plus its exit code to
      `baseline/<case>.txt`. `OUT_DIR` defaults to `baseline/`, so the post-change re-run
      is `OUT_DIR=after ./capture.sh`. Absolute paths are rewritten to `<testdata>/` so
      the captures diff cleanly across machines.
- [ ] 1.4 The matrix (no cluster required; `module apply` / `instance apply` are out of
      scope for the baseline because they need a cluster, and they share the
      `unifyValuesFiles` call site with `module build`):
      | case | command |
      | --- | --- |
      | `vet-single-disallowed` | `opm module vet <module> -f layered/disallowed.cue` |
      | `vet-two-file-conflict` | `opm module vet <module> -f layered/base.cue -f layered/conflict.cue` |
      | `vet-two-file-disallowed` | `opm module vet <module> -f layered/base.cue -f layered/disallowed.cue` |
      | `vet-non-concrete` | `opm module vet <module> -f layered/abstract.cue` |
      | `build-two-file-conflict` | `opm module build <module> -f layered/base.cue -f layered/conflict.cue` |
      | `instance-vet-conflict` | `opm instance vet instance/instance.cue -f layered/base.cue -f layered/conflict.cue` |
      | `components-unknown-ref` | `opm module build components/ --variant unknown-ref` |
      | `components-wrong-type` | `opm module build components/ --variant wrong-type` |
      | `components-missing-field` | `opm module build components/ --variant missing-field` |
      (the `--variant` column is shorthand: select the variant by which file is present
      in the build dir, not by a new flag , no flag is added by this change.)
- [ ] 1.5 Run `./capture.sh`, commit the captures, and read them. Record in
      `baseline/NOTES.md`, per case: exit code, whether the output is grouped or
      flattened, whether any file path appears at all, and which file the reported
      position names when two files contribute. This is the observation that justifies
      the change; if a case already reports the right file, say so.

## 2. Baseline layer B: Go test output

The CLI captures in section 1 show the rendered block only. This layer records
the two things underneath it, so a post-change diff says *where* an output moved:
the raw `cue/errors` tree (message, position, path per leaf) and the printer that
formats it. It also pins the current test inventory, so a test deleted alongside
`unifyValuesFiles` shows up as a diff instead of vanishing quietly.

- [ ] 2.1 Add `baseline/gotest.sh` (same `OUT_DIR` convention as `capture.sh`):
      runs `go test -count=1 -v` over the packages this change touches
      `./internal/workflow/render/...`, `./internal/cmd/module/...`,
      `./internal/cmdutil/...`, `./pkg/validate/...`, `./pkg/loader/...`, writing
      one file per package to `$OUT_DIR/gotest/<pkg>.txt`. Normalize before writing:
      strip elapsed times (`(0.00s)` -> `(Xs)`), absolute paths, and `ok ... 1.234s`
      durations, so a diff only shows test names, pass/fail, and printed output.
- [ ] 2.2 In the same script, record the test inventory: `go test -list '.*'` per
      package into `$OUT_DIR/gotest/inventory.txt`. The four `unifyValuesFiles` tests
      in `internal/workflow/render/render_test.go` (lines 109, 120, 134, 148) are
      expected to disappear here; nothing else should.
- [ ] 2.3 Add `internal/workflow/render/values_baseline_test.go`: a table-driven test
      over the section 1.1 fixtures that, for each case, loads the values files the way
      the CLI does today and prints every leaf of the resulting `cue/errors` tree as
      `path | message | position`. Positions are printed relative to `testdata/`. The
      test always passes; its `-v` output is the artifact. This is the layer that
      actually proves per-file attribution landed: today the position column is expected
      to name the merged value or a single file, not the contributing file.
- [ ] 2.4 Add `internal/cmdutil/print_validation_baseline_test.go`: feed the same error
      trees through the real `PrintValidationError` with stderr captured, and print the
      rendered block. This separates "the error tree changed" from "the formatting
      changed" when section 5 diffs them.
- [ ] 2.5 Run `./gotest.sh`, commit `baseline/gotest/`, and extend `baseline/NOTES.md`
      with what 2.3 shows per case: which file each position names, and whether two
      contributing files produce one leaf or two.

## 3. Design and specs

- [ ] 3.1 Write `design.md` using the section 1 captures as the "current error output"
      evidence and the required post-change output as the target, including the exact
      expected text for `vet-two-file-conflict`.
- [ ] 3.2 Write the `specs/` deltas for `kernel-render`, `module-synthetic-instance` and
      `mod-vet` named in the proposal. `openspec validate cli-layered-values --strict` passes.

## 4. Implementation

- [ ] 4.1 Deferred to design. Do not start before sections 1 and 2 are committed:
      without both baselines there is nothing to compare the new output against.

## 5. Verify the diff is the intended one

- [ ] 5.1 `OUT_DIR=after ./capture.sh && diff -ru baseline after`. Every
      `components-*` case MUST be byte-identical. Every values case MUST differ only by
      added file/line attribution: same exit code, same grouped shape, same message text.
- [ ] 5.2 `OUT_DIR=after ./gotest.sh && diff -ru baseline/gotest after/gotest`.
      Expected diffs, and nothing else: the four `unifyValuesFiles` tests gone from the
      inventory, the 2.3 position column naming the contributing file per case, and any
      new test added by section 4. An unexplained diff in `pkg/validate` or
      `internal/cmdutil` means the change reached further than intended.
- [ ] 5.3 Extend `tests/e2e/vet_output_test.go` with the two-file conflict and two-file
      disallowed cases, asserting each contributing file's basename appears in the grouped
      output. Keep the existing flattened-shape anti-regression assertions.
- [ ] 5.4 `task lint` and `task test` pass.
