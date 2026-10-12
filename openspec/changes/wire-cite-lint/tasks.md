## 1. Pin

- [ ] 1.1 Move the five `.github` references to `2b66da51fede4332969c324b069012632d8edea9 # .github main` and copy `wiring-check.sh` from that SHA; `task cascade:wiring:check` passes.

## 2. Cite lint

- [ ] 2.1 Add `.github/workflows/cite-lint.yml` with one plain call job and the exclude `tests/fixtures/`; declare it in `wiring-check.yaml`.
- [ ] 2.2 Add `task cite-lint`; `task cite-lint` finds 0 problems.

## 3. Verify

- [ ] 3.1 `openspec validate wire-cite-lint --strict`, actionlint, zizmor, `task cascade:wiring:check`, `task fmt`, `task vet`, `task lint`, `task test:unit`.
