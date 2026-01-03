# Skyflow Vault Plugin - SRE Operations Guide

## Quick Reference

| Item | Value |
|------|-------|
| Plugin Name | `skyflow-plugin` |
| Binary Location | `/etc/vault/plugins/skyflow-plugin` |
| Required Vault Version | 1.16.0+ |

---

## 1. Build & Install

```bash
make build                     # Build plugin
make install register enable   # Install to Vault
```

---

## 2. Docker Deployment

```bash
docker build -t skyflow-plugin:v1.0.0 .
docker run -d -p 8200:8200 skyflow-plugin:v1.0.0

export VAULT_ADDR=http://localhost:8200
export VAULT_TOKEN=root
```

---

## 3. Multi-Mount & Multi-App Architecture

```
vault/
├── skyflow/insurance/              # Mount: Insurance BU
│   ├── config                      # Skyflow credentials
│   └── roles/
│       ├── producer                # Insurance → writes PII
│       ├── consumer-admin-portal   # Admin Portal → reads PII
│       └── consumer-nxt            # NXT App → reads PII
│
├── skyflow/lending/                # Mount: Lending BU
│   ├── config                      # Skyflow credentials
│   └── roles/
│       ├── producer                # Lending → writes PII
│       ├── consumer-admin-portal   # Admin Portal → reads PII
│       └── consumer-nxt            # NXT App → reads PII
│
└── skyflow/clcm/                   # Mount: CLCM BU
    ├── config                      # Skyflow credentials
    └── roles/
        ├── producer                # CLCM → writes PII
        ├── consumer-insurance      # Insurance App → reads PII
        ├── consumer-lending        # Lending App → reads PII
        ├── consumer-portfolio      # Portfolio App → reads PII
        └── consumer-cns            # CNS App → reads PII
```

---

## 4. Configuration Commands

### Step 1: Register Plugin & Enable Mounts

```bash
# Register plugin (once)
SHA256=$(sha256sum /etc/vault/plugins/skyflow-plugin | cut -d' ' -f1)
vault plugin register -sha256=$SHA256 -command=skyflow-plugin secret skyflow-plugin

# Enable mount paths
vault secrets enable -path=skyflow/insurance skyflow-plugin
vault secrets enable -path=skyflow/lending skyflow-plugin
```

### Step 2: Configure Credentials

```bash
# Insurance BU
vault write skyflow/insurance/config \
  credentials_file_path="/etc/vault/creds/insurance-sa.json" \
  description="Insurance Skyflow credentials" \
  tags="production,insurance"

# Lending BU
vault write skyflow/lending/config \
  credentials_file_path="/etc/vault/creds/lending-sa.json" \
  description="Lending Skyflow credentials" \
  tags="production,lending"
```

### Step 3: Create Roles

```bash
# ================================ INSURANCE BU ================================

# Producer (Insurance app writes PII)
vault write skyflow/insurance/roles/producer \
  role_ids="skyflow-insurance-write-role" \
  description="Insurance producer - writes PII" \
  tags="producer,insurance"

# Consumer: Admin Portal (reads PII)
vault write skyflow/insurance/roles/consumer-admin-portal \
  role_ids="skyflow-insurance-admin-read-role" \
  description="Admin Portal - reads Insurance PII" \
  tags="consumer,admin-portal,insurance"

# Consumer: NXT (reads PII)
vault write skyflow/insurance/roles/consumer-nxt \
  role_ids="skyflow-insurance-nxt-read-role" \
  description="NXT App - reads Insurance PII" \
  tags="consumer,nxt,insurance"

# ================================ LENDING BU ================================

# Producer (Lending app writes PII)
vault write skyflow/lending/roles/producer \
  role_ids="skyflow-lending-write-role" \
  description="Lending producer - writes PII" \
  tags="producer,lending"

# Consumer: Admin Portal (reads PII)
vault write skyflow/lending/roles/consumer-admin-portal \
  role_ids="skyflow-lending-admin-read-role" \
  description="Admin Portal - reads Lending PII" \
  tags="consumer,admin-portal,lending"

# Consumer: NXT (reads PII)
vault write skyflow/lending/roles/consumer-nxt \
  role_ids="skyflow-lending-nxt-read-role" \
  description="NXT App - reads Lending PII" \
  tags="consumer,nxt,lending"
```

### Step 4: Generate Tokens

```bash
# Insurance - Producer token
vault read skyflow/insurance/creds/producer

# Insurance - Admin Portal consumer token
vault read skyflow/insurance/creds/consumer-admin-portal

# Insurance - NXT consumer token
vault read skyflow/insurance/creds/consumer-nxt

# Lending - Producer token
vault read skyflow/lending/creds/producer

# Lending - Admin Portal consumer token
vault read skyflow/lending/creds/consumer-admin-portal

# With context data
vault read skyflow/insurance/creds/producer ctx="txn:INS-12345"
```

---

## 5. Summary Table

| Mount Path | Role | Application | Access |
|------------|------|-------------|--------|
| `skyflow/insurance` | `producer` | Insurance | Write |
| `skyflow/insurance` | `consumer-admin-portal` | Admin Portal | Read |
| `skyflow/insurance` | `consumer-nxt` | NXT | Read |
| `skyflow/lending` | `producer` | Lending | Write |
| `skyflow/lending` | `consumer-admin-portal` | Admin Portal | Read |
| `skyflow/lending` | `consumer-nxt` | NXT | Read |

---

## 6. Common Commands

| Task | Command |
|------|---------|
| List mounts | `vault secrets list` &#124; `grep skyflow` |
| List Insurance roles | `vault list skyflow/insurance/roles` |
| List Lending roles | `vault list skyflow/lending/roles` |
| Read role config | `vault read skyflow/insurance/roles/producer` |
| Health check | `vault read skyflow/insurance/health` |
| Delete role | `vault delete skyflow/insurance/roles/consumer-nxt` |
| Disable mount | `vault secrets disable skyflow/insurance` |

---

## 7. Environment Variables (Telemetry)

### Master Switch

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `ENV` | Set to `dev` to **disable all telemetry** | - | `ENV=dev` |
| `TELEMETRY_ENABLED` | Master on/off switch | `true` | `TELEMETRY_ENABLED=false` |

### Example: Production Configuration

```bash
# Service identity
export OTEL_SERVICE_NAME=<actual-application-name>
export SERVICE_NAMESPACE=skyflow-vault-plugin
export ENV=production

# Enable telemetry
export TELEMETRY_ENABLED=true
export TELEMETRY_TRACES_ENABLED=true
export TELEMETRY_METRICS_ENABLED=true

# OTEL Collector endpoints
export OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=http://otel-collector:4318/v1/traces
export OTEL_EXPORTER_OTLP_METRICS_ENDPOINT=http://otel-collector:4318/v1/metrics

```

### Example: Development (Disable Telemetry)

```bash
export ENV=dev   # This disables all telemetry
```

### Telemetry Decision Logic

```
ENV=dev?
  └─ YES → Telemetry OFF (no traces, no metrics)
  └─ NO  → Check TELEMETRY_ENABLED
              └─ false → Telemetry OFF
              └─ true  → Check individual flags:
                           ├─ TELEMETRY_TRACES_ENABLED + endpoint → Traces ON
                           └─ TELEMETRY_METRICS_ENABLED + endpoint → Metrics ON
```

> **Note:** Traces and metrics require both the `*_ENABLED` flag AND a valid `*_ENDPOINT` to be active.

---

## 8. Troubleshooting

| Issue | Solution |
|-------|----------|
| Plugin not found | Verify SHA256 matches, re-register plugin |
| Token generation failed | Check credentials file path and permissions |
| Role not found | Verify role exists: `vault list skyflow/.../roles` |
| Connection refused | Check Vault is running and `VAULT_ADDR` is set |
| Invalid role_ids | Ensure exactly one `role_id` is provided per role |

---

**Support:** Contact plugin admin for `role_ids` or credentials configuration issues.

