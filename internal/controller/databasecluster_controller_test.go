//go:build !windows

package controller

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	databasesv1alpha1 "github.com/vaibhavmahiwal/mini_everest/api/v1alpha1"
)

func testKey(name string) types.NamespacedName {
	return types.NamespacedName{
		Name:      name,
		Namespace: "default",
	}
}

func newCluster(name string) *databasesv1alpha1.DatabaseCluster {
	return &databasesv1alpha1.DatabaseCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: databasesv1alpha1.DatabaseClusterSpec{
			Engine:         "postgres",
			Replicas:       1,
			StorageSize:    "1Gi",
			BackupSchedule: "0 2 * * *",
		},
	}
}

func ensureNamespace(t *testing.T, name string) {
	t.Helper()
	ctx := context.Background()

	var ns corev1.Namespace
	err := testClient.Get(ctx, types.NamespacedName{Name: name}, &ns)
	if err == nil {
		return
	}
	if !apierrors.IsNotFound(err) {
		t.Fatalf("failed to get namespace %q: %v", name, err)
	}

	ns = corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	if err := testClient.Create(ctx, &ns); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("failed to create namespace %q: %v", name, err)
	}
}

// TestReconcileCreatesResources verifies the full fan-out: one CR produces a
// StatefulSet, a headless Service, and a backup CronJob, all owned by the CR.
func TestReconcileCreatesResources(t *testing.T) {
	ctx := context.Background()
	ensureNamespace(t, "default")

	dbc := newCluster("test-create")
	if err := testClient.Create(ctx, dbc); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = testClient.Delete(ctx, dbc)
	})

	r := newReconciler()
	if _, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: testKey(dbc.Name)}); err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	var sts appsv1.StatefulSet
	if err := testClient.Get(ctx, testKey(dbc.Name), &sts); err != nil {
		t.Fatalf("failed to get statefulset: %v", err)
	}
	if sts.Spec.Replicas == nil || *sts.Spec.Replicas != 1 {
		t.Errorf("replicas = %v, want 1", sts.Spec.Replicas)
	}
	if len(sts.OwnerReferences) != 1 {
		t.Errorf("ownerReferences len = %d, want 1", len(sts.OwnerReferences))
	}

	var svc corev1.Service
	if err := testClient.Get(ctx, testKey(dbc.Name), &svc); err != nil {
		t.Fatalf("failed to get service: %v", err)
	}
	if svc.Spec.ClusterIP != corev1.ClusterIPNone {
		t.Errorf("clusterIP = %q, want %q", svc.Spec.ClusterIP, corev1.ClusterIPNone)
	}

	var cj batchv1.CronJob
	if err := testClient.Get(ctx, testKey(dbc.Name+"-backup"), &cj); err != nil {
		t.Fatalf("failed to get cronjob: %v", err)
	}
	if cj.Spec.Schedule != "0 2 * * *" {
		t.Errorf("cron schedule = %q, want %q", cj.Spec.Schedule, "0 2 * * *")
	}
}

// TestReconcileIdempotent runs Reconcile twice and confirms the second pass
// does not rewrite unchanged children.
func TestReconcileIdempotent(t *testing.T) {
	ctx := context.Background()
	ensureNamespace(t, "default")

	dbc := newCluster("test-idempotent")
	if err := testClient.Create(ctx, dbc); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = testClient.Delete(ctx, dbc)
	})

	r := newReconciler()
	req := ctrl.Request{NamespacedName: testKey(dbc.Name)}
	if _, err := r.Reconcile(ctx, req); err != nil {
		t.Fatalf("first reconcile failed: %v", err)
	}

	var sts appsv1.StatefulSet
	if err := testClient.Get(ctx, testKey(dbc.Name), &sts); err != nil {
		t.Fatal(err)
	}
	var svc corev1.Service
	if err := testClient.Get(ctx, testKey(dbc.Name), &svc); err != nil {
		t.Fatal(err)
	}
	var cj batchv1.CronJob
	if err := testClient.Get(ctx, testKey(dbc.Name+"-backup"), &cj); err != nil {
		t.Fatal(err)
	}

	stsRV, svcRV, cjRV := sts.ResourceVersion, svc.ResourceVersion, cj.ResourceVersion

	if _, err := r.Reconcile(ctx, req); err != nil {
		t.Fatalf("second reconcile failed: %v", err)
	}

	if err := testClient.Get(ctx, testKey(dbc.Name), &sts); err != nil {
		t.Fatal(err)
	}
	if err := testClient.Get(ctx, testKey(dbc.Name), &svc); err != nil {
		t.Fatal(err)
	}
	if err := testClient.Get(ctx, testKey(dbc.Name+"-backup"), &cj); err != nil {
		t.Fatal(err)
	}

	if sts.ResourceVersion != stsRV {
		t.Errorf("statefulset was rewritten: resourceVersion %q -> %q", stsRV, sts.ResourceVersion)
	}
	if svc.ResourceVersion != svcRV {
		t.Errorf("service was rewritten: resourceVersion %q -> %q", svcRV, svc.ResourceVersion)
	}
	if cj.ResourceVersion != cjRV {
		t.Errorf("cronjob was rewritten: resourceVersion %q -> %q", cjRV, cj.ResourceVersion)
	}
}

// TestReconcileSyncsStatus fakes the StatefulSet's observed status (envtest has
// no kubelet) and confirms it is copied into the CR's status subresource.
func TestReconcileSyncsStatus(t *testing.T) {
	ctx := context.Background()
	ensureNamespace(t, "default")

	dbc := newCluster("test-status")
	if err := testClient.Create(ctx, dbc); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = testClient.Delete(ctx, dbc)
	})

	r := newReconciler()
	req := ctrl.Request{NamespacedName: testKey(dbc.Name)}
	if _, err := r.Reconcile(ctx, req); err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	var sts appsv1.StatefulSet
	if err := testClient.Get(ctx, testKey(dbc.Name), &sts); err != nil {
		t.Fatal(err)
	}
	sts.Status.ReadyReplicas = 1
	if err := testClient.Status().Update(ctx, &sts); err != nil {
		t.Fatalf("failed to update statefulset status: %v", err)
	}

	if _, err := r.Reconcile(ctx, req); err != nil {
		t.Fatalf("reconcile after status update failed: %v", err)
	}

	var got databasesv1alpha1.DatabaseCluster
	if err := testClient.Get(ctx, testKey(dbc.Name), &got); err != nil {
		t.Fatal(err)
	}
	if got.Status.ReadyReplicas != 1 {
		t.Errorf("status.readyReplicas = %d, want 1", got.Status.ReadyReplicas)
	}
}

// TestReconcileBackupCronJobOptional verifies the CronJob exists when a
// schedule is set and is deleted when the schedule is cleared.
func TestReconcileBackupCronJobOptional(t *testing.T) {
	ctx := context.Background()
	ensureNamespace(t, "default")

	dbc := newCluster("test-backup")
	if err := testClient.Create(ctx, dbc); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = testClient.Delete(ctx, dbc)
	})

	r := newReconciler()
	req := ctrl.Request{NamespacedName: testKey(dbc.Name)}
	if _, err := r.Reconcile(ctx, req); err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	var cj batchv1.CronJob
	if err := testClient.Get(ctx, testKey(dbc.Name+"-backup"), &cj); err != nil {
		t.Fatalf("expected cronjob to exist: %v", err)
	}

	var current databasesv1alpha1.DatabaseCluster
	if err := testClient.Get(ctx, testKey(dbc.Name), &current); err != nil {
		t.Fatal(err)
	}
	current.Spec.BackupSchedule = ""
	if err := testClient.Update(ctx, &current); err != nil {
		t.Fatalf("failed to clear backupSchedule: %v", err)
	}

	if _, err := r.Reconcile(ctx, req); err != nil {
		t.Fatalf("reconcile after clearing schedule failed: %v", err)
	}

	err := testClient.Get(ctx, testKey(dbc.Name+"-backup"), &cj)
	if !apierrors.IsNotFound(err) {
		t.Errorf("cronjob still present after clearing schedule (err = %v)", err)
	}
}

// TestReconcileMissingCluster confirms a deleted/absent CR is not an error.
func TestReconcileMissingCluster(t *testing.T) {
	ctx := context.Background()
	ensureNamespace(t, "default")

	r := newReconciler()
	res, err := r.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "does-not-exist", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("expected nil error for missing CR, got %v", err)
	}
	if res.Requeue {
		t.Errorf("expected no requeue for missing CR")
	}
}
