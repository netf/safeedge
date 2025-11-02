-- name: CreateRollout :one
INSERT INTO rollouts (
  organization_id,
  artifact_id,
  target_selector,
  canary_percent,
  soak_time_seconds,
  health_check_url,
  state
) VALUES (
  $1, $2, $3, $4, $5, $6, 'DRAFT'
)
RETURNING *;

-- name: GetRollout :one
SELECT * FROM rollouts
WHERE id = $1;

-- name: ListRollouts :many
SELECT * FROM rollouts
WHERE organization_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateRolloutState :one
UPDATE rollouts
SET state = $2, started_at = CASE WHEN started_at IS NULL THEN NOW() ELSE started_at END
WHERE id = $1
RETURNING *;

-- name: CompleteRollout :one
UPDATE rollouts
SET state = 'COMPLETE', completed_at = NOW()
WHERE id = $1
RETURNING *;

-- name: FailRollout :one
UPDATE rollouts
SET state = 'FAILED', completed_at = NOW()
WHERE id = $1
RETURNING *;
