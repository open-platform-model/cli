## 1. Spec deltas

- [x] 1.1 Write the three MODIFIED requirements and the new scenarios in `specs/template-modules/spec.md`
- [x] 1.2 `openspec validate fix-stale-specs --strict` passes
- [x] 1.3 Commit docs(openspec): propose fix-stale-specs

## 2. Verify

- [x] 2.1 Each new statement traced to code by path and line (design.md); no code change (`git diff base -- . ':!openspec'` is empty)
- [x] 2.2 `task fmt`, `task vet`, `task lint` and `task test:unit` green
- [x] 2.3 `openspec validate --all --strict --no-interactive` and `enh cite-lint` report 0 findings
- [ ] 2.4 Review and fixes; commit docs(openspec): follow the review of fix-stale-specs where it changes files

## 3. Archive

- [ ] 3.1 Archive the change; the sync writes the deltas into `openspec/specs/template-modules/spec.md`
- [ ] 3.2 Commit docs(openspec): archive fix-stale-specs
