-- name: CreateAccessSession :one
INSERT INTO access_sessions (
  device_id,
  user_email,
  wireguard_peer_config,
  expires_at
) VALUES (
  $1, $2, $3, $4
)
RETURNING *;

-- name: GetAccessSession :one
SELECT * FROM access_sessions
WHERE id = $1 AND terminated_at IS NULL;

-- name: ListActiveAccessSessions :many
SELECT * FROM access_sessions
WHERE device_id = $1
  AND terminated_at IS NULL
  AND expires_at > NOW()
ORDER BY created_at DESC;

-- name: TerminateAccessSession :one
UPDATE access_sessions
SET terminated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ExpireOldSessions :exec
UPDATE access_sessions
SET terminated_at = NOW()
WHERE expires_at < NOW()
  AND terminated_at IS NULL;
