## 1. Module scaffolding

- [x] 1.1 Create directory structure: `testing/jellyfin/` and `testing/jellyfin/cue.mod/`
- [x] 1.2 Create `cue.mod/module.cue` declaring the CUE module path and dependencies (`opmodel.dev/core@v0`, `opmodel.dev/resources/workload@v0`, `opmodel.dev/resources/storage@v0`, `opmodel.dev/traits/workload@v0`, `opmodel.dev/traits/network@v0`, `opmodel.dev/schemas@v0`)
- [x] 1.3 Run `cue mod tidy` in `testing/jellyfin/` to resolve and lock dependency versions

## 2. Module metadata and config schema (`module.cue`)

- [x] 2.1 Define module metadata: `apiVersion`, `name` ("jellyfin"), `version` ("0.1.0"), `description`, `defaultNamespace`
- [x] 2.2 Unify the top-level definition with `core.#Module`
- [x] 2.3 Define `#config` schema with fields: `image` (string, no default), `port` (int, bounded 1-65535), `puid` (int, default 1000), `pgid` (int, default 1000), `timezone` (string, no default), `publishedServerUrl` (optional string), `configStorageSize` (string), `media` (struct-keyed map with `mountPath` per entry)
- [x] 2.4 Add `values: #config` to bind concrete values to the schema

## 3. Component definition (`components.cue`)

- [x] 3.1 Import required resources and traits: `resources/workload.#Container`, `resources/storage.#Volumes`, `traits/workload.#Scaling`, `traits/workload.#HealthCheck`, `traits/workload.#RestartPolicy`, `traits/network.#Expose`
- [x] 3.2 Define the `jellyfin` component embedding `#Container`, `#Volumes`, `#Scaling`, `#Expose`, `#HealthCheck`, `#RestartPolicy`
- [x] 3.3 Set workload-type label to `"stateful"`
- [x] 3.4 Configure the container spec: name, image from `#config.image`, HTTP port (8096 targetPort), env vars mapped from config (`PUID`, `PGID`, `TZ`, and conditionally `JELLYFIN_PublishedServerUrl`)
- [x] 3.5 Configure volume mounts on the container: `/config` for the config PVC, plus dynamic mounts from `#config.media`
- [x] 3.6 Define volumes: `config` as `persistentClaim` with size from `#config.configStorageSize`, media volumes as `emptyDir`
- [x] 3.7 Configure scaling with `count: 1`
- [x] 3.8 Configure `#Expose` trait: expose HTTP port with `ClusterIP` type, `exposedPort` from `#config.port`
- [x] 3.9 Configure health check: liveness probe (`httpGet`, path `/health`, port 8096, initialDelaySeconds 30, periodSeconds 10) and readiness probe (`httpGet`, path `/health`, port 8096, initialDelaySeconds 10, periodSeconds 10)
- [x] 3.10 Set restart policy to `"Always"`

## 4. Default values (`values.cue`)

- [x] 4.1 Provide concrete defaults: `image: "lscr.io/linuxserver/jellyfin:latest"`, `port: 8096`, `puid: 1000`, `pgid: 1000`, `timezone: "Etc/UTC"`, `configStorageSize: "10Gi"`
- [x] 4.2 Define default media libraries: `tvshows: { mountPath: "/data/tvshows" }`, `movies: { mountPath: "/data/movies" }`
- [x] 4.3 Add comments documenting that media volumes use `emptyDir` by default and operators should override with real storage at release time

## 5. Validation

- [x] 5.1 Run `cue vet ./...` in `testing/jellyfin/` and fix any schema violations
- [x] 5.2 Verify the module structure matches the three-file convention (`module.cue`, `components.cue`, `values.cue`, `cue.mod/module.cue`)
- [x] 5.3 Verify the rendered output includes: StatefulSet-compatible labels, config PVC, media volume mounts, health probes, exposed service port, and all expected env vars
