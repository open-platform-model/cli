# Tasks: repair-cascade-test

One PR, titled `test(cascade): build the older library from the tree's own`.

**Local gate:** `CASCADE_TEST_SET=all task -x deps:cascade:test`, `task fmt`, `task vet`, `task lint`, `task openspec:check`, `task cascade:wiring:check`.

## 1. The setup

- [ ] 1.1 `test.sh`: make the older library from the tree's library (file proxy in `$TMP`, version `v<X.Y.Z>-0.cascade.<suffix>`), and use it in `setup_older` with the scoped Go environment of design.md.
- [ ] 1.2 `test.sh`: `setup_older` sets `SETUP_WHY` (step and last output lines); S2, S4 and S9 print it.
- [ ] 1.3 `testdata/older.tsv`: drop the library row; update the header comment.
- [ ] 1.4 `test.sh`: move the `older.tsv` check to the pre-checks of both sets; the library is checked through its made-up version.
- [ ] 1.5 Run the full set with `CASCADE_RESOLVER_REAL` set to the resolver at the pinned `.github` commit, and the offline set; both exit 0. Show a stale row failing the offline set, then restore it.
- [ ] 1.6 Run the gates; commit.
