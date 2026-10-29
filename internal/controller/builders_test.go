package controller

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	databasesv1alpha1 "github.com/vaibhavmahiwal/mini_everest/api/v1alpha1"
	"github.com/vaibhavmahiwal/mini_everest/internal/engine"
)

func newTestCluster(engineName string) *databasesv1alpha1.DatabaseCluster {
	return &databasesv1alpha1.DatabaseCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-postgres",
			Namespace: "default",
		},
		Spec: databasesv1alpha1.DatabaseClusterSpec{
			Engine:      engineName,
			Replicas:    3,
			StorageSize: "2Gi",
		},
	}
}

func mustProfile(t *testing.T, engineName string) engine.Profile {
	t.Helper()
	p, err := engine.For(engineName)
	if err != nil {
		t.Fatalf("engine.For(%q): %v", engineName, err)
	}
	return p
}

func TestBuildStatefulSet(t *testing.T) {
	dbc := newTestCluster("postgres")
	sts, err := buildStatefulSet(dbc, mustProfile(t, "postgres"))
	if err != nil {
		t.Fatalf("buildStatefulSet() error = %v", err)
	}

	if sts.Name != "my-postgres" {
		t.Errorf("name = %q, want %q", sts.Name, "my-postgres")
	}
	if sts.Namespace != "default" {
		t.Errorf("namespace = %q, want %q", sts.Namespace, "default")
	}
	if sts.Spec.Replicas == nil || *sts.Spec.Replicas != 3 {
		t.Errorf("replicas = %v, want 3", sts.Spec.Replicas)
	}
	if sts.Spec.ServiceName != "my-postgres" {
		t.Errorf("serviceName = %q, want %q", sts.Spec.ServiceName, "my-postgres")
	}
	if got := sts.Spec.Selector.MatchLabels["databasecluster"]; got != "my-postgres" {
		t.Errorf("selector label = %q, want %q", got, "my-postgres")
	}

	// storage: one PVC template, named data, requesting storageSize
	if len(sts.Spec.VolumeClaimTemplates) != 1 {
		t.Fatalf("volumeClaimTemplates len = %d, want 1", len(sts.Spec.VolumeClaimTemplates))
	}
	pvc := sts.Spec.VolumeClaimTemplates[0]
	if pvc.Name != "data" {
		t.Errorf("pvc name = %q, want %q", pvc.Name, "data")
	}
	if got := pvc.Spec.Resources.Requests.Storage().String(); got != "2Gi" {
		t.Errorf("pvc storage = %q, want %q", got, "2Gi")
	}

	// container: image, port, env, and the mount that binds the PVC
	containers := sts.Spec.Template.Spec.Containers
	if len(containers) != 1 {
		t.Fatalf("containers len = %d, want 1", len(containers))
	}
	c := containers[0]
	if c.Image != "postgres:16" {
		t.Errorf("image = %q, want %q", c.Image, "postgres:16")
	}
	if len(c.Ports) != 1 || c.Ports[0].ContainerPort != 5432 {
		t.Errorf("ports = %v, want [5432]", c.Ports)
	}
	if len(c.Env) != 1 || c.Env[0].Name != "POSTGRES_PASSWORD" {
		t.Errorf("env = %v, want POSTGRES_PASSWORD", c.Env)
	}
	if len(c.VolumeMounts) != 1 {
		t.Fatalf("volumeMounts len = %d, want 1", len(c.VolumeMounts))
	}
	if c.VolumeMounts[0].Name != "data" || c.VolumeMounts[0].MountPath != "/var/lib/postgresql/data" {
		t.Errorf("volumeMount = %+v, want data -> /var/lib/postgresql/data", c.VolumeMounts[0])
	}
}

func TestBuildStatefulSetValkey(t *testing.T) {
	dbc := newTestCluster("valkey")
	sts, err := buildStatefulSet(dbc, mustProfile(t, "valkey"))
	if err != nil {
		t.Fatalf("buildStatefulSet() error = %v", err)
	}

	c := sts.Spec.Template.Spec.Containers[0]
	if c.Image != "valkey/valkey:8" {
		t.Errorf("image = %q, want %q", c.Image, "valkey/valkey:8")
	}
	if len(c.Ports) != 1 || c.Ports[0].ContainerPort != 6379 {
		t.Errorf("ports = %v, want [6379]", c.Ports)
	}
	if len(c.Env) != 1 || c.Env[0].Name != "VALKEY_PASSWORD" {
		t.Errorf("env = %v, want VALKEY_PASSWORD", c.Env)
	}
	if len(c.VolumeMounts) != 1 || c.VolumeMounts[0].MountPath != "/data" {
		t.Errorf("volumeMounts = %+v, want mount at /data", c.VolumeMounts)
	}
}

func TestBuildStatefulSetInvalidStorageSize(t *testing.T) {
	dbc := newTestCluster("postgres")
	dbc.Spec.StorageSize = "garbage"

	sts, err := buildStatefulSet(dbc, mustProfile(t, "postgres"))
	if err == nil {
		t.Fatal("buildStatefulSet() error = nil, want invalid storage size error")
	}
	if sts != nil {
		t.Fatalf("buildStatefulSet() StatefulSet = %#v, want nil on error", sts)
	}
}

func TestBuildService(t *testing.T) {
	dbc := newTestCluster("postgres")
	svc := buildService(dbc, mustProfile(t, "postgres"))

	if svc.Name != "my-postgres" {
		t.Errorf("name = %q, want %q", svc.Name, "my-postgres")
	}
	if svc.Namespace != "default" {
		t.Errorf("namespace = %q, want %q", svc.Namespace, "default")
	}
	if svc.Spec.ClusterIP != "None" {
		t.Errorf("clusterIP = %q, want %q (headless)", svc.Spec.ClusterIP, "None")
	}
	if got := svc.Spec.Selector["databasecluster"]; got != "my-postgres" {
		t.Errorf("selector label = %q, want %q", got, "my-postgres")
	}
	if len(svc.Spec.Ports) != 1 || svc.Spec.Ports[0].Port != 5432 {
		t.Errorf("ports = %v, want [5432]", svc.Spec.Ports)
	}
}

func TestBuildServiceValkey(t *testing.T) {
	dbc := newTestCluster("valkey")
	svc := buildService(dbc, mustProfile(t, "valkey"))

	if len(svc.Spec.Ports) != 1 || svc.Spec.Ports[0].Port != 6379 {
		t.Errorf("ports = %v, want [6379]", svc.Spec.Ports)
	}
}
