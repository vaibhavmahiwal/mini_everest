package engine

import (
	"fmt"
	"strings"

	databasesv1alpha1 "github.com/vaibhavmahiwal/mini_everest/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// Profile encapsulates everything that differs between database engines.
// The controller depends on this interface, never on a concrete engine.
type Profile interface {
	ContainerImage() string
	ContainerPort() int32
	// DataMountPath is where the engine expects its data volume.
	DataMountPath() string
	// Env is the environment for the database container.
	Env(dbc *databasesv1alpha1.DatabaseCluster) []corev1.EnvVar
	// BackupEnv is the environment needed by the backup job.
	BackupEnv(dbc *databasesv1alpha1.DatabaseCluster) []corev1.EnvVar
	// BackupCommand is the shell command used by the backup CronJob.
	BackupCommand(dbc *databasesv1alpha1.DatabaseCluster) []string
}

// For returns the Profile for the given engine name.
func For(engine string) (Profile, error) {
	switch strings.ToLower(strings.TrimSpace(engine)) {
	case "postgres", "postgresql":
		return &postgresProfile{}, nil
	case "valkey":
		return &valkeyProfile{}, nil
	}
	return nil, fmt.Errorf("unsupported engine: %s", engine)
}
