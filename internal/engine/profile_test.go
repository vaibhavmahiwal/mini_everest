package engine

import "testing"

func TestGetProfile(t *testing.T) {
	tests := []struct {
		name      string
		engine    string
		wantType  string
		wantImage string
		wantPort  int32
	}{
		{"postgresql", "postgresql", "postgres", "postgres:latest", 5432},
		{"postgres alias", "postgres", "postgres", "postgres:latest", 5432},
		{"valkey", "valkey", "valkey", "valkey:latest", 6379},
		{"unsupported", "mysql", "err", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := getProfile(tt.engine)
			if tt.wantType == "err" {
				if err == nil {
					t.Errorf("expected error for engine %q, but got nil", tt.engine)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for engine %q: %v", tt.engine, err)
			}
			if got := p.ContainerImage(); got != tt.wantImage {
				t.Errorf("image = %q, want %q", got, tt.wantImage)
			}
			if got := p.ContainerPort(); got != tt.wantPort {
				t.Errorf("port = %d, want %d", got, tt.wantPort)
			}
		})
	}
}
