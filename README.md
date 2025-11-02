# SafeEdge

Zero-trust device access and fleet management platform for IoT/edge devices.

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.25+-00ADD8.svg)](https://golang.org/)

## Overview

SafeEdge provides secure remote access and OTA updates for devices behind NAT/firewalls without requiring public exposure. Devices maintain outbound-only WireGuard tunnels and persistent gRPC connections to the control plane.

**Key Features:**
- 🔐 Device enrollment via time-limited tokens
- 🌐 WireGuard-based zero-trust connectivity (outbound-only)
- 🔌 Remote access sessions (SSH, port forwarding)
- 📦 Signed artifact distribution with staged rollout
- ↩️ Automatic rollback on health check failure
- 📊 Device inventory and comprehensive audit logging

## Quick Start

### Prerequisites

- Go 1.25+
- Docker & Docker Compose
- PostgreSQL 15+ (or use Docker Compose)
- protoc (for gRPC code generation)
- sqlc (for database code generation)

### Development Setup

```bash
# Clone repository
git clone https://github.com/netf/safeedge.git
cd safeedge

# Start infrastructure (PostgreSQL, Redis, MinIO)
./scripts/dev-setup.sh

# Or manually:
docker compose up -d

# Build binaries
make build

# Run control plane
./bin/control-plane

# In another terminal, run agent
./bin/agent run --device-id <device-uuid> --control-plane localhost:9090
```

### Docker Compose

```bash
# Start all services
docker compose up -d

# View logs
docker compose logs -f control-plane

# Stop services
docker compose down
```

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

## API Endpoints

### REST API (Operators → Control Plane)

**Base:** `http://localhost:8080/v1`

```bash
# Health check
curl http://localhost:8080/health

# Create enrollment token
curl -X POST http://localhost:8080/v1/enrollment-tokens \
  -H "Content-Type: application/json" \
  -d '{
    "organization_id": "00000000-0000-0000-0000-000000000001",
    "site_tag": "warehouse-1",
    "expires_in_seconds": 3600,
    "max_uses": 1
  }'

# List devices
curl http://localhost:8080/v1/devices

# Get device
curl http://localhost:8080/v1/devices/{id}

# Suspend device
curl -X POST http://localhost:8080/v1/devices/{id}/suspend
```

### gRPC API (Agent ↔ Control Plane)

**Address:** `localhost:9090`

Bidirectional streaming for:
- Heartbeats (agent → control plane)
- Health reports (agent → control plane)
- Update notifications (control plane → agent)
- Rollback requests (control plane → agent)

## Development

### Build Commands

```bash
# Build all binaries
make build

# Generate protobuf code
make proto

# Generate database code
make sqlc

# Format code
make fmt

# Run linters
make lint

# Run tests
make test
```

### Database

The project uses **sqlc** for type-safe SQL code generation:

```bash
# Regenerate database code after schema/query changes
make sqlc

# Or manually:
cd internal/controlplane/database
sqlc generate
```

**Important:** Never write raw SQL in Go code. Always add queries to `queries/*.sql` and regenerate.

### Project Structure

```
safeedge/
├── cmd/
│   ├── agent/              # Device agent binary
│   ├── control-plane/      # Control plane server
│   └── cli/                # Operator CLI
├── internal/
│   ├── agent/              # Agent implementation
│   └── controlplane/
│       ├── database/       # Schema, queries, generated code
│       ├── server/         # REST + gRPC servers
│       ├── service/        # Business logic
│       └── middleware/     # Auth, logging
├── api/proto/              # Protobuf definitions
└── scripts/                # Helper scripts
```

## Testing

```bash
# Run E2E tests
./scripts/test-all.sh

# Or with Docker Compose:
docker compose -f docker-compose.e2e.yaml up -d
cd e2e && npm test
```

## Milestones

### ✅ M0 - Proof of Concept (Current)
- ✅ Project structure and infrastructure
- ✅ Database schema and queries
- ✅ gRPC communication protocol
- ✅ Control plane REST API skeleton
- ✅ Agent skeleton with heartbeat
- ⏳ WireGuard tunnel implementation
- ⏳ Complete enrollment flow
- **Goal:** SSH to device behind NAT

### M1 - Updates & Rollout (Next)
- Artifact signing and storage
- Rollout engine (canary → full)
- Agent update mechanism with rollback
- **Goal:** Safe rollout to 50 devices

### M2 - Production Ready
- Horizontal scaling
- Observability (logs, metrics)
- Security hardening
- **Goal:** 1,000 devices, p99 <500ms

## Documentation

- [CLAUDE.md](CLAUDE.md) - AI assistant guidance
- [PROJECT.md](PROJECT.md) - Technical specification
- [TESTING.md](TESTING.md) - E2E testing strategy
- [.claude/README.md](.claude/README.md) - Claude Code configuration

## Technology Stack

- **Language:** Go 1.25+
- **Database:** PostgreSQL 15+ with sqlc
- **Cache:** Redis 7+
- **Storage:** MinIO / S3
- **HTTP:** chi router
- **gRPC:** google.golang.org/grpc
- **Logging:** zap
- **Crypto:** Ed25519 (signing), BLAKE3 (hashing), WireGuard (tunneling)

## Contributing

This project uses Claude Code best practices:

```bash
# Use custom slash commands
/catchup          # Read changed files in branch
/pr               # Prepare pull request
/test-e2e         # Run E2E tests
/dbgen            # Regenerate sqlc code
```

Pre-commit hooks will check:
- Go formatting
- sqlc regeneration
- No secrets in commits

## License

MIT License - see [LICENSE](LICENSE) for details

## Support

- Issues: [GitHub Issues](https://github.com/netf/safeedge/issues)
- Documentation: [docs/](docs/)
