# Skyflow Vault Plugin - Guidelines

API reference, security controls, and operational guidelines for the Skyflow Vault Plugin.

## Table of Contents

- [Authentication](#authentication)
- [API Reference](#api-reference)
- [Security Controls](#security-controls)
- [Vault Policies](#vault-policies)
- [Operational Guidelines](#operational-guidelines)
- [Incident Response](#incident-response)

---

## Authentication

All API requests must be authenticated using a valid Vault token:

```bash
curl \
    --header "X-Vault-Token: $VAULT_TOKEN" \
    $VAULT_ADDR/v1/skyflow/...
```

---

## API Reference

### Configuration Endpoints

#### Create/Update Configuration

**Endpoint:** `POST/PUT /skyflow/config`

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `credentials_file_path` | string | conditional | Path to Skyflow credentials JSON file |
| `credentials_json` | string | conditional | Skyflow credentials as JSON string |
| `max_retries` | integer | no | Maximum API retry attempts (0-10). Default: 3 |
| `request_timeout` | integer | no | Request timeout in seconds (1-300). Default: 30 |
| `description` | string | no | Configuration description |

**Validation:**
- Must provide exactly one of `credentials_file_path` or `credentials_json`
- `credentials_json` must be valid JSON if provided

**Example:**
```bash
vault write skyflow/config \
    credentials_file_path="/etc/skyflow/credentials.json" \
    max_retries=3 \
    request_timeout=30 \
    description="Production configuration"
```

#### Read Configuration

**Endpoint:** `GET /skyflow/config`

Returns configuration with credentials masked.

```bash
vault read skyflow/config
```

**Response:**
```json
{
  "data": {
    "credentials_configured": true,
    "credentials_type": "file_path",
    "credentials_file_path": "/etc/skyflow/credentials.json",
    "max_retries": 3,
    "request_timeout": 30
  }
}
```

#### Delete Configuration

**Endpoint:** `DELETE /skyflow/config`

```bash
vault delete skyflow/config
```

---

### Role Endpoints

#### List Roles

**Endpoint:** `LIST /skyflow/roles`

```bash
vault list skyflow/roles
```

#### Create/Update Role

**Endpoint:** `POST/PUT /skyflow/roles/:name`

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `description` | string | no | Role description |
| `vault_id` | string | no | Skyflow Vault ID |
| `account_id` | string | no | Skyflow Account ID |
| `scopes` | []string | no | Permission scopes |
| `ttl` | integer | no | Token TTL in seconds. Default: 3600 |
| `max_ttl` | integer | no | Maximum token TTL. Default: 3600 |
| `credentials_file_path` | string | no | Override credentials file path |
| `credentials_json` | string | no | Override credentials JSON |
| `tags` | []string | no | Organization tags |

**Example:**
```bash
vault write skyflow/roles/my-app \
    description="My application role" \
    vault_id="vault123" \
    ttl=3600 \
    tags="production,backend"
```

#### Read Role

**Endpoint:** `GET /skyflow/roles/:name`

```bash
vault read skyflow/roles/my-app
```

#### Delete Role

**Endpoint:** `DELETE /skyflow/roles/:name`

```bash
vault delete skyflow/roles/my-app
```

---

### Token Generation

#### Generate Token

**Endpoint:** `GET /skyflow/creds/:name`

Generates a fresh Skyflow bearer token for the specified role.

```bash
vault read skyflow/creds/my-app
```

**Response:**
```json
{
  "data": {
    "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
    "token_type": "Bearer",
    "expires_in": 3600,
    "expires_at": "2025-12-30T15:00:00Z"
  }
}
```

---

### Health and Metrics

#### Health Check

**Endpoint:** `GET /skyflow/health`

```bash
vault read skyflow/health
```

#### Metrics

**Endpoint:** `GET /skyflow/metrics`

```bash
vault read skyflow/metrics
```

---

## Security Controls

### Credential Protection

| Control | Implementation |
|---------|----------------|
| **Seal-wrap encryption** | Credentials encrypted with Vault's seal key |
| **Credential masking** | Credentials never returned in API responses |
| **Short-lived tokens** | Maximum TTL: 3600 seconds (1 hour) |

### Circuit Breaker

Protects against Skyflow API failures:

| Setting | Value |
|---------|-------|
| Max failures | 5 |
| Reset timeout | 60 seconds |
| States | closed → open → half-open → closed |

### Telemetry

OpenTelemetry integration for observability:

| Feature | Description |
|---------|-------------|
| **Traces** | Distributed tracing for token generation |
| **Metrics** | Token generation counts, latency histograms |
| **Configurable** | Enable/disable via environment variables |

---

## Vault Policies

### Application Policy (Read-Only)

```hcl
path "skyflow/creds/*" {
    capabilities = ["read"]
}
```

### Admin Policy (Full Access)

```hcl
# Configuration management
path "skyflow/config" {
    capabilities = ["create", "read", "update", "delete"]
}

# Role management
path "skyflow/roles/*" {
    capabilities = ["create", "read", "update", "delete", "list"]
}

# Token generation
path "skyflow/creds/*" {
    capabilities = ["read"]
}

# Health and metrics
path "skyflow/health" {
    capabilities = ["read"]
}
path "skyflow/metrics" {
    capabilities = ["read"]
}
```

---

## Operational Guidelines

### Deployment Checklist

- [ ] Vault running with TLS enabled
- [ ] Audit backend enabled and monitored
- [ ] Least-privilege policies applied
- [ ] Network segmentation in place
- [ ] Telemetry configured for observability

### Configuration Checklist

- [ ] Credentials stored via seal-wrap
- [ ] Appropriate TTLs configured
- [ ] Circuit breaker configured
- [ ] Telemetry endpoints configured

### Using Tokens in Applications

```bash
# Retrieve token
TOKEN_RESPONSE=$(vault read -format=json skyflow/creds/my-app)
ACCESS_TOKEN=$(echo $TOKEN_RESPONSE | jq -r '.data.access_token')

# Use token with Skyflow API
curl -H "Authorization: Bearer $ACCESS_TOKEN" \
     https://api.skyflowapis.com/v1/vaults/abc123/records
```

### Credential Rotation

```bash
# Update configuration with new credentials
vault write skyflow/config \
    credentials_json='{"clientID":"new-...","privateKey":"new-..."}'

# Next token request uses new credentials
vault read skyflow/creds/my-app
```

---

## Incident Response

### Credential Compromise

1. **Immediate**: Delete compromised configuration
   ```bash
   vault delete skyflow/config
   ```

2. **Rotate**: Generate new Skyflow service credentials

3. **Reconfigure**: Update Vault with new credentials
   ```bash
   vault write skyflow/config credentials_json='...'
   ```

4. **Audit**: Review audit logs for unauthorized access

### Suspicious Activity

1. **Monitor**: Check circuit breaker state and metrics
   ```bash
   vault read skyflow/health
   vault read skyflow/metrics
   ```

2. **Investigate**: Review Vault audit logs

3. **Restrict**: Temporarily disable role if needed
   ```bash
   vault delete skyflow/roles/suspicious-role
   ```

---

## Error Responses

### Standard Error Format

```json
{
  "errors": ["error message here"]
}
```

### Common HTTP Status Codes

| Status Code | Description |
|-------------|-------------|
| `200 OK` | Successful read operation |
| `204 No Content` | Successful write/delete operation |
| `400 Bad Request` | Invalid request parameters |
| `404 Not Found` | Resource not found |
| `500 Internal Server Error` | Server-side error |
| `503 Service Unavailable` | Skyflow API unreachable |

