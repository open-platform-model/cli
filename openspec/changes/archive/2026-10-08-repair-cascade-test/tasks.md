# Tasks: repair-cascade-test

One PR, titled `test(cascade): build the older library from the tree's own`.

**Local gate:** `CASCADE_TEST_SET=all task -x deps:cascade:test`, `task fmt`, `task vet`, `task lint`, `task openspec:check`, `task cascade:wiring:check`.

## 1. The setup

- [x] 1.1 `test.sh`: make the older library from the tree's library (file proxy in `$TMP`, version `v0.0.0-0.cascade.<tree version>`), and use it in `setup_older` with the scoped Go environment of design.md.
- [x] 1.2 `test.sh`: `setup_older` sets `SETUP_WHY` (step and last output lines); S2, S4 and S9 print it.
- [x] 1.3 `testdata/older.tsv`: drop the library row; update the header comment.
- [x] 1.4 `test.sh`: move the `older.tsv` check to the pre-checks of both sets; the library is checked through its made-up version.
- [x] 1.5 Run the full set with `CASCADE_RESOLVER_REAL` set to the resolver at the pinned `.github` commit, and the offline set; both exit 0. Show a stale row failing the offline set, then restore it.
- [x] 1.6 Run the gates; commit.

## 2. Review fixes

- [x] 2.1 `lib_older_name`: a `v0.0.0-0.cascade.<tree version>` name, below pseudo-version pins too; the library gets its own FAIL text.
- [x] 2.2 Record the deviation from contract section 8 in design.md and in `test.sh`; name the remaining catalog and core risk of S9; correct the spec sentence on the `oldest` rows and the import timeline.
- [x] 2.3 Run the gates again; commit.

## 3. Owner's decision and archive

- [x] 3.1 Record the owner's decision of 2026-10-09 (the departure from contract section 8 is allowed for the library key) in design.md, proposal.md and the `test.sh` comment.
- [x] 3.2 Sync the delta into `openspec/specs/deps-cascade/spec.md`, run the gates on the final tree, archive.
