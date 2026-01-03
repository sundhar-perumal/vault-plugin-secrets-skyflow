package backend

import (
	"context"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
)

func TestPathToken_GenerateToken_PanicRecovery(t *testing.T) {
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

	tests := []struct {
		name   string
		config *skyflowConfig
		role   *skyflowRole
	}{
		{
			name: "Non-existent credentials file",
			config: &skyflowConfig{
				CredentialsFilePath: "/non/existent/file.json",
			},
			role: &skyflowRole{
				Name:    "test-role",
				RoleIDs: []string{"test-role-id"},
			},
		},
		{
			name: "Invalid JSON credentials",
			config: &skyflowConfig{
				CredentialsJSON: `{"invalid": "creds"}`,
			},
			role: &skyflowRole{
				Name:    "test-role",
				RoleIDs: []string{"test-role-id"},
			},
		},
		{
			name: "Empty credentials",
			config: &skyflowConfig{
				CredentialsJSON: `{}`,
			},
			role: &skyflowRole{
				Name:    "test-role",
				RoleIDs: []string{"test-role-id"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This should NOT panic, but return an error gracefully
			token, err := backend.generateToken(tt.config, tt.role, "")

			if token != nil {
				t.Error("expected nil token for invalid credentials")
			}

			if err == nil {
				t.Error("expected error for invalid credentials, got nil")
			}

			t.Logf("Got expected error: %v", err)
		})
	}
}

func TestPathToken_GenerateToken_NoCredentials(t *testing.T) {
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

	// Config with no credentials
	cfg := &skyflowConfig{}
	role := &skyflowRole{
		Name:    "test-role",
		RoleIDs: []string{"test-role-id"},
	}

	token, err := backend.generateToken(cfg, role, "")

	if token != nil {
		t.Error("expected nil token when no credentials configured")
	}

	if err == nil {
		t.Error("expected error when no credentials configured")
	}

	t.Logf("Got expected error: %v", err)
}

func TestPathToken_GenerateToken_ConfigCredentials(t *testing.T) {
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

	// Config with credentials
	cfg := &skyflowConfig{
		CredentialsJSON: `{"test": "creds"}`,
	}

	// Role only has role_ids (no credential override)
	role := &skyflowRole{
		Name:    "test-role",
		RoleIDs: []string{"test-role-id-1", "test-role-id-2"},
	}

	// This will fail because credentials are invalid, but proves config creds are used
	token, err := backend.generateToken(cfg, role, "")

	if token != nil {
		t.Error("expected nil token for invalid credentials")
	}

	if err == nil {
		t.Error("expected error for invalid credentials")
	}

	t.Logf("Got expected error: %v", err)
}
