-- name: CreateArtifact :one
INSERT INTO artifacts (
  organization_id,
  name,
  type,
  blake3_hash,
  signature,
  signing_key_id,
  s3_url,
  size_bytes
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: GetArtifact :one
SELECT * FROM artifacts
WHERE id = $1;

-- name: GetArtifactByHash :one
SELECT * FROM artifacts
WHERE blake3_hash = $1;

-- name: ListArtifacts :many
SELECT * FROM artifacts
WHERE organization_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
