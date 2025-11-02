#!/bin/bash
# Reset local database to clean state

set -e

echo "🔄 Resetting SafeEdge database"
echo ""

# Check if PostgreSQL container is running
if ! docker compose ps postgres | grep -q "Up"; then
    echo "❌ PostgreSQL container is not running. Start it with: docker compose up -d postgres"
    exit 1
fi

echo "⚠️  WARNING: This will delete all data in the local database!"
read -p "Are you sure you want to continue? (y/N) " -n 1 -r
echo ""

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 0
fi

echo "Dropping and recreating database..."
docker compose exec -T postgres psql -U safeedge -d postgres <<EOF
DROP DATABASE IF EXISTS safeedge;
CREATE DATABASE safeedge;
EOF

echo "✓ Database reset complete"
echo ""
echo "Next steps:"
echo "  1. Run migrations (when implemented): make migrate"
echo "  2. Or manually apply schema: psql -h localhost -U safeedge -d safeedge -f internal/controlplane/database/schema.sql"
