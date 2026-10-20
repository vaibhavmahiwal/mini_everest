package engine

import (
	"fmt"

	databasesv1alpha1 "github.com/vaibhavmahiwal/mini_everest/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

type postgresProfile struct{}

func (postgresProfile) ContainerImage() string {
	return "postgres:16"
}

func (postgresProfile) ContainerPort() int32 {
	return 5432
}

func (postgresProfile) DataMountPath() string {
	return "/var/lib/postgresql/data"
}

func (postgresProfile) Env(dbc *databasesv1alpha1.DatabaseCluster) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "POSTGRES_PASSWORD",
			Value: "postgres",
		},
	}
}

func (postgresProfile) BackupEnv(dbc *databasesv1alpha1.DatabaseCluster) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "PGPASSWORD",
			Value: "postgres",
		},
	}
}

func (postgresProfile) BackupCommand(dbc *databasesv1alpha1.DatabaseCluster) []string {
	host := fmt.Sprintf("%s-0.%s", dbc.Name, dbc.Name)
	return []string{
		"/bin/sh",
		"-c",
		fmt.Sprintf("pg_dump -h %s -U postgres postgres > /tmp/backup.sql && echo backup ok", host),
	}
}
