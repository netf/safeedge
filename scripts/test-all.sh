#!/bin/bash
# Run all tests (unit + E2E)

set -e

echo "🧪 Running all SafeEdge tests"
echo ""

# Unit tests (when they exist)
if [ -d "internal" ] || [ -d "pkg" ]; then
    echo "Running Go unit tests..."
    go test ./... -v -race -timeout=5m
    echo "✓ Unit tests passed"
    echo ""
fi

# E2E tests
if [ -d "e2e" ]; then
    echo "Setting up E2E test environment..."

    # Start test infrastructure
    docker compose -f docker-compose.e2e.yaml up -d

    # Wait for services
    echo "Waiting for test services to be ready..."
    sleep 10

    # Run E2E tests
    echo "Running E2E tests..."
    cd e2e

    # Install dependencies if needed
    if [ ! -d "node_modules" ]; then
        echo "Installing E2E test dependencies..."
        npm install
    fi

    npm test
    TEST_EXIT_CODE=$?

    cd ..

    # Cleanup
    echo "Cleaning up test infrastructure..."
    docker compose -f docker-compose.e2e.yaml down

    if [ $TEST_EXIT_CODE -eq 0 ]; then
        echo "✓ E2E tests passed"
    else
        echo "❌ E2E tests failed"
        exit $TEST_EXIT_CODE
    fi
else
    echo "⚠️  No E2E tests found (e2e/ directory doesn't exist)"
fi

echo ""
echo "✅ All tests completed successfully!"
