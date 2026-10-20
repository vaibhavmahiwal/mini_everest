package engine

import (
	"fmt"

	databasesv1alpha1 "github.com/vaibhavmahiwal/mini_everest/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

type valkeyProfile struct{}

func (valkeyProfile) ContainerImage() string {
	return "valkey/valkey:8"
}

func (valkeyProfile) ContainerPort() int32 {
	return 6379
}

func (valkeyProfile) DataMountPath() string {
	return "/data"
}

func (valkeyProfile) Env(dbc *databasesv1alpha1.DatabaseCluster) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "VALKEY_PASSWORD",
			Value: "valkey",
		},
	}
}

func (valkeyProfile) BackupEnv(dbc *databasesv1alpha1.DatabaseCluster) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "VALKEY_PASSWORD",
			Value: "valkey",
		},
	}
}

func (valkeyProfile) BackupCommand(dbc *databasesv1alpha1.DatabaseCluster) []string {
	host := fmt.Sprintf("%s-0.%s", dbc.Name, dbc.Name)
	return []string{
		"/bin/sh",
		"-c",
		fmt.Sprintf("valkey-cli -h %s save && echo backup ok", host),
	}
}
