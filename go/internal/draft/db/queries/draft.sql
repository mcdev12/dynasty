-- name: CreateDraft :one
INSERT INTO draft (
    id,
    league_id,
    draft_type,
    status,
    settings,
    scheduled_at,
    created_at,
    updated_at
) VALUES (
             $1, -- id
             $2, -- league_id
             $3, -- draft_type
             $4, -- status
             $5, -- settings
             $6, -- scheduled_at
             NOW(),
             NOW()
         )
RETURNING *;

-- name: GetDraft :one
SELECT *
FROM draft
WHERE id = $1;

-- name: UpdateDraftStatus :one
UPDATE draft
SET
    status = $2,
    started_at = CASE WHEN $2::text = 'IN_PROGRESS' THEN NOW() ELSE started_at END,
    completed_at = CASE WHEN $2::text = 'COMPLETED' THEN NOW() ELSE completed_at END,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteDraft :exec
DELETE FROM draft
WHERE id = $1
  AND status = 'NOT_STARTED';
