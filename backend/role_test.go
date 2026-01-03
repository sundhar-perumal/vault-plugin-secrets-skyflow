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
				RoleIDs: []string{"role-id-1"},
			},
			wantError: false,
		},
		{
			name: "Empty name",
			role: &skyflowRole{
				Name:    "",
				RoleIDs: []string{"role-id-1"},
			},
			wantError: true,
			errorMsg:  "role name is required",
		},
		{
			name: "Missing role_ids",
			role: &skyflowRole{
				Name: "test-role",
			},
			wantError: true,
			errorMsg:  "role_ids is required",
		},
		{
			name: "Empty role_ids array",
			role: &skyflowRole{
				Name:    "test-role",
				RoleIDs: []string{},
			},
			wantError: true,
			errorMsg:  "role_ids is required",
		},
		{
			name: "Multiple role_ids not allowed",
			role: &skyflowRole{
				Name:    "test-role",
				RoleIDs: []string{"role-id-1", "role-id-2", "role-id-3"},
			},
			wantError: true,
			errorMsg:  "only one role_id is supported",
		},
		{
			name: "Valid role with description and tags",
			role: &skyflowRole{
				Name:        "test-role",
				RoleIDs:     []string{"role-id-1"},
				Description: "Test description",
				Tags:        []string{"tag1", "tag2"},
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
			RoleIDs:     []string{"skyflow-role-1", "skyflow-role-2"},
			Description: "Test role description",
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

		if len(role.RoleIDs) != len(testRole.RoleIDs) {
			t.Errorf("expected %d role_ids, got %d", len(testRole.RoleIDs), len(role.RoleIDs))
		}

		if role.Description != testRole.Description {
			t.Errorf("expected description '%s', got '%s'", testRole.Description, role.Description)
		}

		if len(role.Tags) != len(testRole.Tags) {
			t.Errorf("expected %d tags, got %d", len(testRole.Tags), len(role.Tags))
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
			role := &skyflowRole{
				Name:    name,
				RoleIDs: []string{"role-id-1"},
			}
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

	role := &skyflowRole{
		Name:    "timestamp-test",
		RoleIDs: []string{"role-id-1"},
	}
	originalUpdatedAt := role.UpdatedAt

	// Wait a bit and save
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
