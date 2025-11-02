-- name: CreateRolloutDeviceStatus :one
INSERT INTO rollout_device_status (
  rollout_id,
  device_id,
  is_canary,
  status
) VALUES (
  $1, $2, $3, 'PENDING'
)
RETURNING *;

-- name: GetRolloutDeviceStatus :one
SELECT * FROM rollout_device_status
WHERE rollout_id = $1 AND device_id = $2;

-- name: ListRolloutDeviceStatuses :many
SELECT * FROM rollout_device_status
WHERE rollout_id = $1
ORDER BY is_canary DESC, updated_at ASC;

-- name: UpdateRolloutDeviceStatus :one
UPDATE rollout_device_status
SET status = $3, health_check_result = $4, updated_at = NOW()
WHERE rollout_id = $1 AND device_id = $2
RETURNING *;

-- name: CountRolloutDevicesByStatus :one
SELECT COUNT(*) FROM rollout_device_status
WHERE rollout_id = $1 AND status = $2;

-- name: GetCanaryDevices :many
SELECT * FROM rollout_device_status
WHERE rollout_id = $1 AND is_canary = true
ORDER BY updated_at ASC;
