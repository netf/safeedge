-- SafeEdge Database Schema
-- PostgreSQL 15+

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Organizations table
CREATE TABLE organizations (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Enrollment tokens for device enrollment
CREATE TABLE enrollment_tokens (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL UNIQUE,
  site_tag TEXT,
  expires_at TIMESTAMPTZ NOT NULL,
  max_uses INTEGER NOT NULL DEFAULT 1,
  used_count INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_enrollment_tokens_org ON enrollment_tokens(organization_id);
CREATE INDEX idx_enrollment_tokens_expires ON enrollment_tokens(expires_at) WHERE used_count < max_uses;

-- Devices (edge/IoT devices)
CREATE TABLE devices (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  public_key TEXT NOT NULL UNIQUE,
  wireguard_public_key TEXT NOT NULL UNIQUE,
  wireguard_ip INET NOT NULL,
  agent_version TEXT NOT NULL,
  platform TEXT NOT NULL,
  site_tag TEXT,
  status TEXT NOT NULL CHECK (status IN ('ACTIVE', 'SUSPENDED', 'DECOMMISSIONED')),
  last_seen_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_devices_org ON devices(organization_id);
CREATE INDEX idx_devices_status ON devices(status);
CREATE INDEX idx_devices_last_seen ON devices(last_seen_at);
CREATE INDEX idx_devices_site_tag ON devices(site_tag) WHERE site_tag IS NOT NULL;

-- Remote access sessions
CREATE TABLE access_sessions (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  user_email TEXT NOT NULL,
  wireguard_peer_config TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  terminated_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_access_sessions_device ON access_sessions(device_id);
CREATE INDEX idx_access_sessions_expires ON access_sessions(expires_at) WHERE terminated_at IS NULL;

-- Artifacts (software updates, container images, binaries)
CREATE TABLE artifacts (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  type TEXT NOT NULL CHECK (type IN ('CONTAINER', 'BINARY')),
  blake3_hash TEXT NOT NULL UNIQUE,
  signature BYTEA NOT NULL,
  signing_key_id TEXT NOT NULL,
  s3_url TEXT NOT NULL,
  size_bytes BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_artifacts_org ON artifacts(organization_id);
CREATE INDEX idx_artifacts_hash ON artifacts(blake3_hash);

-- Rollouts (phased deployment of artifacts)
CREATE TABLE rollouts (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  artifact_id UUID NOT NULL REFERENCES artifacts(id) ON DELETE RESTRICT,
  target_selector JSONB NOT NULL,
  canary_percent INTEGER NOT NULL DEFAULT 10 CHECK (canary_percent >= 0 AND canary_percent <= 100),
  soak_time_seconds INTEGER NOT NULL DEFAULT 300 CHECK (soak_time_seconds >= 0),
  health_check_url TEXT NOT NULL,
  state TEXT NOT NULL CHECK (state IN ('DRAFT', 'CANARY', 'FULL', 'COMPLETE', 'ROLLBACK', 'FAILED')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ
);

CREATE INDEX idx_rollouts_org ON rollouts(organization_id);
CREATE INDEX idx_rollouts_state ON rollouts(state);
CREATE INDEX idx_rollouts_artifact ON rollouts(artifact_id);

-- Rollout device status (tracking rollout progress per device)
CREATE TABLE rollout_device_status (
  rollout_id UUID NOT NULL REFERENCES rollouts(id) ON DELETE CASCADE,
  device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  is_canary BOOLEAN NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('PENDING', 'IN_PROGRESS', 'HEALTHY', 'UNHEALTHY', 'ROLLED_BACK')),
  health_check_result JSONB,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (rollout_id, device_id)
);

CREATE INDEX idx_rollout_device_status_rollout ON rollout_device_status(rollout_id);
CREATE INDEX idx_rollout_device_status_device ON rollout_device_status(device_id);

-- Audit logs for compliance and debugging
CREATE TABLE audit_logs (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  user_email TEXT,
  event_type TEXT NOT NULL,
  resource_type TEXT NOT NULL,
  resource_id TEXT NOT NULL,
  action TEXT NOT NULL,
  result TEXT NOT NULL CHECK (result IN ('success', 'failure')),
  metadata JSONB,
  ip_address INET
);

CREATE INDEX idx_audit_logs_org ON audit_logs(organization_id);
CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp);
CREATE INDEX idx_audit_logs_event_type ON audit_logs(event_type);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
