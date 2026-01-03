package backend

import (
	"context"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

// pathRoles returns the path configuration for managing roles
func pathRoles(b *skyflowBackend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "roles/?$",

			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ListOperation: &framework.PathOperation{
					Callback: b.pathRoleList,
					Summary:  "List all configured roles.",
				},
			},

			HelpSynopsis:    "List configured roles.",
			HelpDescription: "List all roles configured for Skyflow token generation.",
		},
		{
			Pattern: "roles/" + framework.GenericNameRegex("name"),

			Fields: map[string]*framework.FieldSchema{
				"name": {
					Type:        framework.TypeString,
					Description: "Name of the role",
					Required:    true,
				},
				"role_ids": {
					Type:        framework.TypeCommaStringSlice,
					Description: "Skyflow role IDs for token generation (required)",
					Required:    true,
				},
				"description": {
					Type:        framework.TypeString,
					Description: "Description of the role",
				},
				"tags": {
					Type:        framework.TypeCommaStringSlice,
					Description: "Tags for organizing roles",
				},
			},

			ExistenceCheck: b.pathRoleExistenceCheck,

			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.pathRoleWrite,
					Summary:  "Create a new role for Skyflow token generation.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.pathRoleWrite,
					Summary:  "Update an existing role.",
				},
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.pathRoleRead,
					Summary:  "Read a role configuration.",
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.pathRoleDelete,
					Summary:  "Delete a role.",
				},
			},

			HelpSynopsis:    "Manage Skyflow roles.",
			HelpDescription: "Create and manage roles for Skyflow token generation with specific Skyflow role IDs.",
		},
	}
}

// pathRoleExistenceCheck checks if role exists
func (b *skyflowBackend) pathRoleExistenceCheck(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
	name := data.Get("name").(string)
	role, err := b.getRole(ctx, req.Storage, name)
	if err != nil {
		return false, err
	}

	return role != nil, nil
}

// pathRoleList lists all roles
func (b *skyflowBackend) pathRoleList(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	roles, err := b.listRoles(ctx, req.Storage)
	if err != nil {
		return nil, err
	}

	return logical.ListResponse(roles), nil
}

// pathRoleWrite handles create and update operations for roles
func (b *skyflowBackend) pathRoleWrite(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	name := data.Get("name").(string)
	if name == "" {
		return logical.ErrorResponse("role name is required"), nil
	}

	operation := "create"
	if req.Operation == logical.UpdateOperation {
		operation = "update"
	}

	// Load existing role or create new one
	role := defaultRole(name)
	if req.Operation == logical.UpdateOperation {
		existingRole, err := b.getRole(ctx, req.Storage, name)
		if err != nil {
			return nil, err
		}
		if existingRole != nil {
			role = existingRole
		}
	}

	// Update fields from request
	if roleIDs, ok := data.GetOk("role_ids"); ok {
		role.RoleIDs = roleIDs.([]string)
	}

	if desc, ok := data.GetOk("description"); ok {
		role.Description = desc.(string)
	}

	if tags, ok := data.GetOk("tags"); ok {
		role.Tags = tags.([]string)
	}

	// Validate role
	if err := role.validate(); err != nil {
		return logical.ErrorResponse("invalid role: %s", err.Error()), nil
	}

	// Save role
	if err := b.saveRole(ctx, req.Storage, role); err != nil {
		return nil, err
	}

	// Emit telemetry
	if b.emitter != nil {
		b.emitter.EmitRoleWrite(ctx, name, operation, true)
	}

	b.Logger().Info("role saved", "name", name, "operation", req.Operation)

	return nil, nil
}

// pathRoleRead handles read operations for roles
func (b *skyflowBackend) pathRoleRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	name := data.Get("name").(string)

	role, err := b.getRole(ctx, req.Storage, name)
	if err != nil {
		return nil, err
	}

	if role == nil {
		return nil, nil
	}

	responseData := map[string]interface{}{
		"name":        role.Name,
		"role_ids":    role.RoleIDs,
		"description": role.Description,
		"tags":        role.Tags,
		"created_at":  role.CreatedAt.Format(time.RFC3339),
		"updated_at":  role.UpdatedAt.Format(time.RFC3339),
	}

	return &logical.Response{
		Data: responseData,
	}, nil
}

// pathRoleDelete handles delete operations for roles
func (b *skyflowBackend) pathRoleDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	name := data.Get("name").(string)

	if err := b.deleteRole(ctx, req.Storage, name); err != nil {
		return nil, err
	}

	b.Logger().Info("role deleted", "name", name)

	return nil, nil
}
