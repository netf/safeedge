#!/bin/bash
# SafeEdge Development Environment Setup
# One-command setup for local development

set -e

echo "🚀 SafeEdge Development Environment Setup"
echo ""

# Check prerequisites
echo "Checking prerequisites..."

if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker first."
    exit 1
fi

if ! command -v docker compose &> /dev/null; then
    echo "❌ Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.25+ first."
    exit 1
fi

echo "✓ Docker found: $(docker --version)"
echo "✓ Docker Compose found"
echo "✓ Go found: $(go version)"
echo ""

# Start infrastructure
echo "Starting infrastructure (PostgreSQL, Redis, MinIO)..."
docker compose up -d postgres redis minio

echo "Waiting for services to be ready..."
sleep 5

# Check if PostgreSQL is ready
until docker compose exec -T postgres pg_isready -U safeedge &> /dev/null; do
    echo "Waiting for PostgreSQL..."
    sleep 2
done
echo "✓ PostgreSQL is ready"

# Check if Redis is ready
until docker compose exec -T redis redis-cli ping &> /dev/null; do
    echo "Waiting for Redis..."
    sleep 2
done
echo "✓ Redis is ready"

echo ""
echo "✅ Development environment is ready!"
echo ""
echo "Next steps:"
echo "  1. Run database migrations (when implemented): make migrate"
echo "  2. Start control plane: go run ./cmd/control-plane"
echo "  3. Or build and run: go build -o bin/control-plane ./cmd/control-plane && ./bin/control-plane"
echo ""
echo "To stop infrastructure: docker compose down"
echo "To reset database: ./scripts/db-reset.sh"
