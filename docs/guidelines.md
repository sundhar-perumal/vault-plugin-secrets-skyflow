# Guidelines

API reference, security, and testing for developers.

---

## API Reference

### Config Endpoint

**`POST /skyflow/config`** - Configure backend credentials

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `credentials_file_path` | string | conditional | Path to SA JSON file |
| `credentials_json` | string | conditional | SA JSON inline |
| `description` | string | no | Description |
| `tags` | []string | no | Organization tags |
| `validate_credentials` | bool | no | Test credentials (default: true) |

> Provide exactly one of `credentials_file_path` or `credentials_json`

```bash
vault write skyflow/insurance/config \
  credentials_file_path="/etc/vault/creds/sa.json" \
  description="Insurance credentials" \
  tags="production,insurance"
```

---

### Role Endpoints

**`POST /skyflow/roles/:name`** - Create/update role

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `role_ids` | []string | **yes** | Skyflow role ID (exactly 1) |
| `description` | string | no | Role description |
| `tags` | []string | no | Organization tags |

```bash
vault write skyflow/insurance/roles/producer \
  role_ids="skyflow-role-abc123" \
  description="Producer role" \
  tags="producer,insurance"
```

**`LIST /skyflow/roles`** - List all roles
**`GET /skyflow/roles/:name`** - Read role
**`DELETE /skyflow/roles/:name`** - Delete role

---

### Token Endpoint

**`GET /skyflow/creds/:name`** - Generate bearer token

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `ctx` | string | no | Context data for token |

```bash
# Basic
vault read skyflow/insurance/creds/producer

# With context
vault read skyflow/insurance/creds/producer ctx="user:12345"
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "token_type": "Bearer"
}
```

---

### Health Endpoint

**`GET /skyflow/health`** - Plugin health status

---

## Security

### Credential Protection

| Control | Implementation |
|---------|----------------|
| Storage | Vault seal-wrap encryption |
| API Response | Credentials never exposed |
| Tokens | Short-lived JWT (max 1hr) |

### Vault Policies

**Application (token read only):**
```hcl
path "skyflow/+/creds/*" {
  capabilities = ["read"]
}
```

**Admin (full access):**
```hcl
path "skyflow/+/config" {
  capabilities = ["create", "read", "update", "delete"]
}
path "skyflow/+/roles/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
path "skyflow/+/creds/*" {
  capabilities = ["read"]
}
path "skyflow/+/health" {
  capabilities = ["read"]
}
```

### Incident Response

**Credential compromise:**
```bash
vault delete skyflow/insurance/config   # Remove immediately
# Rotate in Skyflow console
vault write skyflow/insurance/config credentials_json='...'  # Reconfigure
```

---

## Testing

### Run Tests

```bash
# All tests
go test ./... -v

# With coverage
go test ./... -cover

# Coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Race detection
go test ./... -race

# Integration (requires credentials)
TEST_MODE=dev go test ./test/... -v
```

### Test Structure

```
backend/
├── backend_test.go       # Factory, lifecycle
├── config_test.go        # Config validation
├── role_test.go          # Role CRUD
├── path_token_test.go    # Token generation
└── telemetry/
    ├── config_test.go
    ├── emitter_test.go
    └── noop_test.go

test/integration/
├── setup.go              # Test harness
└── token_test.go         # E2E token tests
```

### Coverage Target

| Metric | Goal |
|--------|------|
| Line coverage | >80% |
| Critical paths | 100% |

---

## Error Codes

| Status | Meaning |
|--------|---------|
| 200 | Success |
| 400 | Invalid request |
| 404 | Role/config not found |
| 500 | Internal error |
| 503 | Skyflow API unavailable |
