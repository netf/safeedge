-- name: CreateEnrollmentToken :one
INSERT INTO enrollment_tokens (
  organization_id,
  token_hash,
  site_tag,
  expires_at,
  max_uses
) VALUES (
  $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetEnrollmentTokenByHash :one
SELECT * FROM enrollment_tokens
WHERE token_hash = $1
  AND expires_at > NOW()
  AND used_count < max_uses;

-- name: IncrementTokenUsage :one
UPDATE enrollment_tokens
SET used_count = used_count + 1
WHERE id = $1
RETURNING *;

-- name: ListEnrollmentTokens :many
SELECT * FROM enrollment_tokens
WHERE organization_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: DeleteExpiredTokens :exec
DELETE FROM enrollment_tokens
WHERE expires_at < NOW() AND used_count < max_uses;
