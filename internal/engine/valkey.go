package engine

import (
	databasesv1alpha1 "github.com/vaibhavmahiwal/mini_everest/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

type valkeyProfile struct{}

func (p *valkeyProfile) ContainerImage() string {
	return "valkey:latest"
}

func (valkeyProfile) ContainerPort() int32 {
	return 6379
}

func (valkeyProfile) Env(dbc *databasesv1alpha1.DatabaseCluster) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "VALKEY_PASSWORD",
			Value: "valkey",
		},
	}
}
