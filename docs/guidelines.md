# Guidelines

Developer-facing reference for the Skyflow secrets plugin. This guide is intentionally scoped to API usage and the testing surface so engineers can build, verify, and ship integrations quickly.

---

## API Reference

All paths shown below assume a mount such as `skyflow/order/`, `skyflow/purchase/`, or `skyflow/payment/`. Replace the mount prefix to match your deployment.

### Configuration

**`POST {mount}/config`** — Store Skyflow service account credentials for the mount.

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `credentials_file_path` | string | conditional | Absolute path readable by Vault. |
| `credentials_json` | string | conditional | Inline JSON blob. |
| `description` | string | no | Free-form docs. |
| `tags` | []string | no | Use `product:order`, `env:prod`, etc. |
| `validate_credentials` | bool | no | Defaults to `true`. Set `false` to skip Skyflow validation (not recommended outside dev).

Exactly one of `credentials_file_path` or `credentials_json` must be supplied.

```bash
vault write skyflow/order/config \
  credentials_file_path="/etc/vault/creds/order-service.json" \
  description="Order platform credentials" \
  tags="product:order,env:prod"
```

### Roles

**`POST {mount}/roles/{name}`** — Create or update a role representing a downstream application (for example, `order-producer`, `purchase-consumer-portal`, `payment-risk-engine`).

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `role_ids` | []string | yes | Supply exactly one Skyflow role ID. |
| `description` | string | no | Purpose of the role. |
| `tags` | []string | no | Helpful for auditing and search. |

Additional verbs:
- **`LIST {mount}/roles`** — Enumerate roles for the mount.
- **`GET {mount}/roles/{name}`** — Read role definition.
- **`DELETE {mount}/roles/{name}`** — Remove role (irreversible).

```bash
vault write skyflow/payment/roles/payment-risk-engine \
  role_ids="skyflow-role-risk-001" \
  description="Risk engine read/write access" \
  tags="product:payment,app:risk"
```

### Token Issuance

**`GET {mount}/creds/{role}`** — Fetch a short-lived bearer token for the specified role.

Optional query/form field:

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `ctx` | string | no | Free-form context passed to Skyflow, e.g., `ctx="order:12345"`.

```bash
# Order service generating a producer token
vault read skyflow/order/creds/order-producer

# Payment app attaching context for traceability
vault read skyflow/payment/creds/payment-risk-engine ctx="txn:PAY-8934"
```

**Response body:**

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "token_type": "Bearer"
}
```

### Health

**`GET {mount}/health`** — Performs an internal check (storage access + Skyflow reachability). Useful for readiness probes.

### Error Surface

| Code | Cause |
|------|-------|
| 200 | Operation succeeded. |
| 400 | Validation failed (missing fields, invalid JSON, role has multiple IDs). |
| 403 | Vault policy denied the request. |
| 404 | Role or config missing. |
| 409 | Role already exists (when `POST` uses `?force=false`). |
| 500 | Internal plugin error. |
| 503 | Upstream Skyflow service unavailable or timed out. |

---

## Testing Playbook

### Command Suite

```bash
# Unit tests (all packages)
go test ./... -v

# With coverage summary
go test ./... -cover

# Generate HTML report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Data races
go test ./... -race

# Integration tests (needs Vault + Skyflow credentials)
VAULT_ADDR=http://127.0.0.1:8200 \
VAULT_TOKEN=root \
TEST_MODE=integration \
go test ./test/integration -v
```

### Package Responsibilities

```
backend/
├─ backend_test.go       # Mount factory, logical paths
├─ config_test.go        # Config validation + seal-wrap semantics
├─ role_test.go          # CRUD + role_id invariants
├─ path_config_test.go   # HTTP semantics for config operations
├─ path_roles_test.go    # Role lifecycle edge cases
├─ path_token_test.go    # Token issuance + context plumbing
└─ telemetry/
   ├─ config_test.go     # Telemetry toggles
   └─ traces_test.go     # Attribute coverage

test/integration/
├─ setup.go              # Harness bootstraps Vault + plugin
└─ token_test.go         # End-to-end order/purchase/payment flows
```

### Quality Gates

- **Coverage** — Maintain ≥80% line coverage overall and 100% for `path_token` and `config` packages.
- **Static analysis** — `golangci-lint run` must be clean before merging (enforced in CI).
- **Contract tests** — Integration suite must exercise at least one token request for order, purchase, and payment mounts to guarantee parity.

### Test Data Guidelines

- Keep fake Skyflow credentials under `testdata/` and never commit production secrets.
- When simulating roles, prefer names like `order-producer`, `purchase-consumer-portal`, `payment-risk-engine` so examples stay generic.
- Use context strings that resemble real workloads (`ctx="order:ORD-42"`) to exercise logging/telemetry code paths.

---

This guideline should be the single source of truth for how engineers interact with the plugin programmatically and how they prove changes through automated tests.
