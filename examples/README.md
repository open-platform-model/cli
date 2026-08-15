# Examples

Deployable `#ModuleInstance` examples on the OPM v2 line, importing the
neutral `opmodel.dev/modules/test/*` fixture modules.

- `instances/podinfo/` — Deployment + Service with HTTP liveness/readiness
  probes (`opmodel.dev/modules/test/podinfo@v0`).

The former `hello-web` example was removed at the core v2 crossing: its
hyphenated module name cannot exist on the v2 line (module names are
snake_case and equal the module path leaf). The fixture fleet republishes it
as `hello_web`; a new example importing
`opmodel.dev/modules/test/hello_web@v0` can be added once needed.

Build:   `opm instance build ./instances/podinfo/instance.cue`
Apply:   `opm instance apply ./instances/podinfo/instance.cue --create-namespace`
