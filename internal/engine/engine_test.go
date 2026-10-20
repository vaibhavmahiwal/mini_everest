package engine

import (
	"strings"
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
			p, err := For(tt.engine)
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

func TestBackupEnv(t *testing.T) {
	tests := []struct {
		name    string
		engine  string
		wantKey string
	}{
		{"postgres", "postgres", "PGPASSWORD"},
		{"valkey", "valkey", "VALKEY_PASSWORD"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := For(tt.engine)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			env := p.BackupEnv(&databasesv1alpha1.DatabaseCluster{})
			if len(env) == 0 {
				t.Fatal("backup env is empty")
			}
			if env[0].Name != tt.wantKey {
				t.Errorf("backup env name = %q, want %q", env[0].Name, tt.wantKey)
			}
		})
	}
}

func TestBackupCommand(t *testing.T) {
	tests := []struct {
		name         string
		engine       string
		wantContains string
	}{
		{"postgres", "postgres", "pg_dump"},
		{"valkey", "valkey", "valkey-cli"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := For(tt.engine)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			cmd := p.BackupCommand(&databasesv1alpha1.DatabaseCluster{})
			if len(cmd) < 3 || cmd[0] != "/bin/sh" || cmd[1] != "-c" {
				t.Fatalf("backup command = %v, want [/bin/sh -c ...]", cmd)
			}
			if !strings.Contains(cmd[2], tt.wantContains) {
				t.Errorf("backup command %q does not contain %q", cmd[2], tt.wantContains)
			}
		})
	}
}

func TestDataMountPath(t *testing.T) {
	tests := []struct {
		engine string
		want   string
	}{
		{"postgres", "/var/lib/postgresql/data"},
		{"valkey", "/data"},
	}
	for _, tt := range tests {
		t.Run(tt.engine, func(t *testing.T) {
			p, err := For(tt.engine)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := p.DataMountPath(); got != tt.want {
				t.Errorf("mount path = %q, want %q", got, tt.want)
			}
		})
	}
}
