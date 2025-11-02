# SafeEdge - Technical Spec

Zero-trust device access and fleet management platform. Secure remote access and OTA updates for IoT/edge devices behind NAT/firewalls without public exposure.

---

## Core Features

- Device enrollment via time-limited tokens
- WireGuard-based zero-trust connectivity (outbound-only)
- Remote access sessions (SSH, port forwarding)
- Signed artifact distribution with staged rollout
- Automatic rollback on health check failure
- Device inventory and audit logging

---

## Architecture

```
┌────────────────────────────────┐
│      Operators (CLI/API)       │
└───────────┬────────────────────┘
            │ HTTPS (REST)
┌───────────▼────────────────────┐
│     Control Plane (Go)         │
│  ┌────────┐  ┌──────────┐     │
│  │  REST  │  │  Rollout │     │
│  │   API  │  │  Engine  │     │
│  └────────┘  └──────────┘     │
│  ┌────────┐  ┌──────────┐     │
│  │  gRPC  │  │Artifact  │     │
│  │Gateway │  │  Store   │     │
│  └────────┘  └──────────┘     │
│  ┌──────────────────────┐     │
│  │  PostgreSQL + Redis  │     │
│  └──────────────────────┘     │
└───────────┬────────────────────┘
            │ WireGuard + gRPC
    ┌───────┼───────┬───────┐
    │       │       │       │
┌───▼──┐ ┌──▼──┐ ┌─▼───┐ ...
│Agent │ │Agent│ │Agent│
└──────┘ └─────┘ └─────┘
```

**Layer Architecture:**
```
Handler Layer (chi HTTP, gRPC)
    ↓
Service Layer (Business Logic)
    ↓
sqlc.Queries (Generated SQL)
    ↓
PostgreSQL
```

---

## Components

### Device Agent (Go Binary)

- Outbound WireGuard tunnel (no inbound ports)
- Device enrollment and identity management
- Persistent gRPC connection for heartbeat
- Download, verify, apply updates
- Health reporting and automatic rollback

**Specs:**
- Binary: ~10-15MB (stripped)
- Memory: ~50-80MB resident
- Identity: Ed25519 keys in `/var/lib/safeedge/identity.json`

### Control Plane (Go Services)

1. **REST API** - Operator requests (JWT auth)
2. **gRPC Gateway** - Device bidirectional streams
3. **Rollout Engine** - Orchestrate canary → full rollout
4. **Artifact Store** - S3 storage + signature verification

---

## Connectivity

### Enrollment (One-Time)

1. Generate enrollment token (REST API)
2. Run: `safeedge-agent enroll --token <token>`
3. Agent generates Ed25519 keypair
4. Agent enrolls via HTTPS (pre-tunnel)
5. Control Plane issues device identity + WireGuard config
6. Agent establishes WireGuard tunnel
7. Agent connects via gRPC stream

### Ongoing

- Persistent WireGuard tunnel (outbound-only)
- Persistent gRPC bidirectional stream
- Heartbeat every 60s
- Device offline if no heartbeat for 5 min

### Remote Access

1. Operator requests access (REST API)
2. Control Plane provisions WireGuard peer config
3. Time-limited session (1-24 hours)
4. SSH/port-forward via tunnel
5. Session auto-terminates

---

## Rollout Flow

**States:** `DRAFT → CANARY → FULL → COMPLETE`
**Failure:** `↘ ROLLBACK → FAILED`

1. Upload artifact (compute BLAKE3 hash, sign with Ed25519)
2. Create rollout with canary percentage (e.g., 10%)
3. Start rollout → CANARY
4. Canary devices download, verify, apply
5. Health check after soak time (default: 5 min)
6. If healthy → FULL (remaining devices)
7. If unhealthy → ROLLBACK → FAILED

---

## Database Schema

```sql
CREATE TABLE organizations (
  id UUID PRIMARY KEY,
  name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE enrollment_tokens (
  id UUID PRIMARY KEY,
  organization_id UUID NOT NULL REFERENCES organizations(id),
  token_hash TEXT NOT NULL UNIQUE,
  site_tag TEXT,
  expires_at TIMESTAMPTZ NOT NULL,
  max_uses INTEGER NOT NULL DEFAULT 1,
  used_count INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_enrollment_tokens_org ON enrollment_tokens(organization_id);
CREATE INDEX idx_enrollment_tokens_expires ON enrollment_tokens(expires_at) WHERE used_count < max_uses;

CREATE TABLE devices (
  id UUID PRIMARY KEY,
  organization_id UUID NOT NULL REFERENCES organizations(id),
  public_key TEXT NOT NULL UNIQUE,
  wireguard_public_key TEXT NOT NULL UNIQUE,
  wireguard_ip INET NOT NULL,
  agent_version TEXT NOT NULL,
  platform TEXT NOT NULL,
  site_tag TEXT,
  status TEXT NOT NULL, -- ACTIVE, SUSPENDED, DECOMMISSIONED
  last_seen_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_devices_org ON devices(organization_id);
CREATE INDEX idx_devices_status ON devices(status);
CREATE INDEX idx_devices_last_seen ON devices(last_seen_at);

CREATE TABLE access_sessions (
  id UUID PRIMARY KEY,
  device_id UUID NOT NULL REFERENCES devices(id),
  user_email TEXT NOT NULL,
  wireguard_peer_config TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  terminated_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_access_sessions_device ON access_sessions(device_id);
CREATE INDEX idx_access_sessions_expires ON access_sessions(expires_at) WHERE terminated_at IS NULL;

CREATE TABLE artifacts (
  id UUID PRIMARY KEY,
  organization_id UUID NOT NULL REFERENCES organizations(id),
  name TEXT NOT NULL,
  type TEXT NOT NULL, -- CONTAINER, BINARY
  blake3_hash TEXT NOT NULL UNIQUE,
  signature BYTEA NOT NULL,
  signing_key_id TEXT NOT NULL,
  s3_url TEXT NOT NULL,
  size_bytes BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_artifacts_org ON artifacts(organization_id);
CREATE INDEX idx_artifacts_hash ON artifacts(blake3_hash);

CREATE TABLE rollouts (
  id UUID PRIMARY KEY,
  organization_id UUID NOT NULL REFERENCES organizations(id),
  artifact_id UUID NOT NULL REFERENCES artifacts(id),
  target_selector JSONB NOT NULL, -- e.g., {"site_tag": "warehouse-5"}
  canary_percent INTEGER NOT NULL DEFAULT 10,
  soak_time_seconds INTEGER NOT NULL DEFAULT 300,
  health_check_url TEXT NOT NULL, -- e.g., http://localhost:9090/health
  state TEXT NOT NULL, -- DRAFT, CANARY, FULL, COMPLETE, ROLLBACK, FAILED
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ
);
CREATE INDEX idx_rollouts_org ON rollouts(organization_id);
CREATE INDEX idx_rollouts_state ON rollouts(state);

CREATE TABLE rollout_device_status (
  rollout_id UUID NOT NULL REFERENCES rollouts(id),
  device_id UUID NOT NULL REFERENCES devices(id),
  is_canary BOOLEAN NOT NULL,
  status TEXT NOT NULL, -- PENDING, IN_PROGRESS, HEALTHY, UNHEALTHY, ROLLED_BACK
  health_check_result JSONB,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (rollout_id, device_id)
);
CREATE INDEX idx_rollout_device_status_rollout ON rollout_device_status(rollout_id);

CREATE TABLE audit_logs (
  id UUID PRIMARY KEY,
  timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  organization_id UUID NOT NULL REFERENCES organizations(id),
  user_email TEXT,
  event_type TEXT NOT NULL, -- device.enrolled, access.started, rollout.started, etc.
  resource_type TEXT NOT NULL, -- device, rollout, artifact, etc.
  resource_id TEXT NOT NULL,
  action TEXT NOT NULL,
  result TEXT NOT NULL, -- success, failure
  metadata JSONB,
  ip_address INET
);
CREATE INDEX idx_audit_logs_org ON audit_logs(organization_id);
CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp);
CREATE INDEX idx_audit_logs_event_type ON audit_logs(event_type);
```

---

## API Endpoints

### REST API (Operators → Control Plane)

**Base:** `https://api.safeedge.io/v1`
**Auth:** `Authorization: Bearer <jwt>`

```
# Enrollment
POST   /v1/enrollment-tokens              # Generate token
POST   /v1/enrollments                    # Device enrollment (HTTPS, pre-tunnel)

# Devices
GET    /v1/devices                        # List devices
GET    /v1/devices/:id                    # Get device
POST   /v1/devices/:id/suspend            # Suspend device
POST   /v1/devices/:id/reactivate         # Reactivate device

# Access
POST   /v1/access-sessions                # Create access session
DELETE /v1/access-sessions/:id            # Terminate session

# Artifacts
POST   /v1/artifacts                      # Upload artifact
GET    /v1/artifacts/:id                  # Get artifact

# Rollouts
POST   /v1/rollouts                       # Create rollout (DRAFT)
GET    /v1/rollouts/:id                   # Get rollout
POST   /v1/rollouts/:id/start             # Start rollout
POST   /v1/rollouts/:id/abort             # Abort rollout

# Audit
GET    /v1/audit-logs                     # List logs (?start_time, ?end_time, ?event_type)
```

### gRPC API (Agent ↔ Control Plane)

**Transport:** Over WireGuard tunnel

```protobuf
service DeviceService {
  rpc DeviceStream(stream DeviceMessage) returns (stream ControlMessage);
}

message DeviceMessage {
  oneof payload {
    HeartbeatRequest heartbeat = 1;
    HealthReport health = 2;
    UpdateAck update_ack = 3;
  }
}

message ControlMessage {
  oneof payload {
    HeartbeatAck heartbeat_ack = 1;
    UpdateNotification update = 2;
    RollbackRequest rollback = 3;
  }
}
```

---

## Project Structure

```
safeedge/
├── cmd/
│   ├── agent/              # Device agent binary
│   ├── control-plane/      # Control plane server
│   └── cli/                # Operator CLI
├── pkg/
│   ├── agent/              # Agent library
│   └── crypto/              # Crypto utilities (Ed25519, BLAKE3)
├── internal/
│   ├── agent/
│   │   ├── enrollment/
│   │   ├── tunnel/         # WireGuard
│   │   ├── workload/       # Docker/binary management
│   │   └── update/         # Download, verify, apply
│   └── controlplane/
│       ├── database/
│       │   ├── schema.sql          # DDL
│       │   ├── queries/            # sqlc query files
│       │   │   ├── devices.sql
│       │   │   ├── enrollment_tokens.sql
│       │   │   ├── rollouts.sql
│       │   │   └── audit_logs.sql
│       │   ├── sqlc.yaml           # sqlc config
│       │   └── generated/          # Generated Go code
│       ├── server/
│       │   ├── rest/               # chi router
│       │   │   ├── handlers/
│       │   │   │   ├── device.go
│       │   │   │   ├── enrollment.go
│       │   │   │   ├── rollout.go
│       │   │   │   └── ...
│       │   │   └── server.go
│       │   └── grpc/               # gRPC server
│       │       └── server.go
│       ├── service/                # Business logic
│       │   ├── device_service.go
│       │   ├── rollout_service.go
│       │   └── ...
│       └── middleware/             # Auth, logging, etc.
├── api/proto/                      # Protobuf definitions
└── deployments/
    └── docker-compose.yml
```

---

## Technology Stack

### Agent
- **Language:** Go 1.25+
- **Dependencies:**
  - `golang.zx2c4.com/wireguard` - WireGuard
  - `crypto/ed25519` - Signing
  - `github.com/zeebo/blake3` - Hashing
  - `github.com/spf13/cobra` - CLI
  - `go.uber.org/zap` - Logging
  - `modernc.org/sqlite` - Local state

### Control Plane
- **Language:** Go 1.25+
- **Dependencies:**
  - `github.com/go-chi/chi/v5` - HTTP router
  - `github.com/jackc/pgx/v5` - PostgreSQL driver
  - `github.com/sqlc-dev/sqlc` - Type-safe SQL code generation
  - `github.com/redis/go-redis/v9` - Redis client
  - `github.com/golang-jwt/jwt/v5` - JWT
  - `google.golang.org/grpc` - gRPC
  - `go.uber.org/zap` - Logging

### Infrastructure
- PostgreSQL 15+
- Redis 7+ (sessions, rate limiting)
- S3-compatible storage (MinIO/AWS S3)
- nginx/Caddy (load balancer)

---

## sqlc Configuration

**`internal/controlplane/database/sqlc.yaml`:**
```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "queries/"
    schema: "schema.sql"
    gen:
      go:
        package: "generated"
        out: "generated"
        emit_json_tags: true
        emit_prepared_queries: false
        emit_interface: true
        emit_exact_table_names: false
```

**Usage:**
```bash
cd internal/controlplane/database
sqlc generate
```

**Service Layer Example:**
```go
type DeviceService struct {
    queries *generated.Queries
    pool    *pgxpool.Pool
}

func (s *DeviceService) SuspendDevice(ctx context.Context, deviceID uuid.UUID) error {
    return s.queries.UpdateDeviceStatus(ctx, generated.UpdateDeviceStatusParams{
        ID:     deviceID,
        Status: "SUSPENDED",
    })
}
```

---

## Cryptography

- **Signing:** Ed25519 (256-bit)
- **Hashing:** BLAKE3 (256-bit)
- **Tunneling:** WireGuard (Curve25519)
- **TLS:** TLS 1.3
- **Random:** crypto/rand

---

## Performance Targets

**Agent:**
- Binary: ~10-15MB
- Memory: ~50-80MB
- Startup: <5s
- CPU idle: <5%

**Control Plane:**
- API latency: p99 <500ms
- Throughput: 100 req/s per instance
- Device scale: 1,000 per instance

---

## Deployment (Docker Compose)

```yaml
version: '3.8'
services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: safeedge
      POSTGRES_USER: safeedge
      POSTGRES_PASSWORD: safeedge
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine

  minio:
    image: minio/minio
    command: server /data
    environment:
      MINIO_ROOT_USER: safeedge
      MINIO_ROOT_PASSWORD: safeedge123
    volumes:
      - minio_data:/data

  control-plane:
    build: .
    ports:
      - "8080:8080"   # REST API
      - "9090:9090"   # gRPC
      - "51820:51820/udp"  # WireGuard
    environment:
      DATABASE_URL: postgres://safeedge:safeedge@postgres:5432/safeedge
      REDIS_URL: redis://redis:6379
      S3_ENDPOINT: http://minio:9000
    depends_on:
      - postgres
      - redis
      - minio

  agent:
    ...

volumes:
  postgres_data:
  minio_data:
```

---

## Milestones

### M0 - Proof of Concept (2-3 weeks)
- Agent: enrollment, WireGuard tunnel, heartbeat
- Control Plane: token generation, enrollment, device inventory
- **Goal:** SSH to device behind NAT

### M1 - Updates & Rollout (4-5 weeks)
- Artifact signing and storage
- Rollout engine (canary → full)
- Agent update mechanism with rollback
- **Goal:** Safe rollout to 50 devices

### M2 - Production Ready (3-4 weeks)
- Horizontal scaling
- Observability (logs, metrics)
- Security hardening
- **Goal:** 1,000 devices, p99 <500ms

---

**Success Criteria:**
1. Remote access to devices behind NAT without VPN
2. Zero-downtime updates with automatic rollback
3. Complete audit trail for compliance
