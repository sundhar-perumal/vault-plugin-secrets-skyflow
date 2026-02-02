# Skyflow Vault Plugin - SRE CI/CD Guide

CI/CD for this plugin focuses on building reproducible binaries, validating integrations, and promoting releases safely across environments that host the order, purchase, and payment mounts. This guide documents the automation flow and the manual checkpoints required from SREs.

---

## Pipeline Blueprint

```
┌────────────┐   lint+unit   ┌─────────────┐   integ tests   ┌─────────────┐   package/sign   ┌─────────────┐   deploy
│  Git push  │ ─────────────▶│ Stage: Lint │ ───────────────▶│ Stage: Test │ ────────────────▶│ Stage: Art  │ ────────────────▶│ Stage: Deploy │
└────────────┘               └─────────────┘                 └─────────────┘                   └─────────────┘                   └────────────────┘
                                                                │                                                       │
                                                                ▼                                                       ▼
                                                         Coverage + race                                          Dev → Stg → Prod
```

### Stage Summary

| Stage | Purpose | Blocking Rules |
|-------|---------|----------------|
| **Lint** | `golangci-lint run` + formatting checks. | Must be clean; warnings fail the build. |
| **Unit** | `go test ./... -race -cover`. | Requires >=80% coverage. |
| **Integration** | `go test ./test/integration -tags=integration`. | Runs only when Skyflow sandbox secrets are available; failures block promotion. |
| **Artifact** | Produce Linux AMD64 binary + Docker image. Generate SHA256 + SBOM. | SHA mismatch halts pipeline. |
| **Deploy** | Publish binary to object storage, push Docker image, register plugin in Vault for dev/stg/prod. | Requires manual approval + health check gates. |

---

## Build & Test Stages

- **Lint job**
  - Command: `golangci-lint run --timeout=3m`
  - Cache the module download directory (`$GOMODCACHE`) to keep runtimes low.

- **Unit test job**
  - Commands:
    ```bash
    go test ./... -race -coverprofile=coverage.out
    go tool cover -func=coverage.out | tee coverage.txt
    ```
  - Upload `coverage.out` as an artifact; pipeline enforces >=80%.

- **Integration job**
  - Secrets required: `SKYFLOW_SANDBOX_JSON`, `VAULT_DEV_TOKEN`.
  - Commands:
    ```bash
    VAULT_ADDR=${VAULT_ADDR:-http://127.0.0.1:8200}
    VAULT_TOKEN=$VAULT_DEV_TOKEN TEST_MODE=integration go test ./test/integration -v
    ```
  - Job marked optional for forks but mandatory on protected branches.

---

## Artifact Packaging

1. **Build binary**
   ```bash
   make clean build            # outputs bin/skyflow-plugin
   sha256sum bin/skyflow-plugin > dist/skyflow-plugin.sha256
   ```
2. **Container image**
   ```bash
   docker build -t registry.example.com/infra/skyflow-plugin:${GIT_SHA} .
   docker push registry.example.com/infra/skyflow-plugin:${GIT_SHA}
   ```
3. **SBOM + signatures**
   ```bash
   syft packages bin/skyflow-plugin -o json > dist/skyflow-plugin.sbom.json
   cosign sign --key azure-kv://vault/skyflow bin/skyflow-plugin
   ```
4. **Release bundle**
   - Upload `bin/skyflow-plugin`, checksum, and SBOM to artifact storage.
   - Tag git release `vX.Y.Z` only after artifacts replicate.

---

## Deployment Automation

### 1. Dev rollout (order mount)

```bash
SHA=$(cat dist/skyflow-plugin.sha256 | cut -d' ' -f1)

  -sha256=$SHA \
  -command="skyflow-plugin" \
  -env="ENV=dev" \
  secret skyflow-svc

vault secrets enable -path=skyflow/order skyflow-svc || true
vault write skyflow/order/config credentials_json=@order-dev-sa.json
```

Verify: `vault read skyflow/order/health`.

### 2. Staging rollout (purchase mount)

```bash
vault plugin reload -plugin=skyflow-svc
vault secrets enable -path=skyflow/purchase skyflow-svc || true
vault write skyflow/purchase/config credentials_json=@purchase-stg-sa.json
```

Run smoke tests: `vault read skyflow/purchase/creds/purchase-producer`.

### 3. Production rollout (payment mount)

```bash
vault plugin reload -plugin=skyflow-svc
vault secrets enable -path=skyflow/payment skyflow-svc || true
vault write skyflow/payment/config credentials_file_path="/etc/vault/creds/payment.json"
```

Post-checks:
- `vault list skyflow/payment/roles`
- `vault read skyflow/payment/health`
- Telemetry: verify traces on OTLP collector (service name `skyflow-plugin`).

Promotion requires two-person approval plus confirmation that the previous environment ran for 24h without errors.

---

## Release Checklist

1. Pipeline green on `main` (lint/unit/integration).
2. Artifacts signed and stored with matching SHA.
3. Release notes include:
   - Commit range
   - Config/role schema changes
   - Required Vault version
4. Dev rollout complete; health endpoint returning 200.
5. Purchase mount updated in staging; integration smoke tests (`go test ./test/integration -run Purchase`) succeed.
6. Production CAB approval captured.
7. Payment mount updated; first token request logged with correct telemetry attributes.

---

## Rollback Plan

- Keep previous binary + checksum for two releases.
- To revert:
  ```bash
  vault plugin register -sha256=$(cat prev.sha256) -command=skyflow-plugin secret skyflow-svc
  vault plugin reload -plugin=skyflow-svc
  ```
- Re-run health checks for order, purchase, payment mounts.
- Update release log with reason + timeline.

---

## CI/CD Observability

| Signal | Location | Notes |
|--------|----------|-------|
| Pipeline metrics | GitHub Actions /build insights | Watch duration and failure rate per job. |
| Artifact integrity | Artifact storage + SHA file | SHA mismatch triggers alert. |
| Vault deploy | Vault audit logs | Ensure `plugin reload` + `secrets enable` recorded. |
| Telemetry | OTLP traces/metrics | Expect attributes `mount=order|purchase|payment`, `role`, `env`. |

Set alerts for:
- Two consecutive pipeline failures on `main`.
- Token latency p95 > 400ms after deployment.
- Missing telemetry for a mount for >30 minutes.

---

Following this runbook keeps the Skyflow secrets plugin release train predictable and auditable while supporting multiple business-critical mounts.


| Mount Path | Role | Application | Access |
