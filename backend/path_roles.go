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
				"description": {
					Type:        framework.TypeString,
					Description: "Description of the role",
				},
				"vault_id": {
					Type:        framework.TypeString,
					Description: "Skyflow Vault ID for this role",
				},
				"account_id": {
					Type:        framework.TypeString,
					Description: "Skyflow Account ID for this role",
				},
				"scopes": {
					Type:        framework.TypeCommaStringSlice,
					Description: "List of scopes for the token",
				},
				"ttl": {
					Type:        framework.TypeDurationSecond,
					Description: "Token TTL (default: 3600s)",
					Default:     3600,
				},
				"max_ttl": {
					Type:        framework.TypeDurationSecond,
					Description: "Maximum token TTL (default: 3600s)",
					Default:     3600,
				},
				"credentials_file_path": {
					Type:        framework.TypeString,
					Description: "Override credentials file path for this role",
				},
				"credentials_json": {
					Type:        framework.TypeString,
					Description: "Override credentials JSON for this role",
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
			HelpDescription: "Create and manage roles for Skyflow token generation with specific permissions and settings.",
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
	if desc, ok := data.GetOk("description"); ok {
		role.Description = desc.(string)
	}

	if vaultID, ok := data.GetOk("vault_id"); ok {
		role.VaultID = vaultID.(string)
	}

	if accountID, ok := data.GetOk("account_id"); ok {
		role.AccountID = accountID.(string)
	}

	if scopes, ok := data.GetOk("scopes"); ok {
		role.Scopes = scopes.([]string)
	}

	if ttl, ok := data.GetOk("ttl"); ok {
		role.TTL = time.Duration(ttl.(int)) * time.Second
	}

	if maxTTL, ok := data.GetOk("max_ttl"); ok {
		role.MaxTTL = time.Duration(maxTTL.(int)) * time.Second
	}

	if credPath, ok := data.GetOk("credentials_file_path"); ok {
		role.CredentialsFilePath = credPath.(string)
		role.CredentialsJSON = ""
	}

	if credJSON, ok := data.GetOk("credentials_json"); ok {
		role.CredentialsJSON = credJSON.(string)
		role.CredentialsFilePath = ""
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

	// Don't return sensitive credentials
	responseData := map[string]interface{}{
		"name":        role.Name,
		"description": role.Description,
		"vault_id":    role.VaultID,
		"account_id":  role.AccountID,
		"scopes":      role.Scopes,
		"ttl":         int64(role.TTL.Seconds()),
		"max_ttl":     int64(role.MaxTTL.Seconds()),
		"tags":        role.Tags,
		"created_at":  role.CreatedAt.Format(time.RFC3339),
		"updated_at":  role.UpdatedAt.Format(time.RFC3339),
	}

	if role.CredentialsFilePath != "" {
		responseData["credentials_type"] = "file_path"
		responseData["has_credentials_override"] = true
	} else if role.CredentialsJSON != "" {
		responseData["credentials_type"] = "json"
		responseData["has_credentials_override"] = true
	} else {
		responseData["has_credentials_override"] = false
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
