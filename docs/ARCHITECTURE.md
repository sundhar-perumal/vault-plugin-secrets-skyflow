# Architecture

High-level overview of the Skyflow Vault Plugin.

## System Flow

```
┌─────────────┐      ┌──────────────────┐      ┌─────────────┐
│  Application │ ──▶ │   Vault Plugin   │ ──▶ │ Skyflow API │
│  (Consumer)  │     │  skyflow/creds/  │     │  (Token)    │
└─────────────┘      └──────────────────┘      └─────────────┘
        │                     │
        │                     ▼
        │            ┌──────────────────┐
        │            │  Vault Storage   │
        │            │  (Seal-Wrapped)  │
        │            │  - config        │
        │            │  - roles/*       │
        │            └──────────────────┘
        │
        ▼
┌─────────────────────────────────────────────────────────┐
│                    Skyflow Vault                        │
│              (PII Data Operations)                      │
└─────────────────────────────────────────────────────────┘
```

## Multi-Mount Design

Each business unit gets its own mount path with isolated credentials:

```
vault/
├── skyflow/insurance/     # Insurance BU credentials
│   ├── config
│   └── roles/
│       ├── producer
│       └── consumer-*
│
├── skyflow/lending/       # Lending BU credentials
│   ├── config
│   └── roles/
│       ├── producer
│       └── consumer-*
│
└── skyflow/clcm/          # CLCM BU credentials
    ├── config
    └── roles/
```

## Components

| Component | Purpose |
|-----------|---------|
| `backend.go` | Core plugin, factory, lifecycle |
| `config.go` | Credential storage (seal-wrapped) |
| `role.go` | Role definitions with `role_ids` |
| `path_token.go` | Token generation via Skyflow SDK |
| `path_health.go` | Health check endpoint |
| `telemetry/` | OpenTelemetry tracing & metrics |

## Token Flow

1. **Request** → `GET /skyflow/insurance/creds/producer`
2. **Lookup** → Load role config (`role_ids`)
3. **Lookup** → Load backend config (credentials)
4. **Generate** → Call Skyflow SDK with `role_ids` + optional `ctx`
5. **Return** → `{ access_token, token_type }`

## Security Model

| Layer | Protection |
|-------|------------|
| Credentials | Vault seal-wrap encryption |
| Tokens | Short-lived JWT (max 1hr TTL) |
| API Access | Vault ACL policies |
| Audit | Vault audit backend |

## Telemetry

| Signal | Data |
|--------|------|
| Traces | Token requests, config/role operations |
| Metrics | Token generation count, latency, errors |
| Control | `ENV=dev` disables telemetry |
