## 1. Vet exits 4 for a refused registry credential

- [x] 1.1 `internal/cmdutil/publish.go`: move the mapping of a failed core schema fetch into `CoreSchemaError` and call it from `RunPublish`; the publish tests stay green
- [x] 1.2 `internal/cmd/module/vet.go`: map the failed fetch through `CoreSchemaError` and state the exit codes in the help; `TestModVet_SchemaFetchFailureIsNamed` (401 exits 4 with the login hint, refused connection and 503 exit 3, 403 pinned as the known limit) and `TestNewModuleVetCmd_HelpStatesTheExitCodes` fail before and pass after
- [x] 1.3 `task fmt`, `task vet`, `task lint`, `task openspec:check`, `task docs:bundle:check`, `task test:unit` and `task cascade:wiring:check` green, then commit `fix(cmd): exit 4 from module vet when the registry refuses the credentials`
