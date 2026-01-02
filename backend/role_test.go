package backend

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/vault/sdk/logical"
)

func TestRole_DefaultRole(t *testing.T) {
	role := defaultRole("test-role")

	if role.Name != "test-role" {
		t.Errorf("expected name 'test-role', got '%s'", role.Name)
	}

	if role.TTL != 3600*time.Second {
		t.Errorf("expected default TTL 3600s, got %v", role.TTL)
	}

	if role.MaxTTL != 3600*time.Second {
		t.Errorf("expected default MaxTTL 3600s, got %v", role.MaxTTL)
	}

	if role.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}

	if role.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestRole_Validate(t *testing.T) {
	tests := []struct {
		name      string
		role      *skyflowRole
		wantError bool
		errorMsg  string
	}{
		{
			name: "Valid role",
			role: &skyflowRole{
				Name:    "test-role",
				TTL:     3600 * time.Second,
				MaxTTL:  3600 * time.Second,
				VaultID: "vault123",
			},
			wantError: false,
		},
		{
			name: "Empty name",
			role: &skyflowRole{
				Name:   "",
				TTL:    3600 * time.Second,
				MaxTTL: 3600 * time.Second,
			},
			wantError: true,
			errorMsg:  "role name is required",
		},
		{
			name: "Negative TTL",
			role: &skyflowRole{
				Name:   "test-role",
				TTL:    -1 * time.Second,
				MaxTTL: 3600 * time.Second,
			},
			wantError: true,
			errorMsg:  "ttl must be non-negative",
		},
		{
			name: "Negative MaxTTL",
			role: &skyflowRole{
				Name:   "test-role",
				TTL:    3600 * time.Second,
				MaxTTL: -1 * time.Second,
			},
			wantError: true,
			errorMsg:  "max_ttl must be non-negative",
		},
		{
			name: "TTL exceeds MaxTTL",
			role: &skyflowRole{
				Name:   "test-role",
				TTL:    7200 * time.Second,
				MaxTTL: 3600 * time.Second,
			},
			wantError: true,
			errorMsg:  "ttl cannot exceed max_ttl",
		},
		{
			name: "Both credentials provided",
			role: &skyflowRole{
				Name:                "test-role",
				TTL:                 3600 * time.Second,
				MaxTTL:              3600 * time.Second,
				CredentialsFilePath: "/path/to/creds.json",
				CredentialsJSON:     `{"key": "value"}`,
			},
			wantError: true,
			errorMsg:  "only one of credentials_file_path or credentials_json can be provided",
		},
		{
			name: "Valid role with file path override",
			role: &skyflowRole{
				Name:                "test-role",
				TTL:                 3600 * time.Second,
				MaxTTL:              3600 * time.Second,
				CredentialsFilePath: "/path/to/creds.json",
			},
			wantError: false,
		},
		{
			name: "Valid role with JSON override",
			role: &skyflowRole{
				Name:            "test-role",
				TTL:             3600 * time.Second,
				MaxTTL:          3600 * time.Second,
				CredentialsJSON: `{"key": "value"}`,
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.role.validate()
			if tt.wantError {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRole_GetSaveDelete(t *testing.T) {
	ctx := context.Background()
	storage := &logical.InmemStorage{}

	config := &logical.BackendConfig{
		Logger:      nil,
		System:      &logical.StaticSystemView{},
		StorageView: storage,
	}

	b, err := Factory(ctx, config)
	if err != nil {
		t.Fatalf("unable to create backend: %v", err)
	}

	backend := b.(*skyflowBackend)

	t.Run("Get non-existent role", func(t *testing.T) {
		role, err := backend.getRole(ctx, storage, "nonexistent")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if role != nil {
			t.Error("expected nil role for non-existent entry")
		}
	})

	t.Run("Get with empty name", func(t *testing.T) {
		_, err := backend.getRole(ctx, storage, "")
		if err == nil {
			t.Error("expected error for empty role name")
		}
	})

	t.Run("Save and get role", func(t *testing.T) {
		testRole := &skyflowRole{
			Name:        "test-role",
			Description: "Test role description",
			VaultID:     "vault123",
			AccountID:   "account456",
			Scopes:      []string{"read", "write"},
			TTL:         1800 * time.Second,
			MaxTTL:      3600 * time.Second,
			Tags:        []string{"test", "dev"},
		}

		err := backend.saveRole(ctx, storage, testRole)
		if err != nil {
			t.Fatalf("failed to save role: %v", err)
		}

		role, err := backend.getRole(ctx, storage, "test-role")
		if err != nil {
			t.Fatalf("failed to get role: %v", err)
		}

		if role == nil {
			t.Fatal("role should not be nil")
		}

		if role.Name != testRole.Name {
			t.Errorf("expected name '%s', got '%s'", testRole.Name, role.Name)
		}

		if role.VaultID != testRole.VaultID {
			t.Errorf("expected vault_id '%s', got '%s'", testRole.VaultID, role.VaultID)
		}

		if len(role.Scopes) != len(testRole.Scopes) {
			t.Errorf("expected %d scopes, got %d", len(testRole.Scopes), len(role.Scopes))
		}

		if role.TTL != testRole.TTL {
			t.Errorf("expected TTL %v, got %v", testRole.TTL, role.TTL)
		}
	})

	t.Run("Delete role", func(t *testing.T) {
		err := backend.deleteRole(ctx, storage, "test-role")
		if err != nil {
			t.Fatalf("failed to delete role: %v", err)
		}

		role, err := backend.getRole(ctx, storage, "test-role")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if role != nil {
			t.Error("role should be nil after deletion")
		}
	})

	t.Run("Delete with empty name", func(t *testing.T) {
		err := backend.deleteRole(ctx, storage, "")
		if err == nil {
			t.Error("expected error for empty role name")
		}
	})
}

func TestRole_ListRoles(t *testing.T) {
	ctx := context.Background()
	storage := &logical.InmemStorage{}

	config := &logical.BackendConfig{
		Logger:      nil,
		System:      &logical.StaticSystemView{},
		StorageView: storage,
	}

	b, err := Factory(ctx, config)
	if err != nil {
		t.Fatalf("unable to create backend: %v", err)
	}

	backend := b.(*skyflowBackend)

	t.Run("List empty roles", func(t *testing.T) {
		roles, err := backend.listRoles(ctx, storage)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(roles) != 0 {
			t.Errorf("expected 0 roles, got %d", len(roles))
		}
	})

	t.Run("List multiple roles", func(t *testing.T) {
		// Save some roles
		for _, name := range []string{"role1", "role2", "role3"} {
			role := defaultRole(name)
			if err := backend.saveRole(ctx, storage, role); err != nil {
				t.Fatalf("failed to save role %s: %v", name, err)
			}
		}

		roles, err := backend.listRoles(ctx, storage)
		if err != nil {
			t.Fatalf("failed to list roles: %v", err)
		}

		if len(roles) != 3 {
			t.Errorf("expected 3 roles, got %d", len(roles))
		}
	})
}

func TestRole_UpdateTimestamp(t *testing.T) {
	ctx := context.Background()
	storage := &logical.InmemStorage{}

	config := &logical.BackendConfig{
		Logger:      nil,
		System:      &logical.StaticSystemView{},
		StorageView: storage,
	}

	b, err := Factory(ctx, config)
	if err != nil {
		t.Fatalf("unable to create backend: %v", err)
	}

	backend := b.(*skyflowBackend)

	role := defaultRole("timestamp-test")
	originalUpdatedAt := role.UpdatedAt

	// Wait a bit and save again
	time.Sleep(10 * time.Millisecond)

	err = backend.saveRole(ctx, storage, role)
	if err != nil {
		t.Fatalf("failed to save role: %v", err)
	}

	savedRole, err := backend.getRole(ctx, storage, "timestamp-test")
	if err != nil {
		t.Fatalf("failed to get role: %v", err)
	}

	if !savedRole.UpdatedAt.After(originalUpdatedAt) {
		t.Error("UpdatedAt should be updated on save")
	}
}
