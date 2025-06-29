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
    started_at = CASE WHEN $2 = 'IN_PROGRESS'::draft_status THEN NOW() ELSE started_at END,
    completed_at = CASE WHEN $2 = 'COMPLETED'::draft_status THEN NOW() ELSE completed_at END,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteDraft :exec
DELETE FROM draft
WHERE id = $1
  AND status = 'NOT_STARTED';


-- name: FetchNextDeadline :one
-- Fetch the single soonest deadline across all in-progress drafts.
SELECT
    id      AS draft_id,
    next_deadline
FROM draft
WHERE status = 'IN_PROGRESS'
ORDER BY next_deadline
LIMIT 1;

-- name: FetchDraftsDueForPick :many
-- Claim up to $1 drafts whose deadline has passed, locking them to avoid races.
SELECT
    id AS draft_id
FROM draft
WHERE status = 'IN_PROGRESS'
  AND next_deadline <= NOW()
ORDER BY next_deadline
LIMIT $1
    FOR UPDATE SKIP LOCKED;

-- name: UpdateNextDeadline :exec
-- Set the next pick deadline for a draft (e.g. after a pick or resume).
UPDATE draft
SET next_deadline = $2
WHERE id = $1;

-- name: ClearNextDeadline :exec
-- Clear the deadline (e.g. when pausing or completing a draft).
UPDATE draft
SET next_deadline = NULL
WHERE id = $1;

-- name: UpdateDraft :one
-- Update draft settings and/or scheduled_at
UPDATE draft
SET
    settings = COALESCE($2, settings),
    scheduled_at = COALESCE($3, scheduled_at),
    updated_at = NOW()
WHERE id = $1
RETURNING *;
