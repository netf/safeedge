# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

SafeEdge is a zero-trust device access and fleet management platform for IoT/edge devices. It provides secure remote access and OTA updates for devices behind NAT/firewalls without requiring public exposure.

**Core Capabilities:**
- Device enrollment via time-limited tokens
- WireGuard-based zero-trust connectivity (outbound-only from devices)
- Remote access sessions (SSH, port forwarding)
- Signed artifact distribution with staged rollout
- Automatic rollback on health check failure
- Device inventory and audit logging

## Architecture

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

**Components:**
- **Device Agent**: Go binary (~10-15MB) running on edge devices with outbound WireGuard tunnel and persistent gRPC connection
- **Control Plane**: Go service with REST API (operators), gRPC Gateway (devices), Rollout Engine, and Artifact Store

**Connectivity Model:**
- Devices maintain persistent outbound WireGuard tunnels (no inbound ports)
- Devices maintain persistent gRPC bidirectional streams for heartbeat/control
- Remote access via time-limited WireGuard peer configs provisioned by control plane

## Technology Stack

**Backend:**
- Go 1.25+
- PostgreSQL 15+ (primary datastore)
- Redis 7+ (sessions, rate limiting)
- S3-compatible storage (MinIO/AWS S3 for artifacts)

**Key Dependencies:**
- `github.com/go-chi/chi/v5` - HTTP router
- `github.com/jackc/pgx/v5` - PostgreSQL driver
- `github.com/sqlc-dev/sqlc` - Type-safe SQL code generation
- `github.com/redis/go-redis/v9` - Redis client
- `github.com/golang-jwt/jwt/v5` - JWT authentication
- `google.golang.org/grpc` - gRPC
- `golang.zx2c4.com/wireguard` - WireGuard
- `github.com/zeebo/blake3` - BLAKE3 hashing for artifacts
- `crypto/ed25519` - Ed25519 signing
- `go.uber.org/zap` - Structured logging

## Database Code Generation with sqlc

This project uses **sqlc** for type-safe SQL code generation. All database interactions go through generated code.

**Location:** `internal/controlplane/database/`

**Workflow:**
1. Define schema in `schema.sql`
2. Write queries in `queries/*.sql` (devices.sql, enrollment_tokens.sql, rollouts.sql, etc.)
3. Run `sqlc generate` to create Go code in `generated/`
4. Use generated `Queries` type in service layer

**To regenerate database code:**
```bash
cd internal/controlplane/database
sqlc generate
```

**Service Layer Pattern:**
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

**Database Query Changes:** Instead of writing raw SQL in Go code, add queries to `queries/*.sql` and regenerate with `sqlc generate`.

## Project Structure

```
safeedge/
├── cmd/
│   ├── agent/              # Device agent binary
│   ├── control-plane/      # Control plane server
│   └── cli/                # Operator CLI
├── pkg/
│   ├── agent/              # Agent library (public API)
│   └── crypto/             # Crypto utilities (Ed25519, BLAKE3)
├── internal/
│   ├── agent/
│   │   ├── enrollment/     # Device enrollment logic
│   │   ├── tunnel/         # WireGuard tunnel management
│   │   ├── workload/       # Docker/binary management
│   │   └── update/         # Download, verify, apply updates
│   └── controlplane/
│       ├── database/
│       │   ├── schema.sql          # PostgreSQL DDL
│       │   ├── queries/            # sqlc query files
│       │   ├── sqlc.yaml           # sqlc config
│       │   └── generated/          # Generated Go code (DO NOT EDIT)
│       ├── server/
│       │   ├── rest/               # chi HTTP router + handlers
│       │   └── grpc/               # gRPC server for agent streams
│       ├── service/                # Business logic layer
│       └── middleware/             # Auth, logging, etc.
├── api/proto/                      # Protobuf definitions
└── deployments/
    └── docker-compose.yml
```

## Common Development Commands

**Database:**
```bash
# Regenerate database code after modifying schema or queries
cd internal/controlplane/database
sqlc generate
```

**Testing:**
```bash
# Run E2E tests (Playwright + PostgreSQL snapshots)
cd e2e
npm install
npx playwright install --with-deps chromium
docker compose -f ../docker-compose.e2e.yaml up -d
npm test

# Run specific test file
npm test tests/enrollment.spec.ts

# Interactive test mode
npm run test:ui
```

**Build:**
```bash
# Build agent binary
go build -ldflags="-s -w" -o bin/safeedge-agent ./cmd/agent

# Build control plane
go build -o bin/control-plane ./cmd/control-plane

# Build agent Docker image for testing
docker build -f Dockerfile.agent -t safeedge-agent:test .
```

**Deployment:**
```bash
# Start infrastructure (PostgreSQL, Redis, MinIO, Control Plane)
docker compose up -d

# View logs
docker compose logs -f control-plane
```

## API Endpoints

**REST API (Operators → Control Plane):**
- Base: `https://api.safeedge.io/v1`
- Auth: `Authorization: Bearer <jwt>`

Key endpoints:
- `POST /v1/enrollment-tokens` - Generate enrollment token
- `POST /v1/enrollments` - Device enrollment (HTTPS, pre-tunnel)
- `GET /v1/devices` - List devices
- `POST /v1/devices/:id/suspend` - Suspend device
- `POST /v1/access-sessions` - Create remote access session
- `DELETE /v1/access-sessions/:id` - Terminate session
- `POST /v1/artifacts` - Upload artifact
- `POST /v1/rollouts` - Create rollout (DRAFT state)
- `POST /v1/rollouts/:id/start` - Start rollout
- `GET /v1/audit-logs` - Query audit logs

**gRPC API (Agent ↔ Control Plane):**
- Transport: Over WireGuard tunnel
- Service: `DeviceService.DeviceStream` (bidirectional stream)
- Messages: Heartbeat, HealthReport, UpdateNotification, RollbackRequest

## Rollout State Machine

```
States: DRAFT → CANARY → FULL → COMPLETE
Failure: ↘ ROLLBACK → FAILED
```

**Rollout Flow:**
1. Upload artifact (compute BLAKE3 hash, sign with Ed25519)
2. Create rollout with canary percentage (e.g., 10%)
3. Start rollout → CANARY state
4. Canary devices download, verify signature, apply update
5. Health check after soak time (default: 5 min)
6. If healthy → FULL state (remaining devices get update)
7. If unhealthy → ROLLBACK → FAILED

## Cryptography Standards

- **Artifact Signing:** Ed25519 (256-bit)
- **Artifact Hashing:** BLAKE3 (256-bit)
- **Tunneling:** WireGuard (Curve25519)
- **TLS:** TLS 1.3 for HTTPS enrollment
- **Random:** crypto/rand

**Security Requirements:**
- All artifacts MUST be signed with Ed25519 before storage
- All artifact downloads MUST verify BLAKE3 hash + signature
- Device identity keys stored in `/var/lib/safeedge/identity.json`
- No inbound ports on devices - outbound WireGuard only

## Performance Targets

**Agent:**
- Binary size: ~10-15MB (stripped)
- Memory: ~50-80MB resident
- Startup: <5s
- CPU idle: <5%

**Control Plane:**
- API latency: p99 <500ms
- Throughput: 100 req/s per instance
- Device scale: 1,000 devices per instance

## Database Schema Key Points

**Device States:** `ACTIVE`, `SUSPENDED`, `DECOMMISSIONED`

**Rollout States:** `DRAFT`, `CANARY`, `FULL`, `COMPLETE`, `ROLLBACK`, `FAILED`

**Rollout Device Status:** `PENDING`, `IN_PROGRESS`, `HEALTHY`, `UNHEALTHY`, `ROLLED_BACK`

**Important Indexes:**
- `idx_enrollment_tokens_expires` - For token cleanup
- `idx_devices_last_seen` - For offline detection
- `idx_access_sessions_expires` - For session expiration
- `idx_rollout_device_status_rollout` - For rollout queries

## Testing Strategy

**E2E Tests** (see TESTING.md):
- Playwright for API testing
- PostgreSQL snapshots for test isolation
- Docker Compose for infrastructure
- Agent tests via Docker containers
- Sequential execution (workers: 1) for DB consistency

**Test Files:**
- `e2e/tests/enrollment.spec.ts` - Device enrollment flows
- `e2e/tests/devices.spec.ts` - Device management
- `e2e/tests/access-sessions.spec.ts` - Remote access
- `e2e/tests/rollouts.spec.ts` - Rollout state machine
- `e2e/tests/audit.spec.ts` - Audit logging
- `e2e/tests/agent.spec.ts` - Agent heartbeat/reconnection

## Milestones

**M0 - Proof of Concept:**
- Agent: enrollment, WireGuard tunnel, heartbeat
- Control Plane: token generation, enrollment, device inventory
- Goal: SSH to device behind NAT

**M1 - Updates & Rollout:**
- Artifact signing and storage
- Rollout engine (canary → full)
- Agent update mechanism with rollback
- Goal: Safe rollout to 50 devices

**M2 - Production Ready:**
- Horizontal scaling
- Observability (logs, metrics)
- Security hardening
- Goal: 1,000 devices, p99 <500ms

## Common Pitfalls & Troubleshooting

**sqlc Generation Issues:**
- **Symptom:** `sqlc generate` fails with parsing errors
- **Fix:** Check `queries/*.sql` syntax - each query must have a `-- name:` comment and proper type annotations
- **Example:** `-- name: GetDevice :one` for single row, `-- name: ListDevices :many` for multiple rows

**WireGuard Tunnel Problems:**
- **Symptom:** Agent can't establish tunnel or connection drops
- **Check:** Ensure kernel has WireGuard support (`modprobe wireguard`), verify no IP conflicts in tunnel CIDR
- **Agent Logs:** Look for "tunnel established" message - absence indicates enrollment or key exchange failure

**gRPC Stream Disconnects:**
- **Symptom:** Device shows offline despite agent running
- **Fix:** Check firewall rules, verify gRPC port (9090) accessible, examine `last_seen_at` timestamp
- **Reconnection:** Agent should auto-reconnect with exponential backoff - check for "reconnecting" log entries

**Rollout Stuck in CANARY:**
- **Symptom:** Rollout doesn't progress to FULL state
- **Check:** Verify health check URL returns 200, ensure soak time elapsed, check `rollout_device_status` table for device states
- **Manual Inspection:** Query `SELECT status, COUNT(*) FROM rollout_device_status WHERE rollout_id = $1 GROUP BY status`

**Docker Compose E2E Test Failures:**
- **Symptom:** Tests fail with connection errors
- **Fix:** Ensure network `safeedge-e2e` exists, verify PostgreSQL/Redis started fully (check `docker compose logs`)
- **Reset:** Run `docker compose -f docker-compose.e2e.yaml down -v && docker compose -f docker-compose.e2e.yaml up -d`

**Token Budget Awareness:**
- This CLAUDE.md is intentionally concise (~7-8KB). For complex questions about external dependencies (WireGuard, gRPC, Playwright), ask user if you should fetch official docs rather than guessing.
- Use PROJECT.md for detailed architecture, TESTING.md for comprehensive E2E test patterns.

## Code Review Checklist

When reviewing or writing code, ensure:
- [ ] All database changes go through sqlc (no `db.Exec()` with raw SQL strings)
- [ ] gRPC service definitions match `api/proto/*.proto` files
- [ ] Artifact signatures verified before trust (Ed25519 + BLAKE3)
- [ ] Rollout state transitions follow: DRAFT → CANARY → FULL → COMPLETE (or → ROLLBACK → FAILED)
- [ ] Errors logged with structured fields (`zap.String("device_id", id)` not string concatenation)
- [ ] Time comparisons use UTC (`time.Now().UTC()`)
- [ ] Device identity keys loaded from `/var/lib/safeedge/identity.json` (not hardcoded paths)

## Development Workflow Recommendations

**When modifying database schema:**
1. Edit `internal/controlplane/database/schema.sql`
2. Add/modify queries in `internal/controlplane/database/queries/*.sql`
3. Run `cd internal/controlplane/database && sqlc generate`
4. Update service layer to use new generated types
5. Add E2E test for the feature in `e2e/tests/`

**When adding new REST endpoint:**
1. Add handler in `internal/controlplane/server/rest/handlers/`
2. Register route in `internal/controlplane/server/rest/server.go`
3. Implement service logic using `sqlc.Queries`
4. Add API test in `e2e/tests/`
5. Update this CLAUDE.md's API Endpoints section if it's a major feature

**When debugging E2E test failures:**
1. Check `e2e/playwright-report/` for detailed traces
2. Run with `npm run test:debug` to step through
3. Verify Docker containers running: `docker compose -f docker-compose.e2e.yaml ps`
4. Check agent logs: `docker logs safeedge-test-agent`
5. Inspect database state: `psql -h localhost -p 5433 -U safeedge -d safeedge_test`
