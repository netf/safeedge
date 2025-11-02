---
description: Run E2E tests with proper Docker infrastructure setup
---

Run the E2E test suite with full infrastructure (PostgreSQL, Redis, MinIO, Control Plane).

Steps:
1. Check if Docker is running: `docker info`
2. Start test infrastructure: `docker compose -f docker-compose.e2e.yaml up -d`
3. Wait 10 seconds for services to stabilize
4. Check service health: `docker compose -f docker-compose.e2e.yaml ps`
5. Navigate to E2E directory: `cd e2e`
6. Install dependencies if needed: `npm install` (only if package.json changed)
7. Run tests: `npm test`
8. Report test results
9. If tests fail, provide path to `playwright-report/` for detailed traces
