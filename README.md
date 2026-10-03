# Mini-Everest — Kubernetes Database Operator in Go

A Kubernetes Operator that automates the lifecycle of containerized database instances (PostgreSQL, Valkey) using Custom Resource Definitions (CRDs). Modeled on how production operators like [OpenEverest](https://github.com/percona/everest), [CloudNativePG](https://github.com/cloudnative-pg/cloudnative-pg), and the [Zalando Postgres Operator](https://github.com/zalando/postgres-operator) work.

> **Status: in progress (active development).** See [Build Roadmap](#build-roadmap) for what's done. Full plan lives in [`docs/plan.md`](docs/plan.md).

Instead of hand-writing and maintaining YAML for StatefulSets, storage, networking, and backups every time a database is needed, you declare **intent** via a custom resource:

```yaml
apiVersion: databases.mini-everest.io/v1alpha1
kind: DatabaseCluster
metadata:
  name: my-postgres
spec:
  engine: postgres
  replicas: 3
  storageSize: 10Gi
  backupSchedule: "0 2 * * *"  # planned
```

The controller watches for objects like this and continuously provisions and maintains everything underneath — a StatefulSet, a headless Service, PersistentVolumeClaims, and (planned) a backup CronJob — automatically, idempotently, forever.

**Not in scope:** writing a database engine from scratch. This project orchestrates existing database images (`postgres:16`, `valkey/valkey`, etc.) via Kubernetes primitives. The value is in the automation layer, not the database itself.

---

## Table of Contents

- [How It Works](#how-it-works)
- [Core Concepts](#core-concepts)
- [Tech Stack](#tech-stack)
- [Repository Structure](#repository-structure)
- [API Reference](#api-reference)
- [What the Controller Manages Today](#what-the-controller-manages-today)
- [Quickstart](#quickstart)
- [Build Roadmap](#build-roadmap)
- [Design Decisions](#design-decisions)
- [Known Gaps vs Production Operators](#known-gaps-vs-production-operators)
- [Testing Strategy](#testing-strategy)
- [Reference Projects](#reference-projects)

---

## How It Works

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
         ├──▶ PVC(s)        (via StatefulSet volumeClaimTemplates — planned)
         └──▶ CronJob       (scheduled backups — planned)
         │
         ▼
   .status updated to reflect real observed state
```

Key entry points:

- `main.go` — manager setup, scheme registration (`client-go` scheme + `databasesv1alpha1`), entry point.
- `internal/controller/databasecluster_controller.go` — `Reconcile()` + `SetupWithManager()`. Currently reconciles a StatefulSet, a headless Service, then syncs `.status.readyReplicas`.
- `api/v1alpha1/databasecluster_types.go` — `Spec`/`Status` struct definitions (hand-edited source of truth for codegen).
- `config/crd/bases/*.yaml` — generated CRD schema applied to the cluster.

---

## Core Concepts

This project is deliberately a learning vehicle for the operator pattern:

| Concept | How Mini-Everest uses it |
|---|---|
| **CRD → CR → Controller → Reconcile loop** | `DatabaseCluster` CRD defines the schema; each CR instance triggers `Reconcile()` |
| **Level-triggered reconciliation** | The reconciler is told "something changed", not "what changed", and always recomputes desired state from scratch |
| **Idempotency** | `controllerutil.CreateOrUpdate` makes it safe to run the same reconcile 1,000 times with no side effects — writes only happen on real diffs |
| **Spec vs Status** | `spec` is user-authored desired state; `status.readyReplicas` is controller-authored observed state (via `+kubebuilder:subresource:status`). Never crossed |
| **Owner references & cascade delete** | `SetControllerReference` ties the StatefulSet/Service to the CR, so deleting the CR garbage-collects children automatically |
| **Multi-resource orchestration** | One CR fans out to several owned resource types (`Owns(&StatefulSet{})`, `Owns(&Service{})`, CronJob planned) |
| **Pluggable behavior via interfaces** | Planned `EngineProfile` interface (`internal/engine/`) isolates Postgres vs Valkey differences (image, env vars, ports) from the reconciler |

---

## Tech Stack

| Layer | Choice |
|---|---|
| Language | Go 1.26 |
| K8s client library | `client-go` (raw, for the Pod Label Syncer stepping stone) + `controller-runtime` v0.25.1 (for Mini-Everest itself) |
| Scaffolding | `kubebuilder` layout (hand-rolled, no full kubebuilder init) |
| CRD / RBAC / deepcopy generation | `controller-gen` (`controller-gen.kubebuilder.io/version: v0.22.0`) |
| Local test cluster | `kind` |
| Testing | Go `testing` package, table-driven tests, `envtest` for integration tests (planned, Step 17) |

---

## Repository Structure

```
mini-everest/
├── main.go                                    # manager setup, scheme registration, entry point
├── go.mod / go.sum
│
├── api/v1alpha1/
│   ├── databasecluster_types.go               # Spec/Status struct definitions (hand-edited)
│   ├── groupversion_info.go                   # scheme registration (GroupVersion v1alpha1)
│   └── zz_generated.deepcopy.go               # generated via controller-gen — never hand-edit
│
├── internal/
│   └── controller/
│       └── databasecluster_controller.go      # Reconcile() + SetupWithManager()
│       # planned (Step 15):
│       # └── engine/profile.go, postgres.go, valkey.go
│
├── config/
│   ├── crd/bases/*.yaml                       # generated CRD schema
│   ├── rbac/manager-role.yaml / role.yaml     # generated from +kubebuilder:rbac markers
│   └── samples/*.yaml                         # example CRs for testing
│
├── learn/                                     # stepping-stone exercises (separate from operator)
│   ├── pod-label-syncer/                      # raw client-go informer/workqueue demo
│   └── proj-1/
│
└── docs/
    ├── plan.md                                # full project plan, roadmap, design rationale
    └── NOTES.md                               # running log: bugs hit, how found, how fixed
```

---

## API Reference

Group: `databases.mini-everest.io`, Version: `v1alpha1`, Kind: `DatabaseCluster` (Namespaced).

**Spec** (`api/v1alpha1/databasecluster_types.go`):

| Field | Type | Required | Description |
|---|---|---|---|
| `engine` | `string` | yes | Database engine (e.g. `postgres`). Pluggable via `EngineProfile` (planned) |
| `replicas` | `int32` | yes | Desired number of pods; wired to `StatefulSet.spec.replicas` |
| `storageSize` | `string` | yes | Volume size (e.g. `1Gi`); to be wired into PVC `volumeClaimTemplates` (Step 14) |
| `backupSchedule` | `string` | no (planned) | Cron expression for backup CronJob (Step 16) |

**Status:**

| Field | Type | Description |
|---|---|---|
| `readyReplicas` | `int32` | Copied from the owned StatefulSet's `status.readyReplicas` each reconcile |

Sample CR (`config/samples/databases_v1alpha1_databasecluster.yaml`):

```yaml
apiVersion: databases.mini-everest.io/v1alpha1
kind: DatabaseCluster
metadata:
  name: my-postgres
spec:
  engine: postgresql
  replicas: 5
  storageSize: 1Gi
```

---

## What the Controller Manages Today

`Reconcile()` (`internal/controller/databasecluster_controller.go`) does three things in order:

1. **`reconcileStatefulSet`** — `CreateOrUpdate` a StatefulSet named after the CR:
   - `spec.replicas` from `spec.replicas`, `spec.serviceName` = CR name (required to bind to the headless Service)
   - Pod template: single `database` container, `postgres:16`, `POSTGRES_PASSWORD=postgres`, port `5432`, labeled `databasecluster: <cr-name>`
   - Owner reference set via `SetControllerReference` → cascade delete works
2. **`reconcileService`** — `CreateOrUpdate` a headless Service named after the CR:
   - `ClusterIP: None`, selector `databasecluster: <cr-name>`, port `5432 → 5432`
   - Gives each pod stable DNS (`my-postgres-0.my-postgres`) for StatefulSet replication/discovery
3. **`updateStatus`** — reads the live StatefulSet, copies `status.readyReplicas` to the CR status (no-op write if unchanged, so it's unit-testable in isolation later)

Watch setup:

```go
ctrl.NewControllerManagedBy(mgr).
    For(&databasesv1alpha1.DatabaseCluster{}).
    Owns(&appsv1.StatefulSet{}).
    Owns(&corev1.Service{}).
    Complete(r)
```

RBAC markers request: `get;list;watch` on `databaseclusters`, `get;update;patch` on `databaseclusters/status`, plus full CRUD on `statefulsets`, `persistentvolumeclaims`, `services`/`events`.

---

## Quickstart

Prerequisites: Go 1.26+, `kubectl`, a cluster (`kind create cluster`), `controller-gen` (for regeneration).

```bash
# 1. Install the CRD
kubectl apply -f config/crd/bases/databases.mini-everest.io_databaseclusters.yaml

# 2. Verify the new API
kubectl get databaseclusters

# 3. Run the operator locally (uses current kubeconfig)
go run ./...

# 4. In another terminal, create a cluster
kubectl apply -f config/samples/databases_v1alpha1_databasecluster.yaml

# 5. Watch what the operator built
kubectl get databaseclusters
kubectl get statefulset my-postgres
kubectl get svc my-postgres
kubectl get endpoints my-postgres
kubectl get databasecluster my-postgres -o yaml   # check .status.readyReplicas

# 6. Scale — edit spec.replicas, re-apply, watch pods roll
kubectl edit databasecluster my-postgres

# 7. Cascade delete — children disappear with the CR
kubectl delete databasecluster my-postgres
kubectl get statefulset,svc
```

Regenerate after editing API types or RBAC markers:

```bash
controller-gen object paths="./api/..."
controller-gen crd paths="./api/..." output:crd:dir=config/crd/bases
controller-gen rbac paths="./..." output:artifacts:config=config/rbac
go build ./...
```

---

## Build Roadmap

Phases -1 and 0 (Go fundamentals, kubectl exercises, Pod Label Syncer with raw `client-go`, Website Operator with `controller-runtime`) are done in separate repos / `learn/`.

Mini-Everest steps (from `docs/plan.md`):

- [x] **Step 1** — `go mod init`, folder scaffold, Go toolchain works
- [x] **Step 2** — `DatabaseClusterSpec`/`Status`/`DatabaseCluster`/`List` defined
- [x] **Step 3** — Scheme registration in `groupversion_info.go`
- [x] **Step 4** — `zz_generated.deepcopy.go` via `controller-gen object`; `go build ./...` passes
- [x] **Step 5** — CRD YAML generated and applied; `kubectl get databaseclusters` works
- [x] **Step 6** — Do-nothing reconciler: fetches CR, logs it, proves watch/reconcile chain
- [x] **Step 7** — Reconciler creates a real Deployment sized from `spec.replicas` via `CreateOrUpdate` + `SetControllerReference`
- [ ] **Step 8** — Confirm cascade delete + idempotency stress test (re-apply N times, zero drift)
- [ ] **Step 9** — Confirm generated RBAC actually lists the right resources/verbs
- [ ] **Step 10** — `docs/NOTES.md` caught up to date
- [x] **Step 11** — Swap Deployment → StatefulSet + `VolumeClaimTemplates` *(in progress — compile fixes landed, storage wiring remains)*
- [x] **Step 12** — Headless Service as second owned resource *(code landed — `reconcileService` + `Owns(&Service{})`; endpoint verification pending)*
- [x] **Step 13** — Sync `.status.readyReplicas` from live StatefulSet *(code landed — `updateStatus`; live verification pending)*
- [ ] **Step 14** — Wire `storageSize` into PVC resource request; add `backupSchedule` spec field
- [ ] **Step 15** — `EngineProfile` interface (`postgresProfile{}`/`valkeyProfile{}`); remove hardcoded `postgres:16` from controller
- [ ] **Step 16** — Backup CronJob as third owned resource from `spec.backupSchedule`
- [ ] **Step 17** — Unit tests for desired-state builders + `envtest` integration tests for `Reconcile()`
- [ ] **Step 18** — Production concerns: leader election, `PodDisruptionBudget`, Kubernetes `Event` recording, finalizers (only if external cleanup needed)

---

## Design Decisions

| Decision | Why |
|---|---|
| `controllerutil.CreateOrUpdate` instead of hand-rolled Get/Create/Update | Enforces idempotency mechanically — mutate function only triggers a write if something actually changed |
| `EngineProfile` interface instead of `if engine == "postgres"` branching | Keeps engine-specific logic (image, env vars, ports) out of the reconciler; testable in isolation, no Kubernetes dependency |
| One `data` PVC per replica via `VolumeClaimTemplates` | StatefulSet-native — storage stays bound to a specific pod ordinal across restarts, unlike a shared volume |
| Headless Service (`ClusterIP: None`) | Gives each pod stable, individually addressable DNS (`my-postgres-0.my-postgres`); required by `StatefulSetSpec.ServiceName` itself |
| `spec.backupSchedule` as a plain string (not a separate CRD) | Deliberately simplified for the learning stage; CloudNativePG splits this into `Backup`/`ScheduledBackup` CRDs — noted as a refinement, not implemented yet |
| Built Pod Label Syncer with raw `client-go` before `controller-runtime` | To see the informer/cache/work-queue machinery directly before relying on the framework that hides it |

---

## Known Gaps vs Production Operators

Intentional scope cuts for now (not oversights):

- No read/write-split Services (`-rw`/`-ro` as CloudNativePG does) — one Service covers all pods
- No in-Pod health/instance-manager process — relies on Kubernetes liveness/readiness probes
- No automated failover/primary election — StatefulSet ordinal 0 is primary by convention only, not enforced
- No admission webhooks — invalid specs (e.g. bad cron syntax) are caught in-reconciler, not rejected at `kubectl apply` time
- No connection pooling layer (PgBouncer equivalent)

---

## Testing Strategy

- **Manual, per-step:** apply sample CR → inspect created resources → edit spec, re-apply → confirm correct diff → delete CR → confirm cascade cleanup
- **Idempotency check:** loop `kubectl apply` on an unchanged CR N times, confirm zero errors and zero unnecessary API writes
- **Unit tests (Step 17):** table-driven tests on pure "build desired StatefulSet/Service" functions — no cluster required
- **Integration tests (Step 17):** `envtest` spins up a real API server + etcd (no kubelet) to test `Reconcile()` against real API behavior

`docs/NOTES.md` is the running bug log — bugs hit, how found, how fixed — and doubles as interview material.

---

## Reference Projects

For comparison, not copying:

- [`kubernetes/sample-controller`](https://github.com/kubernetes/sample-controller) — canonical minimal client-go example
- [Kubebuilder Book](https://book.kubebuilder.io/) — CronJob tutorial, closest guided walkthrough to this project's shape
- [CloudNativePG](https://github.com/cloudnative-pg/cloudnative-pg) — production Postgres operator, same core pattern with real operational depth
- [Zalando Postgres Operator](https://github.com/zalando/postgres-operator) — another mature reference implementation
- [AWS Controllers for Kubernetes (ACK)](https://github.com/aws-controllers-k8s) — same operator pattern, reconciling against real AWS APIs instead of in-cluster resources
