-- name: CreateAuditLog :one
INSERT INTO audit_logs (
  organization_id,
  user_email,
  event_type,
  resource_type,
  resource_id,
  action,
  result,
  metadata,
  ip_address
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

-- name: ListAuditLogs :many
SELECT * FROM audit_logs
WHERE organization_id = $1
  AND ($2::timestamptz IS NULL OR timestamp >= $2)
  AND ($3::timestamptz IS NULL OR timestamp <= $3)
  AND ($4::text IS NULL OR event_type = $4)
ORDER BY timestamp DESC
LIMIT $5 OFFSET $6;

-- name: GetAuditLogsByResource :many
SELECT * FROM audit_logs
WHERE resource_type = $1 AND resource_id = $2
ORDER BY timestamp DESC
LIMIT $3;

-- name: DeleteOldAuditLogs :exec
DELETE FROM audit_logs
WHERE timestamp < NOW() - INTERVAL '90 days';
