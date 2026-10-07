package engine

import (
	databasesv1alpha1 "github.com/vaibhavmahiwal/mini_everest/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

type postgresProfile struct{}

func (p *postgresProfile) ContainerImage() string {
	return "postgres:latest"
}

func (postgresProfile) ContainerPort() int32 {
	return 5432
}

func (postgresProfile) Env(dbc *databasesv1alpha1.DatabaseCluster) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "POSTGRES_PASSWORD",
			Value: "postgres",
		},
	}
}
