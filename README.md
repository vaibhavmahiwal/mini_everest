# Mini-Everest

A Kubernetes operator in Go that automates the lifecycle of containerized databases (PostgreSQL, Valkey). You declare intent with one custom resource; the controller provisions and maintains the StatefulSet, headless Service, per-pod storage, and backup CronJob underneath — idempotently, forever.

> **Status: in progress.** Core reconcile loop, both engines, storage, backups, status, and tests are working. No Dockerfile/manager Deployment yet (see [Roadmap](#roadmap)).

## Example

```yaml
apiVersion: databases.mini-everest.io/v1alpha1
kind: DatabaseCluster
metadata:
  name: my-postgres
spec:
  engine: postgres          # postgres | postgresql | valkey
  replicas: 3
  storageSize: 10Gi
  backupSchedule: "0 2 * * *"   # optional
```

```bash
kubectl apply -f my-postgres.yaml
kubectl get databaseclusters,statefulsets,services,cronjobs
```

## How it works

```
kubectl apply (DatabaseCluster)
        │
        ▼
   API server (validates against CRD)
        │
        ▼
   Reconcile()  ── fetch CR, build desired state, create/update only on diff
        ├──▶ StatefulSet     (DB pods, image/port/env/mount from engine profile)
        ├──▶ Service         (headless, stable per-pod DNS)
        ├──▶ PVC per replica (via volumeClaimTemplates)
        ├──▶ CronJob         (backup, only when backupSchedule is set)
        └──▶ .status.readyReplicas
```

Every child is owned by the CR (`SetControllerReference`), so deleting the CR garbage-collects them. All writes go through `controllerutil.CreateOrUpdate`, so re-running a reconcile is a no-op unless something actually changed.

## Engine profiles

Engine-specific behavior lives behind one interface (`internal/engine`), so the controller never branches on engine name:

| Engine | Image | Port | Data path | Env | Backup |
|---|---|---|---|---|---|
| postgres | `postgres:16` | 5432 | `/var/lib/postgresql/data` | `POSTGRES_PASSWORD` | `pg_dump` |
| valkey | `valkey/valkey:8` | 6379 | `/data` | `VALKEY_PASSWORD` | `valkey-cli save` |

`engine.For(spec.engine)` returns the profile; an unsupported engine returns an error at reconcile time.

## API

Group `databases.mini-everest.io`, version `v1alpha1`, namespaced.

| Field | Type | Required | Meaning |
|---|---|---|---|
| `spec.engine` | string | yes | Database engine |
| `spec.replicas` | int32 | yes | Desired pod count |
| `spec.storageSize` | string | yes | Per-replica PVC size (e.g. `10Gi`) |
| `spec.backupSchedule` | string | no | Cron expression; CronJob deleted when empty |
| `status.readyReplicas` | int32 | — | Copied from the owned StatefulSet |

## Layout

```
main.go                              manager + scheme wiring
api/v1alpha1/                        CRD types, scheme, generated deepcopy
internal/controller/
  databasecluster_controller.go      Reconcile, reconcilers, status
  builders.go                        pure buildStatefulSet / buildService
internal/engine/                     Profile interface + postgres/valkey
config/crd/bases/                    generated CRD
config/rbac/role.yaml                generated ClusterRole
config/samples/                      example DatabaseCluster
docs/plan.md                         full plan + rationale
docs/NOTES.md                        running bug log
```

## Quickstart

Prereqs: Go 1.26+, a cluster (e.g. `kind create cluster`), `controller-gen`.

```bash
# install CRD
kubectl apply -f config/crd/bases/databases.mini-everest.io_databaseclusters.yaml

# run the operator (uses current kubeconfig)
go run ./...

# in another terminal
kubectl apply -f config/samples/databases_v1alpha1_databasecluster.yaml
kubectl get databaseclusters
kubectl get sts,svc,cronjob
kubectl get databasecluster my-postgres-2 -o jsonpath='{.status.readyReplicas}'

# cascade delete
kubectl delete databasecluster my-postgres-2
```

## Tests

```bash
go build ./...
go vet ./...
go test ./... -count=1
```

- **Unit:** engine profiles (`internal/engine`) and desired-state builders (`internal/controller/builders_test.go`).
- **Integration (envtest):** `internal/controller/databasecluster_controller_test.go` runs a real apiserver + etcd and verifies create/idempotency/status/backup/missing-CR behavior.

envtest does **not** compile on Windows with controller-runtime v0.25.x, so those files are guarded with `//go:build !windows`. Run them in WSL/Linux/CI:

```bash
go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
export KUBEBUILDER_ASSETS="$(setup-envtest use -p path)"
go test ./internal/controller/... -v -count=1
```

CI (`.github/workflows/build.yaml`) runs format, build, vet, tests, and fails if generated CRD/RBAC/deepcopy is stale.

## Regenerate

```bash
controller-gen object paths="./api/v1alpha1/..."
controller-gen crd paths="./api/v1alpha1/..." output:crd:dir=config/crd/bases
controller-gen rbac:roleName=manager-role paths="./internal/controller/..." output:artifacts:config=config/rbac
```

## Roadmap

- [x] CRD, scheme, deepcopy, CRD apply
- [x] Reconcile → StatefulSet, headless Service, per-replica PVC
- [x] `storageSize` wired into PVC template and mounted
- [x] `status.readyReplicas` sync
- [x] Pluggable engine profiles (postgres/valkey)
- [x] Backup CronJob from `spec.backupSchedule`
- [x] Unit + envtest tests, CI
- [ ] Validate `storageSize` (currently `MustParse`, panics on bad input)
- [ ] Dockerfile + manager Deployment + ServiceAccount/RoleBinding
- [ ] Leader election, PodDisruptionBudget, Event recording
- [ ] Helm chart

## Known gaps (intentional)

No failover/primary election, no read/write split Services, no admission webhooks, no connection pooling, and the DB password is hardcoded. See `docs/plan.md` for the rationale.
