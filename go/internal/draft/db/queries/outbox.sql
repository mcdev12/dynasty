-- name: InsertOutboxPickMade :exec
INSERT INTO draft_outbox (id, draft_id, event_type, payload)
VALUES ($1, $2, 'PickMade', $3);

-- name: FetchUnsentOutbox :many
SELECT id, draft_id, event_type, payload
FROM draft_outbox
WHERE sent_at IS NULL
ORDER BY created_at
LIMIT $1
    FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxSent :exec
UPDATE draft_outbox
SET sent_at = NOW()
WHERE id = $1;


-- name: FetchOutboxByID :one
SELECT
    id,
    draft_id,
    event_type,
    payload
FROM draft_outbox
WHERE id = $1
  AND sent_at IS NULL
    FOR UPDATE SKIP LOCKED;