# Mini-Everest — Project Plan

A Kubernetes Operator in Go that automates the lifecycle of containerized database instances (PostgreSQL, Valkey) using Custom Resource Definitions, modeled on how production operators like OpenEverest and CloudNativePG work.

---

## 1. What This Project Is

Instead of hand-writing and maintaining YAML for StatefulSets, storage, networking, and backups every time a database is needed, a user declares intent via a custom resource:

```yaml
apiVersion: databases.mini-everest.io/v1alpha1
kind: DatabaseCluster
metadata:
  name: my-postgres
spec:
  engine: postgres
  replicas: 3
  storageSize: 10Gi
  backupSchedule: "0 2 * * *"
```

A controller (this project) watches for objects like this and continuously provisions/maintains everything underneath: a StatefulSet, a headless Service, PersistentVolumeClaims, and a backup CronJob — automatically, idempotently, forever.

**Not in scope:** writing a database engine from scratch. This project orchestrates existing database images (`postgres:16`, `valkey/valkey`, etc.) via Kubernetes primitives — the value is in the automation layer, not the database itself.

---

## 2. Core Concepts This Project Teaches

- **CRD → CR → Controller → Reconcile Loop** — the full operator pattern
- **Level-triggered reconciliation** — the reconciler is told "something changed," not "what changed," and always recomputes from scratch
- **Idempotency** — safe to run the same reconcile 1,000 times with no side effects
- **Spec vs. Status** — user-authored desired state vs. controller-authored observed state, never crossed
- **Owner references & cascade delete** — child resources cleaned up automatically via Kubernetes' garbage collector
- **Multi-resource orchestration** — one CR fans out to several owned resource types (StatefulSet, Service, PVC, CronJob)
- **Pluggable behavior via interfaces** — an `EngineProfile` interface for Postgres vs. Valkey differences

---

## 3. Tech Stack

| Layer | Choice |
|---|---|
| Language | Go |
| K8s client library | `client-go` (raw, for the Pod Label Syncer stepping stone) + `controller-runtime` (for Mini-Everest itself) |
| Scaffolding | `kubebuilder` |
| CRD/RBAC/deepcopy generation | `controller-gen` |
| Local test cluster | `kind` |
| Testing | Go's `testing` package, table-driven tests, `envtest` for integration tests |

---

## 4. Architecture Overview

```
kubectl apply (DatabaseCluster CR)
         │
         ▼
   API Server ── validates against CRD schema, stores in etcd
         │
         ▼
   Informer (via controller-runtime Manager) ── detects the change
         │
         ▼
   Work Queue ── holds the object's namespace/name key
         │
         ▼
   Reconcile() ── re-fetches current state, builds desired state,
         │          compares, creates/updates only what's needed
         ├──▶ StatefulSet   (the database pods themselves)
         ├──▶ Service       (headless, stable per-pod DNS)
         ├──▶ PVC(s)        (via StatefulSet volumeClaimTemplates)
         └──▶ CronJob       (scheduled backups)
         │
         ▼
   .status updated to reflect real observed state
```

---

## 5. Repository Structure

```
mini-everest/
├── main.go                                    # manager setup, scheme registration, entry point
├── go.mod / go.sum
│
├── api/v1alpha1/
│   ├── databasecluster_types.go               # Spec/Status struct definitions (hand-edited)
│   ├── groupversion_info.go                   # scheme registration (hand-edited, boilerplate)
│   └── zz_generated.deepcopy.go                # generated — never hand-edit
│
├── internal/
│   ├── controller/
│   │   └── databasecluster_controller.go       # Reconcile() + SetupWithManager()
│   └── engine/                                 # added at Step 15
│       ├── profile.go                          # EngineProfile interface
│       ├── postgres.go
│       └── valkey.go
│
├── config/
│   ├── crd/bases/*.yaml                        # generated CRD schema
│   ├── rbac/manager-role.yaml                  # generated from +kubebuilder:rbac markers
│   └── samples/*.yaml                          # example CRs for testing
│
├── docs/
│   └── NOTES.md                                # running log: bugs hit, how found, how fixed
│
└── PLAN.md                                     # this file
```

---

## 6. Build Roadmap

### Phase -1 — Prerequisites (done before this repo existed)
- [x] Go fundamentals (structs, pointers, interfaces, error handling, JSON tags)
- [x] `kubectl`-only Kubernetes exercises (Pods, Deployments, Services, StatefulSets, CronJobs, watching self-healing)

### Phase 0 — Stepping stones (separate repos)
- [x] Pod Label Syncer — raw `client-go`, hand-built informer/work queue/worker
- [x] Website Operator — first CRD, `controller-runtime`, Deployment + Service

### Mini-Everest build steps

- [x] **Step 1** — `go mod init`, folder scaffold, confirms Go toolchain works
- [x] **Step 2** — `DatabaseClusterSpec`/`Status`/`DatabaseCluster`/`DatabaseClusterList` defined in `api/v1alpha1/databasecluster_types.go`
- [x] **Step 3** — Scheme registration in `groupversion_info.go`
- [x] **Step 4** — `zz_generated.deepcopy.go` generated via `controller-gen object`; `go build ./...` passes
- [x] **Step 5** — CRD YAML generated and applied; `kubectl get databaseclusters` works
- [x] **Step 6** — Do-nothing reconciler: fetches the CR, logs it, proves the full watch/reconcile chain
- [x] **Step 7** — Reconciler creates a real `Deployment` sized from `spec.replicas`, using `controllerutil.CreateOrUpdate` + `SetControllerReference`
- [ ] **Step 8** — Confirm cascade delete (deleting the CR removes the Deployment) + idempotency stress test (re-apply N times, zero drift)
- [ ] **Step 9** — Confirm generated `config/rbac/manager-role.yaml` actually lists the right resources/verbs
- [ ] **Step 10** — `docs/NOTES.md` caught up to date
- [x] **Step 11** — Swap `Deployment` → `StatefulSet`, add `VolumeClaimTemplates` for per-replica storage *(in progress — fixing compile errors)*
- [ ] **Step 12** — Add headless `Service` as a second owned resource; `.Owns(&corev1.Service{})`; verify `kubectl get endpoints` shows real endpoints *(in progress)*
- [ ] **Step 13** — Sync `.status.readyReplicas` from the live StatefulSet's observed status
- [ ] **Step 14** — Add `storageSize`/`backupSchedule` fields to the spec; wire `storageSize` into the PVC's resource request
- [ ] **Step 15** — `EngineProfile` interface; `postgresProfile{}`/`valkeyProfile{}`; remove hardcoded `postgres:16` image from the controller
- [ ] **Step 16** — Backup `CronJob` as a third owned resource, built from `spec.backupSchedule`
- [ ] **Step 17** — Unit tests for desired-state builders (pure functions); `envtest` integration tests for `Reconcile()`
- [ ] **Step 18** — Production concerns: leader election, `PodDisruptionBudget`, Kubernetes `Event` recording, finalizers (only if external cleanup is ever needed)

---

## 7. Design Decisions & Rationale

| Decision | Why |
|---|---|
| `controllerutil.CreateOrUpdate` instead of hand-rolled Get/Create/Update | Enforces the idempotency pattern mechanically — mutate function only triggers a write if something actually changed |
| `EngineProfile` interface instead of `if engine == "postgres"` branching | Keeps engine-specific logic (image, env vars, ports) out of the reconciler; testable in isolation, no Kubernetes dependency |
| One `data` PVC per replica via `VolumeClaimTemplates` | StatefulSet-native mechanism — storage stays bound to a specific pod ordinal across restarts, unlike a shared volume |
| Headless Service (`ClusterIP: None`) | Gives each pod stable, individually addressable DNS (`my-postgres-0.my-postgres`) — required for StatefulSet replication/discovery, and required by `StatefulSetSpec.ServiceName` itself |
| `spec.backupSchedule` as a plain string for now (not a separate CRD) | Deliberately simplified for the learning stage; CloudNativePG splits this into its own `Backup`/`ScheduledBackup` CRDs — noted as a real-world refinement, not implemented yet |
| Built Pod Label Syncer with raw `client-go` before `controller-runtime` | To see the informer/cache/work-queue machinery directly before relying on the framework that hides it |

---

## 8. Known Gaps vs. Production Operators (intentional, for now)

Documented explicitly so this reads as informed scope-cutting, not oversight:

- No read/write-split Services (`-rw`/`-ro`, as CloudNativePG does) — one Service covers all pods for now
- No in-Pod health/instance-manager process — relying on Kubernetes' own liveness/readiness probes
- No automated failover/primary election logic — StatefulSet ordinal 0 is treated as primary by convention only, not enforced
- No admission webhooks — invalid specs (e.g., bad cron syntax) are only caught in-reconciler, not rejected at `kubectl apply` time
- No connection pooling layer (PgBouncer equivalent)

---

## 9. Testing Strategy

- **Manual, per-step:** apply sample CR → inspect created resources → edit spec, re-apply → confirm correct diff behavior → delete CR → confirm cascade cleanup
- **Idempotency check:** loop `kubectl apply` on an unchanged CR N times, confirm zero errors and zero unnecessary API writes
- **Unit tests (Step 17):** table-driven tests on pure "build desired StatefulSet/Service" functions — no cluster required
- **Integration tests (Step 17):** `envtest` spins up a real API server + etcd (no kubelet) to test `Reconcile()` against real API behavior

---

## 10. Resume / Portfolio Notes

- Project status should always be labeled accurately (`in progress` until Step 18 is meaningfully complete) — see `docs/NOTES.md` for the running log of real bugs hit, which doubles as interview material
- Public repo + README + this PLAN.md are the actual portfolio artifact; resume bullets should only claim what's checked off above
- Good candidate for an architecture diagram and a short terminal-recording demo (`kubectl apply` → pods appear → edit replicas → scale → delete → cascade cleanup) rather than attempting a live-hosted demo

---

## 11. Reference Projects (for comparison, not copying)

- [`kubernetes/sample-controller`](https://github.com/kubernetes/sample-controller) — canonical minimal client-go example
- [Kubebuilder Book](https://book.kubebuilder.io/) — CronJob tutorial, closest guided walkthrough to this project's shape
- [CloudNativePG](https://github.com/cloudnative-pg/cloudnative-pg) — production Postgres operator, same core pattern with real operational depth
- [Zalando Postgres Operator](https://github.com/zalando/postgres-operator) — another mature reference implementation
- [AWS Controllers for Kubernetes (ACK)](https://github.com/aws-controllers-k8s) — same operator pattern, reconciling against real AWS APIs instead of in-cluster resources