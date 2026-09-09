# mini_everest 


# Pod Label Syncer-a helper project to learn

A tiny Kubernetes controller, built directly on `client-go` (no framework, no CRD), that keeps a Pod's `team` label in sync with its `sync-label` annotation.

It exists to teach the raw mechanics of a reconcile loop — informer, cache, work queue, worker — before using `controller-runtime` to hide them in later projects.

## What it does

Watches Pods in the `default` namespace. When a Pod has:

```yaml
metadata:
  annotations:
    sync-label: "team-x"
```

the controller adds/updates the label `team: team-x` on that Pod. Edit the annotation, and the label follows within a second or two. Already-correct Pods are left untouched (idempotent).

## Prerequisites

- Go 1.21+
- A running cluster and a working `kubectl` context (a local `kind` cluster is fine)
- `~/.kube/config` pointing at that cluster

## Install & Run

```bash
go mod tidy
go run main.go
```

Leave it running in a terminal — it blocks and processes events until you `Ctrl+C`.

## Try It

In another terminal:

```bash
kubectl run test --image=nginx
kubectl annotate pod test sync-label=team-x
kubectl get pod test --show-labels          # -> team=team-x

kubectl annotate pod test sync-label=team-y --overwrite
kubectl get pod test --show-labels          # -> team=team-y (label follows the annotation)
```

## How It Works

```
API server --(watch)--> Informer --(cache)--> Indexer
                            |
                      event handlers
                            |
                        Work Queue  (namespace/name keys only)
                            |
                       Worker loop --(GetByKey)--> Indexer
                            |
                    compare labels vs annotation
                            |
                  Update() only if out of sync --> API server
```

- **Informer**: watches the API server so the controller doesn't have to poll.
- **Indexer/cache**: local, eventually-consistent copy of Pod state, kept fresh by the informer.
- **Work queue**: decouples "something changed" from "do the work" — event handlers only push a key (`namespace/name`), never the object itself.
- **Worker**: pulls a key, re-reads current state from the cache, and only calls `Update()` if the label is actually wrong. This re-read-before-acting step is what makes the loop safe to run continuously.

## Project Structure

```
pod-label-syncer/
├── go.mod
├── go.sum
└── main.go   # everything lives here — client setup, informer, queue, worker, sync logic
```

## Known Limitations (intentional, for this stage)

- Hardcoded to the `default` namespace
- No removal of the `team` label if `sync-label` is deleted (see stretch goal below)
- No RBAC manifest — assumes you're running locally with a kubeconfig that already has permission to get/list/watch/update Pods
- No tests yet

## Stretch Goal

Make the sync bidirectional-clean: if `sync-label` is removed from a Pod, remove the `team` label too, instead of leaving it stale. Good five-minute exercise in extending the idempotency check in `syncPod`.

## Why This Project Exists

This is stepping stone #1 toward building [Mini-Everest](#), a full database operator with CRDs and StatefulSets. The next stepping stone (`website-operator`) rebuilds this same loop using `controller-runtime`, so the informer/queue/worker built here can be compared directly against what the framework automates.