## 1. Rules

- [ ] 1.1 `openspec/config.yaml`: add `references:`, the `Delivers` rule, the `context` line and the comment rule; remove the `NNNN:Dn` bullets and the `enhancement.yaml` and `delivery:log` rules
- [ ] 1.2 `AGENTS.md`: same text
- [ ] 1.3 `task openspec:check` green, then commit docs(openspec): point the cli rules at the enhancements store

## 2. The rewrite by tool

- [ ] 2.1 Run `enh cite-rewrite --write --worklist --exclude tests/fixtures/` once
- [ ] 2.2 Spot-read at least 40 edits; fix wrong ones by hand
- [ ] 2.3 `task fmt`, `task lint`, `task test` green, then commit one `docs:` or `test:` commit per place

## 3. The worklist by hand

- [ ] 3.1 Clear each worklist row by the rule, or record the reason in the report
- [ ] 3.2 `enh cite-lint` reports 0 findings and `enh cite-rewrite --dry-run` reports 0 edits
- [ ] 3.3 `task check` green, then commit docs: clear the remaining enhancement citations

## 4. Archive

- [ ] 4.1 Verify, review, fix findings, then archive the change and commit docs(openspec): archive align-enhancement-store
