-- name: CreateDevice :one
INSERT INTO devices (
  organization_id,
  public_key,
  wireguard_public_key,
  wireguard_ip,
  agent_version,
  platform,
  site_tag,
  status
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, 'ACTIVE'
)
RETURNING *;

-- name: GetDevice :one
SELECT * FROM devices
WHERE id = $1;

-- name: GetDeviceByPublicKey :one
SELECT * FROM devices
WHERE public_key = $1;

-- name: ListDevices :many
SELECT * FROM devices
WHERE organization_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListDevicesBySiteTag :many
SELECT * FROM devices
WHERE organization_id = $1 AND site_tag = $2
ORDER BY created_at DESC;

-- name: UpdateDeviceStatus :one
UPDATE devices
SET status = $2
WHERE id = $1
RETURNING *;

-- name: UpdateDeviceHeartbeat :one
UPDATE devices
SET last_seen_at = NOW()
WHERE id = $1
RETURNING *;

-- name: CountDevicesByStatus :one
SELECT COUNT(*) FROM devices
WHERE organization_id = $1 AND status = $2;

-- name: GetStaleDevices :many
SELECT * FROM devices
WHERE status = 'ACTIVE'
  AND last_seen_at < NOW() - INTERVAL '5 minutes'
ORDER BY last_seen_at ASC;
