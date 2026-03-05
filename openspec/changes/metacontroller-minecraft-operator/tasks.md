## 1. Project Scaffolding

- [ ] 1.1 Create `experiments/metacontroller/` directory structure with subdirectories: `cmd/`, `internal/`, `embed/`, `deploy/`, `examples/`
- [ ] 1.2 Initialize `go.mod` with module path `github.com/opmodel/experiments/metacontroller` and add dependencies: `cuelang.org/go`, `k8s.io/apimachinery`
- [ ] 1.3 Copy the `minecraft-java` module files (module.cue, components.cue, values.cue, cue.mod/) into `embed/minecraft-java/` for embedding. Strip values*.cue variants — only keep the base schema files and one default values.cue
- [ ] 1.4 Create the embedded provider CUE definition in `embed/provider.cue` — a minimal Kubernetes provider with transformers for StatefulSet, Service, Secret, and PVC generation

## 2. CUE Evaluation Pipeline

- [ ] 2.1 Implement `internal/cueeval/overlay.go` — build a `load.Config.Overlay` map from the `//go:embed` filesystem of the minecraft-java module, using a virtual base path
- [ ] 2.2 Implement `internal/cueeval/provider.go` — compile the embedded provider CUE string via `ctx.CompileString()` at startup and expose as a `cue.Value`
- [ ] 2.3 Implement `internal/cueeval/pipeline.go` — the core render function: accept a values CUE string, create fresh `cue.Context`, load module via overlay, build module release (FillPath chain), match transformers, execute to produce `[]map[string]any` Kubernetes resources
- [ ] 2.4 Implement `internal/cueeval/values.go` — write CUE values string to a temp file, return path for pipeline consumption, ensure cleanup via returned closer

## 3. Webhook Server

- [ ] 3.1 Define Go types for Metacontroller sync request/response in `internal/webhook/types.go` — SyncRequest (controller, parent, children, finalizing), SyncResponse (status, children, finalized, resyncAfterSeconds)
- [ ] 3.2 Implement `internal/webhook/translate.go` — translate MinecraftServer CR `.spec` fields to a CUE values string matching the minecraft-java `#config` schema
- [ ] 3.3 Implement `internal/webhook/status.go` — compute CR status from observed children: extract StatefulSet readyReplicas, derive phase (Pending/Running/Error), build conditions array
- [ ] 3.4 Implement `internal/webhook/handler.go` — the sync HTTP handler: parse request, handle finalize fast path, translate spec, call CUE pipeline, handle errors (preserve children), compute status, return response
- [ ] 3.5 Implement `cmd/main.go` — HTTP server entry point, wire up `/sync` route, configure port from env var or default 8080, add health check endpoint `/healthz`

## 4. Kubernetes Manifests

- [ ] 4.1 Create `deploy/crds/minecraftserver-crd.yaml` — MinecraftServer CRD with OpenAPI v3 schema, status subresource, and validation rules (maxPlayers 1-1000, enums for difficulty/mode/serviceType/storage.type)
- [ ] 4.2 Create `deploy/controller/composite-controller.yaml` — CompositeController resource with generateSelector, child resources (StatefulSet InPlace, Service InPlace, Secret InPlace, PVC OnDelete), sync+finalize webhook pointing to the webhook Service
- [ ] 4.3 Create `deploy/webhook/deployment.yaml` and `deploy/webhook/service.yaml` — Deployment (1 replica, port 8080, resource limits, health probe on /healthz) and ClusterIP Service (port 80 → 8080)
- [ ] 4.4 Create `examples/paper-server.yaml` — sample MinecraftServer CR with Paper server, 2G JVM, 10 players, easy difficulty, emptyDir storage, ClusterIP

## 5. Testing

- [ ] 5.1 Write unit tests for `internal/webhook/translate.go` — table-driven tests covering: defaults, custom properties, PVC storage, emptyDir storage
- [ ] 5.2 Write unit tests for `internal/webhook/status.go` — table-driven tests covering: no children, pending StatefulSet, running StatefulSet, error status
- [ ] 5.3 Write unit tests for `internal/webhook/handler.go` — test sync happy path, finalize path, and render-failure-preserves-children path using mock CUE pipeline
- [ ] 5.4 Write integration test for `internal/cueeval/pipeline.go` — verify that the full CUE evaluation pipeline produces valid StatefulSet and Service JSON from a known MinecraftServer spec

## 6. Documentation and Validation

- [ ] 6.1 Create `experiments/metacontroller/README.md` — overview, prerequisites (Metacontroller installed), build instructions, deployment steps, testing with sample CR
- [ ] 6.2 Verify the experiment builds: `go build ./cmd/...`
- [ ] 6.3 Verify tests pass: `go test ./...`
- [ ] 6.4 Verify Go formatting: `gofmt -l .` returns no output
