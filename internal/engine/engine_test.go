package engine

import (
	"testing"

	databasesv1alpha1 "github.com/vaibhavmahiwal/mini_everest/api/v1alpha1"
)

func TestEnv(t *testing.T) {
	tests := []struct {
		name    string
		engine  string
		wantKey string
		wantVal string
		wantLen int
	}{
		{"postgres", "postgres", "POSTGRES_PASSWORD", "postgres", 1},
		{"valkey", "valkey", "VALKEY_PASSWORD", "valkey", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := getProfile(tt.engine)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			env := p.Env(&databasesv1alpha1.DatabaseCluster{})
			if len(env) != tt.wantLen {
				t.Fatalf("env len = %d, want %d", len(env), tt.wantLen)
			}
			if env[0].Name != tt.wantKey {
				t.Errorf("env name = %q, want %q", env[0].Name, tt.wantKey)
			}
			if env[0].Value != tt.wantVal {
				t.Errorf("env value = %q, want %q", env[0].Value, tt.wantVal)
			}
		})
	}
}

func TestBackupCommand(t *testing.T) {
	tests := []struct {
		name       string
		engine     string
		wantPrefix string
	}{
		{"postgres", "postgres", "pg_dump"},
		{"valkey", "valkey", "valkey_dump"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := getProfile(tt.engine)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			cmd := p.BackupCommand(&databasesv1alpha1.DatabaseCluster{})
			if len(cmd) == 0 {
				t.Fatal("backup command is empty")
			}
			if cmd[0] != tt.wantPrefix {
				t.Errorf("backup command[0] = %q, want %q", cmd[0], tt.wantPrefix)
			}
		})
	}
}
