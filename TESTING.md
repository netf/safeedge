# SafeEdge - E2E Testing

End-to-end API testing using Playwright and PostgreSQL snapshots for test isolation.

---

## Stack

- **Playwright** - API testing via browser context
- **PostgreSQL** - Database with snapshot/restore
- **Docker Compose** - Test infrastructure
- **TypeScript** - Type-safe test code

---

## Directory Structure

```
e2e/
├── config.ts                    # Test configuration
├── types.ts                     # TypeScript types
├── playwright.config.ts         # Playwright config
├── package.json
├── .env
│
├── tests/
│   ├── enrollment.spec.ts       # Device enrollment
│   ├── devices.spec.ts          # Device management
│   ├── access-sessions.spec.ts  # Remote access
│   ├── artifacts.spec.ts        # Artifact upload
│   ├── rollouts.spec.ts         # Rollout management
│   ├── audit.spec.ts            # Audit logs
│   └── agent.spec.ts            # Agent e2e tests (heartbeat)
│
└── utils/
    ├── api/
    │   ├── enrollment.ts        # Enrollment API helpers
    │   ├── devices.ts           # Device API helpers
    │   ├── artifacts.ts         # Artifact API helpers
    │   ├── rollouts.ts          # Rollout API helpers
    │   └── audit.ts             # Audit API helpers
    ├── db/
    │   ├── connection.ts        # PostgreSQL connection
    │   └── snapshot.ts          # Snapshot/restore
    ├── agent.ts                 # Agent container management
    ├── docker.ts                # Docker management
    ├── setup.ts                 # Global setup
    └── teardown.ts              # Global teardown
```

---

## Setup

```bash
cd e2e
npm install
npx playwright install --with-deps chromium

# Start test infrastructure
docker compose -f ../docker-compose.e2e.yaml up -d

# Run tests
npm test
```

---

## Configuration

### `playwright.config.ts`

```typescript
import { defineConfig } from '@playwright/test';
import { testsConfig } from './config';

export default defineConfig({
  timeout: 60000,
  testDir: './tests',
  fullyParallel: false,
  workers: 1,  // Sequential execution for DB state consistency
  retries: process.env.CI ? 2 : 0,

  use: {
    baseURL: testsConfig.BASE_URL,
    trace: 'retain-on-failure',
    video: { mode: 'retain-on-failure' },
    screenshot: 'only-on-failure'
  },

  globalSetup: require.resolve('./utils/setup'),
  globalTeardown: require.resolve('./utils/teardown')
});
```

### `config.ts`

```typescript
export const testsConfig = {
  BASE_URL: process.env.BASE_URL || 'http://localhost:8080',
  API_BASE_URL: process.env.API_BASE_URL || 'http://localhost:8080/v1',
  DB_HOST: process.env.DB_HOST || 'localhost',
  DB_PORT: parseInt(process.env.DB_PORT || '5432'),
  DB_NAME: process.env.DB_NAME || 'safeedge_test',
  DB_USER: process.env.DB_USER || 'safeedge',
  DB_PASSWORD: process.env.DB_PASSWORD || 'safeedge'
};

export const routes = {
  enrollmentTokens: '/v1/enrollment-tokens',
  enrollments: '/v1/enrollments',
  devices: '/v1/devices',
  accessSessions: '/v1/access-sessions',
  artifacts: '/v1/artifacts',
  rollouts: '/v1/rollouts',
  auditLogs: '/v1/audit-logs'
};

export const testOrg = {
  id: '00000000-0000-0000-0000-000000000001',
  name: 'Test Organization'
};

export const testUser = {
  email: 'admin@test.com',
  password: 'test123'
};
```

### `types.ts`

```typescript
export type EnrollmentToken = {
  id: string;
  token: string;
  organization_id: string;
  site_tag?: string;
  expires_at: string;
  max_uses: number;
  used_count: number;
};

export type Device = {
  id: string;
  organization_id: string;
  public_key: string;
  wireguard_public_key: string;
  wireguard_ip: string;
  agent_version: string;
  platform: string;
  site_tag?: string;
  status: 'ACTIVE' | 'SUSPENDED' | 'DECOMMISSIONED';
  last_seen_at?: string;
  created_at: string;
};

export type AccessSession = {
  id: string;
  device_id: string;
  user_email: string;
  wireguard_peer_config: string;
  expires_at: string;
  terminated_at?: string;
  created_at: string;
};

export type Artifact = {
  id: string;
  organization_id: string;
  name: string;
  type: 'CONTAINER' | 'BINARY';
  blake3_hash: string;
  signature: string;
  signing_key_id: string;
  s3_url: string;
  size_bytes: number;
  created_at: string;
};

export type Rollout = {
  id: string;
  organization_id: string;
  artifact_id: string;
  target_selector: Record<string, any>;
  canary_percent: number;
  soak_time_seconds: number;
  health_check_url: string;
  state: 'DRAFT' | 'CANARY' | 'FULL' | 'COMPLETE' | 'ROLLBACK' | 'FAILED';
  created_at: string;
  started_at?: string;
  completed_at?: string;
};

export type AuditLog = {
  id: string;
  timestamp: string;
  organization_id: string;
  user_email?: string;
  event_type: string;
  resource_type: string;
  resource_id: string;
  action: string;
  result: 'success' | 'failure';
  metadata?: Record<string, any>;
  ip_address?: string;
};
```

### `package.json`

```json
{
  "name": "safeedge-e2e",
  "version": "0.1.0",
  "scripts": {
    "test": "playwright test",
    "test:headed": "playwright test --headed",
    "test:debug": "playwright test --debug",
    "test:ui": "playwright test --ui"
  },
  "dependencies": {
    "@playwright/test": "^1.55.1",
    "pg": "^8.16.3",
    "dotenv": "^17.2.2"
  },
  "devDependencies": {
    "@types/node": "^22.18.6",
    "@types/pg": "^8.15.5",
    "typescript": "^5.7.3"
  }
}
```

---

## Test Patterns

### Device Enrollment

**`tests/enrollment.spec.ts`:**

```typescript
import { expect, test } from '@playwright/test';
import { routes } from '../config';
import { apiCreateEnrollmentToken } from '../utils/api/enrollment';
import { apiEnrollDevice } from '../utils/api/devices';
import { dockerRestart } from '../utils/docker';

test.describe('Device Enrollment', () => {
  test.beforeEach(async () => {
    await dockerRestart();  // Clean state
  });

  test('Enroll device with valid token', async ({ page }) => {
    // Create enrollment token
    const token = await apiCreateEnrollmentToken(page, {
      site_tag: 'warehouse-1',
      expires_in_seconds: 3600
    });

    expect(token.token).toBeDefined();

    // Enroll device
    const responsePromise = page.waitForResponse('**/v1/enrollments');
    const device = await apiEnrollDevice(page, {
      token: token.token,
      public_key: 'ed25519_test_pubkey_123',
      wireguard_public_key: 'wg_test_pubkey_456',
      platform: 'linux-amd64',
      agent_version: '0.1.0'
    });
    const response = await responsePromise;

    expect(response.status()).toBe(201);
    expect(device.id).toBeDefined();
    expect(device.wireguard_ip).toBeDefined();
    expect(device.status).toBe('ACTIVE');
  });

  test('Enrollment fails with expired token', async ({ page }) => {
    const token = await apiCreateEnrollmentToken(page, {
      expires_in_seconds: -3600  // Expired
    });

    const responsePromise = page.waitForResponse('**/v1/enrollments');

    try {
      await apiEnrollDevice(page, {
        token: token.token,
        public_key: 'test_key',
        wireguard_public_key: 'wg_key',
        platform: 'linux-amd64',
        agent_version: '0.1.0'
      });
    } catch (error) {
      // Expected to fail
    }

    const response = await responsePromise;
    expect(response.status()).toBe(401);
  });

  test('Token single-use enforcement', async ({ page }) => {
    const token = await apiCreateEnrollmentToken(page, {
      max_uses: 1
    });

    // First enrollment succeeds
    await apiEnrollDevice(page, {
      token: token.token,
      public_key: 'key1',
      wireguard_public_key: 'wg_key1',
      platform: 'linux-amd64',
      agent_version: '0.1.0'
    });

    // Second enrollment fails
    const responsePromise = page.waitForResponse('**/v1/enrollments');

    try {
      await apiEnrollDevice(page, {
        token: token.token,
        public_key: 'key2',
        wireguard_public_key: 'wg_key2',
        platform: 'linux-amd64',
        agent_version: '0.1.0'
      });
    } catch (error) {
      // Expected
    }

    const response = await responsePromise;
    expect(response.status()).toBe(401);
  });
});
```

### Rollout Management

**`tests/rollouts.spec.ts`:**

```typescript
import { expect, test } from '@playwright/test';
import { apiUploadArtifact } from '../utils/api/artifacts';
import { apiCreateRollout, apiStartRollout, apiGetRollout } from '../utils/api/rollouts';
import { apiEnrollDevice } from '../utils/api/devices';
import { dockerRestart } from '../utils/docker';

test.describe('Rollout Management', () => {
  test.beforeEach(async () => {
    await dockerRestart();
  });

  test('Create and start canary rollout', async ({ page }) => {
    // Enroll test devices
    for (let i = 0; i < 10; i++) {
      await apiEnrollDevice(page, {
        token: 'test-token',
        public_key: `key_${i}`,
        wireguard_public_key: `wg_key_${i}`,
        platform: 'linux-amd64',
        agent_version: '0.1.0',
        site_tag: 'warehouse-1'
      });
    }

    // Upload artifact
    const artifact = await apiUploadArtifact(page, {
      name: 'app-v1.2.0',
      type: 'BINARY',
      file_content: Buffer.from('test binary').toString('base64')
    });

    // Create rollout
    const rollout = await apiCreateRollout(page, {
      artifact_id: artifact.id,
      target_selector: { site_tag: 'warehouse-1' },
      canary_percent: 20,
      soak_time_seconds: 60,
      health_check_url: 'http://localhost:9090/health'
    });

    expect(rollout.state).toBe('DRAFT');

    // Start rollout
    const responsePromise = page.waitForResponse(`**/v1/rollouts/${rollout.id}/start`);
    await apiStartRollout(page, rollout.id);
    const response = await responsePromise;

    expect(response.status()).toBe(200);

    const updatedRollout = await apiGetRollout(page, rollout.id);
    expect(updatedRollout.state).toBe('CANARY');
  });

  test('Rollback on health check failure', async ({ page }) => {
    // Create rollout with failing health check
    const artifact = await apiUploadArtifact(page, {
      name: 'app-broken',
      type: 'BINARY',
      file_content: Buffer.from('broken').toString('base64')
    });

    const rollout = await apiCreateRollout(page, {
      artifact_id: artifact.id,
      target_selector: {},
      canary_percent: 10,
      soak_time_seconds: 5,
      health_check_url: 'http://localhost:9090/fail'
    });

    await apiStartRollout(page, rollout.id);

    // Wait for health check period + processing
    await page.waitForTimeout(10000);

    const updatedRollout = await apiGetRollout(page, rollout.id);
    expect(updatedRollout.state).toBe('ROLLBACK');
  });
});
```

### Access Sessions

**`tests/access-sessions.spec.ts`:**

```typescript
import { expect, test } from '@playwright/test';
import { apiEnrollDevice } from '../utils/api/devices';
import { apiCreateAccessSession, apiTerminateSession } from '../utils/api/access-sessions';
import { dockerRestart } from '../utils/docker';

test.describe('Access Sessions', () => {
  test.beforeEach(async () => {
    await dockerRestart();
  });

  test('Create and terminate access session', async ({ page }) => {
    const device = await apiEnrollDevice(page, {
      token: 'test-token',
      public_key: 'test_key',
      wireguard_public_key: 'wg_key',
      platform: 'linux-amd64',
      agent_version: '0.1.0'
    });

    // Create session
    const responsePromise = page.waitForResponse('**/v1/access-sessions');
    const session = await apiCreateAccessSession(page, {
      device_id: device.id,
      user_email: 'operator@example.com',
      duration_hours: 2
    });
    const response = await responsePromise;

    expect(response.status()).toBe(201);
    expect(session.wireguard_peer_config).toBeDefined();
    expect(session.expires_at).toBeDefined();

    // Terminate session
    const deletePromise = page.waitForResponse(`**/v1/access-sessions/${session.id}`);
    await apiTerminateSession(page, session.id);
    const deleteResponse = await deletePromise;

    expect(deleteResponse.status()).toBe(204);
  });

  test('Session auto-expiration', async ({ page }) => {
    const device = await apiEnrollDevice(page, {
      token: 'test-token',
      public_key: 'test_key',
      wireguard_public_key: 'wg_key',
      platform: 'linux-amd64',
      agent_version: '0.1.0'
    });

    // Create short-lived session
    const session = await apiCreateAccessSession(page, {
      device_id: device.id,
      user_email: 'operator@example.com',
      duration_hours: 0.001  // ~3 seconds
    });

    // Wait for expiration
    await page.waitForTimeout(5000);

    // Verify session is expired (should return 404 or terminated)
    const responsePromise = page.waitForResponse(`**/v1/access-sessions/${session.id}`);

    try {
      await page.evaluate(async (args) => {
        const [url] = args;
        const response = await fetch(url);
        return response.status;
      }, [`${testsConfig.API_BASE_URL}/access-sessions/${session.id}`]);
    } catch (error) {
      // Expected
    }

    const response = await responsePromise;
    expect([404, 410].includes(response.status())).toBeTruthy();
  });
});
```

### Audit Logging

**`tests/audit.spec.ts`:**

```typescript
import { expect, test } from '@playwright/test';
import { apiEnrollDevice } from '../utils/api/devices';
import { apiGetAuditLogs } from '../utils/api/audit';
import { dockerRestart } from '../utils/docker';

test.describe('Audit Logging', () => {
  test.beforeEach(async () => {
    await dockerRestart();
  });

  test('Device enrollment logged', async ({ page }) => {
    const beforeTime = new Date().toISOString();

    await apiEnrollDevice(page, {
      token: 'test-token',
      public_key: 'test_key',
      wireguard_public_key: 'wg_key',
      platform: 'linux-arm64',
      agent_version: '0.1.0'
    });

    const logs = await apiGetAuditLogs(page, {
      start_time: beforeTime,
      event_type: 'device.enrolled'
    });

    expect(logs.length).toBeGreaterThan(0);
    expect(logs[0].event_type).toBe('device.enrolled');
    expect(logs[0].result).toBe('success');
    expect(logs[0].metadata?.platform).toBe('linux-arm64');
  });

  test('Failed enrollment logged', async ({ page }) => {
    const beforeTime = new Date().toISOString();

    try {
      await apiEnrollDevice(page, {
        token: 'invalid-token',
        public_key: 'test_key',
        wireguard_public_key: 'wg_key',
        platform: 'linux-amd64',
        agent_version: '0.1.0'
      });
    } catch (error) {
      // Expected to fail
    }

    const logs = await apiGetAuditLogs(page, {
      start_time: beforeTime,
      event_type: 'device.enrollment_failed'
    });

    expect(logs.length).toBeGreaterThan(0);
    expect(logs[0].result).toBe('failure');
  });
});
```

### Agent E2E Tests

**`tests/agent.spec.ts`:**

Tests agent running in Docker container connecting to control plane via gRPC.

```typescript
import { expect, test } from '@playwright/test';
import { apiCreateEnrollmentToken } from '../utils/api/enrollment';
import { apiGetDevices } from '../utils/api/devices';
import { startAgent, stopAgent, getAgentLogs } from '../utils/agent';
import { dockerRestart } from '../utils/docker';
import { makeConnection } from '../utils/db/connection';

test.describe('Agent E2E', () => {
  test.beforeEach(async () => {
    await dockerRestart();
  });

  test.afterEach(async () => {
    await stopAgent();
  });

  test('Agent enrolls and sends heartbeat', async ({ page }) => {
    // Create enrollment token
    const token = await apiCreateEnrollmentToken(page, {
      site_tag: 'test-site',
      expires_in_seconds: 3600
    });

    // Start agent container with enrollment token
    await startAgent({
      enrollmentToken: token.token,
      controlPlaneUrl: 'http://control-plane:8080'
    });

    // Wait for enrollment
    await page.waitForTimeout(5000);

    // Verify device enrolled
    const devices = await apiGetDevices(page);
    expect(devices.length).toBe(1);
    expect(devices[0].status).toBe('ACTIVE');
    expect(devices[0].agent_version).toBeDefined();

    // Wait for heartbeat
    await page.waitForTimeout(65000); // 60s + buffer

    // Verify device is online (last_seen_at updated)
    const updatedDevices = await apiGetDevices(page);
    const lastSeen = new Date(updatedDevices[0].last_seen_at);
    const now = new Date();
    const diffSeconds = (now.getTime() - lastSeen.getTime()) / 1000;

    expect(diffSeconds).toBeLessThan(70); // Within heartbeat window
  });

  test('Agent marked offline after no heartbeat', async ({ page }) => {
    const token = await apiCreateEnrollmentToken(page, {
      expires_in_seconds: 3600
    });

    await startAgent({
      enrollmentToken: token.token,
      controlPlaneUrl: 'http://control-plane:8080'
    });

    // Wait for enrollment
    await page.waitForTimeout(5000);

    const devices = await apiGetDevices(page);
    expect(devices[0].status).toBe('ACTIVE');

    // Stop agent
    await stopAgent();

    // Wait for offline detection (5 min + buffer)
    await page.waitForTimeout(310000);

    // Verify device marked offline
    const client = await makeConnection();
    const result = await client.query(
      'SELECT last_seen_at FROM devices WHERE id = $1',
      [devices[0].id]
    );
    await client.end();

    const lastSeen = new Date(result.rows[0].last_seen_at);
    const now = new Date();
    const diffMinutes = (now.getTime() - lastSeen.getTime()) / 60000;

    expect(diffMinutes).toBeGreaterThan(5);
  });

  test('Agent reconnects after control plane restart', async ({ page }) => {
    const token = await apiCreateEnrollmentToken(page, {
      expires_in_seconds: 3600
    });

    await startAgent({
      enrollmentToken: token.token,
      controlPlaneUrl: 'http://control-plane:8080'
    });

    await page.waitForTimeout(5000);

    // Restart control plane
    await dockerRestart();

    // Wait for reconnection
    await page.waitForTimeout(70000); // Reconnect + heartbeat

    // Verify agent reconnected and sent heartbeat
    const devices = await apiGetDevices(page);
    const lastSeen = new Date(devices[0].last_seen_at);
    const now = new Date();
    const diffSeconds = (now.getTime() - lastSeen.getTime()) / 1000;

    expect(diffSeconds).toBeLessThan(120);
  });

  test('Agent logs show successful enrollment', async ({ page }) => {
    const token = await apiCreateEnrollmentToken(page, {
      expires_in_seconds: 3600
    });

    await startAgent({
      enrollmentToken: token.token,
      controlPlaneUrl: 'http://control-plane:8080'
    });

    await page.waitForTimeout(5000);

    const logs = await getAgentLogs();

    expect(logs).toContain('enrollment successful');
    expect(logs).toContain('tunnel established');
    expect(logs).toContain('heartbeat sent');
  });
});
```

---

## Utilities

### API Helpers

**`utils/api/enrollment.ts`:**

```typescript
import { Page } from '@playwright/test';
import { testsConfig } from '../../config';
import { EnrollmentToken } from '../../types';

export const apiCreateEnrollmentToken = async (
  page: Page,
  params: {
    site_tag?: string;
    expires_in_seconds: number;
    max_uses?: number;
  }
): Promise<EnrollmentToken> => {
  return await page.evaluate(async (args) => {
    const [url, data] = args;
    const response = await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
      credentials: 'include'
    });
    return await response.json();
  }, [testsConfig.API_BASE_URL + '/enrollment-tokens', params]);
};
```

**`utils/api/devices.ts`:**

```typescript
import { Page } from '@playwright/test';
import { testsConfig } from '../../config';
import { Device } from '../../types';

export const apiEnrollDevice = async (
  page: Page,
  params: {
    token: string;
    public_key: string;
    wireguard_public_key: string;
    platform: string;
    agent_version: string;
    site_tag?: string;
  }
): Promise<Device> => {
  return await page.evaluate(async (args) => {
    const [url, data] = args;
    const response = await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data)
    });
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    return await response.json();
  }, [testsConfig.API_BASE_URL + '/enrollments', params]);
};

export const apiGetDevices = async (page: Page): Promise<Device[]> => {
  return await page.evaluate(async (url) => {
    const response = await fetch(url, { credentials: 'include' });
    return await response.json();
  }, [testsConfig.API_BASE_URL + '/devices']);
};

export const apiSuspendDevice = async (page: Page, deviceId: string): Promise<void> => {
  await page.evaluate(async (args) => {
    const [url] = args;
    await fetch(url, {
      method: 'POST',
      credentials: 'include'
    });
  }, [`${testsConfig.API_BASE_URL}/devices/${deviceId}/suspend`]);
};
```

### Agent Utilities

**`utils/agent.ts`:**

```typescript
import { execSync, exec } from 'child_process';
import { promisify } from 'util';

const execAsync = promisify(exec);

const AGENT_CONTAINER_NAME = 'safeedge-test-agent';
const AGENT_IMAGE = 'safeedge-agent:test';

export const startAgent = async (config: {
  enrollmentToken: string;
  controlPlaneUrl: string;
}): Promise<void> => {
  // Stop existing agent if running
  await stopAgent();

  // Start agent container
  execSync(`docker run -d \
    --name ${AGENT_CONTAINER_NAME} \
    --network safeedge-e2e \
    -e ENROLLMENT_TOKEN=${config.enrollmentToken} \
    -e CONTROL_PLANE_URL=${config.controlPlaneUrl} \
    ${AGENT_IMAGE}`, { stdio: 'inherit' });

  // Wait for agent to start
  await new Promise(resolve => setTimeout(resolve, 2000));
};

export const stopAgent = async (): Promise<void> => {
  try {
    execSync(`docker stop ${AGENT_CONTAINER_NAME} 2>/dev/null || true`, { stdio: 'ignore' });
    execSync(`docker rm ${AGENT_CONTAINER_NAME} 2>/dev/null || true`, { stdio: 'ignore' });
  } catch (error) {
    // Ignore errors if container doesn't exist
  }
};

export const getAgentLogs = async (): Promise<string> => {
  const { stdout } = await execAsync(`docker logs ${AGENT_CONTAINER_NAME} 2>&1`);
  return stdout;
};

export const restartAgent = async (): Promise<void> => {
  execSync(`docker restart ${AGENT_CONTAINER_NAME}`, { stdio: 'inherit' });
  await new Promise(resolve => setTimeout(resolve, 2000));
};
```

### Database Utilities

**`utils/db/connection.ts`:**

```typescript
import { Client } from 'pg';
import { testsConfig } from '../../config';

export const makeConnection = async (): Promise<Client> => {
  const client = new Client({
    host: testsConfig.DB_HOST,
    port: testsConfig.DB_PORT,
    database: testsConfig.DB_NAME,
    user: testsConfig.DB_USER,
    password: testsConfig.DB_PASSWORD
  });
  await client.connect();
  return client;
};
```

**`utils/db/snapshot.ts`:**

```typescript
import { execSync } from 'child_process';
import { testsConfig } from '../../config';

const pgDumpCmd = `PGPASSWORD=${testsConfig.DB_PASSWORD} pg_dump -h ${testsConfig.DB_HOST} -U ${testsConfig.DB_USER} -d ${testsConfig.DB_NAME}`;
const psqlCmd = `PGPASSWORD=${testsConfig.DB_PASSWORD} psql -h ${testsConfig.DB_HOST} -U ${testsConfig.DB_USER} -d ${testsConfig.DB_NAME}`;

export const createSnapshot = () => {
  execSync(`${pgDumpCmd} > /tmp/safeedge-test-snapshot.sql`);
};

export const restoreSnapshot = () => {
  execSync(`${psqlCmd} < /tmp/safeedge-test-snapshot.sql`);
};
```

### Docker Management

**`utils/docker.ts`:**

```typescript
import { execSync } from 'child_process';
import { restoreSnapshot } from './db/snapshot';

const dockerCompose = 'docker compose -f docker-compose.e2e.yaml';

export const dockerUp = () => {
  execSync(`${dockerCompose} up -d`);
};

export const dockerDown = () => {
  execSync(`${dockerCompose} down`);
};

export const dockerRestart = () => {
  execSync(`${dockerCompose} stop control-plane`);
  restoreSnapshot();
  execSync(`${dockerCompose} start control-plane`);
};
```

### Global Setup/Teardown

**`utils/setup.ts`:**

```typescript
import { FullConfig } from '@playwright/test';
import { dockerUp } from './docker';
import { createSnapshot } from './db/snapshot';

const globalSetup = async (_: FullConfig) => {
  dockerUp();
  // Wait for services to be ready
  await new Promise(resolve => setTimeout(resolve, 5000));
  createSnapshot();
};

export default globalSetup;
```

**`utils/teardown.ts`:**

```typescript
import { FullConfig } from '@playwright/test';
import { dockerDown } from './docker';

const globalTeardown = async (_: FullConfig) => {
  dockerDown();
};

export default globalTeardown;
```

---

## Running Tests

```bash
# All tests
npm test

# Specific test file
npm test tests/enrollment.spec.ts

# Headed mode (see browser)
npm run test:headed

# Debug mode (step through)
npm run test:debug

# UI mode (interactive)
npm run test:ui

# Watch mode
npm test -- --watch
```

---

## Docker Compose for Testing

**`docker-compose.e2e.yaml`:**

```yaml
version: '3.8'

networks:
  safeedge-e2e:
    driver: bridge

services:
  postgres:
    image: postgres:15
    networks:
      - safeedge-e2e
    environment:
      POSTGRES_DB: safeedge_test
      POSTGRES_USER: safeedge
      POSTGRES_PASSWORD: safeedge
    ports:
      - "5433:5432"

  redis:
    image: redis:7-alpine
    networks:
      - safeedge-e2e
    ports:
      - "6380:6379"

  minio:
    image: minio/minio
    command: server /data
    networks:
      - safeedge-e2e
    environment:
      MINIO_ROOT_USER: safeedge
      MINIO_ROOT_PASSWORD: safeedge123
    ports:
      - "9001:9000"

  control-plane:
    build:
      context: .
      dockerfile: Dockerfile
    networks:
      - safeedge-e2e
    ports:
      - "8080:8080"   # REST API + gRPC
      - "51820:51820/udp"  # WireGuard
    environment:
      DATABASE_URL: postgres://safeedge:safeedge@postgres:5432/safeedge_test
      REDIS_URL: redis://redis:6379
      S3_ENDPOINT: http://minio:9000
      LOG_LEVEL: debug
    depends_on:
      - postgres
      - redis
      - minio

# Note: Agent container is started dynamically by tests via utils/agent.ts
# Agent connects to control-plane via safeedge-e2e network
```

**Agent Dockerfile (`Dockerfile.agent`):**

```dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags="-s -w" -o safeedge-agent ./cmd/agent

FROM alpine:latest
RUN apk --no-cache add ca-certificates wireguard-tools
COPY --from=builder /build/safeedge-agent /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/safeedge-agent"]
CMD ["run"]
```

**Build Agent Image:**

```bash
docker build -f Dockerfile.agent -t safeedge-agent:test .
```

---

## CI Integration

**`.github/workflows/e2e.yml`:**

```yaml
name: E2E Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-node@v4
        with:
          node-version: '20'

      - name: Install dependencies
        working-directory: e2e
        run: npm ci

      - name: Install Playwright
        working-directory: e2e
        run: npx playwright install --with-deps chromium

      - name: Start test infrastructure
        run: docker compose -f docker-compose.e2e.yaml up -d

      - name: Run tests
        working-directory: e2e
        run: npm test

      - name: Upload test results
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: playwright-report
          path: e2e/playwright-report/
```
