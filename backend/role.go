package backend

import (
	"fmt"
	"context"
	"time"
	"github.com/hashicorp/vault/sdk/logical"
)

// skyflowRole represents a role configuration for token generation
type skyflowRole struct {
	// Role identification
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`

	// Skyflow role IDs (mandatory) - passed to SDK for token generation
	RoleIDs []string `json:"role_ids"`

	// Metadata
	Tags      []string  `json:"tags,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// defaultRole returns a role with default values
func defaultRole(name string) *skyflowRole {
	now := time.Now()
	return &skyflowRole{
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// validate checks if the role configuration is valid
func (r *skyflowRole) validate() error {
	if r.Name == "" {
		return fmt.Errorf("role name is required")
	}

	// Role IDs - exactly one required (future: may support multiple)
	if len(r.RoleIDs) == 0 {
		return fmt.Errorf("role_ids is required")
	}
	if len(r.RoleIDs) > 1 {
		return fmt.Errorf("only one role_id is supported. for multiple roles please contact plugin admin")
	}

	return nil
}

// getRole retrieves a role from storage
func (b *skyflowBackend) getRole(ctx context.Context, s logical.Storage, name string) (*skyflowRole, error) {
	if name == "" {
		return nil, fmt.Errorf("role name is required")
	}

	entry, err := s.Get(ctx, "role/"+name)
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	if entry == nil {
		return nil, nil
	}

	role := &skyflowRole{}
	if err := entry.DecodeJSON(role); err != nil {
		return nil, fmt.Errorf("failed to decode role: %w", err)
	}

	return role, nil
}

// saveRole stores a role in Vault storage
func (b *skyflowBackend) saveRole(ctx context.Context, s logical.Storage, role *skyflowRole) error {
	if role.Name == "" {
		return fmt.Errorf("role name is required")
	}

	role.UpdatedAt = time.Now()

	entry, err := logical.StorageEntryJSON("role/"+role.Name, role)
	if err != nil {
		return fmt.Errorf("failed to create storage entry: %w", err)
	}

	if err := s.Put(ctx, entry); err != nil {
		return fmt.Errorf("failed to save role: %w", err)
	}

	return nil
}

// deleteRole removes a role from storage
func (b *skyflowBackend) deleteRole(ctx context.Context, s logical.Storage, name string) error {
	if name == "" {
		return fmt.Errorf("role name is required")
	}

	if err := s.Delete(ctx, "role/"+name); err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	return nil
}

// listRoles returns all role names
func (b *skyflowBackend) listRoles(ctx context.Context, s logical.Storage) ([]string, error) {
	roles, err := s.List(ctx, "role/")
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}

	return roles, nil
}
