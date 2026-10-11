package engine

import (
	"fmt"

	databasesv1alpha1 "github.com/vaibhavmahiwal/mini_everest/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

type Profile interface {
	ContainerImage() string
	ContainerPort() int32
	BackupCommand(dbc) []string
	Env(dbc *databasesv1alpha1.DatabaseCluster) []corev1.EnvVar
}

func getProfile(engine string) (Profile, error) {
	if engine == "postgresql" {
		return &postgresProfile{}, nil
	} else if engine == "valkey" {
		return &valkeyProfile{}, nil
	}
	return nil, fmt.Errorf("unsupported engine: %s", engine)
}
