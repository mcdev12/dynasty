// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0
// source: outbox.sql

package db

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

const fetchOutboxByID = `-- name: FetchOutboxByID :one
SELECT
    id,
    draft_id,
    event_type,
    payload
FROM draft_outbox
WHERE id = $1
  AND sent_at IS NULL
    FOR UPDATE SKIP LOCKED
`

type FetchOutboxByIDRow struct {
	ID        uuid.UUID       `json:"id"`
	DraftID   uuid.UUID       `json:"draft_id"`
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
}

func (q *Queries) FetchOutboxByID(ctx context.Context, id uuid.UUID) (FetchOutboxByIDRow, error) {
	row := q.db.QueryRowContext(ctx, fetchOutboxByID, id)
	var i FetchOutboxByIDRow
	err := row.Scan(
		&i.ID,
		&i.DraftID,
		&i.EventType,
		&i.Payload,
	)
	return i, err
}

const fetchUnsentOutbox = `-- name: FetchUnsentOutbox :many
SELECT id, draft_id, event_type, payload
FROM draft_outbox
WHERE sent_at IS NULL
ORDER BY created_at
LIMIT $1
    FOR UPDATE SKIP LOCKED
`

type FetchUnsentOutboxRow struct {
	ID        uuid.UUID       `json:"id"`
	DraftID   uuid.UUID       `json:"draft_id"`
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
}

func (q *Queries) FetchUnsentOutbox(ctx context.Context, limit int32) ([]FetchUnsentOutboxRow, error) {
	rows, err := q.db.QueryContext(ctx, fetchUnsentOutbox, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []FetchUnsentOutboxRow
	for rows.Next() {
		var i FetchUnsentOutboxRow
		if err := rows.Scan(
			&i.ID,
			&i.DraftID,
			&i.EventType,
			&i.Payload,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const insertOutboxDraftCompleted = `-- name: InsertOutboxDraftCompleted :exec
INSERT INTO draft_outbox (id, draft_id, event_type, payload)
VALUES ($1, $2, 'DraftCompleted', $3)
`

type InsertOutboxDraftCompletedParams struct {
	ID      uuid.UUID       `json:"id"`
	DraftID uuid.UUID       `json:"draft_id"`
	Payload json.RawMessage `json:"payload"`
}

func (q *Queries) InsertOutboxDraftCompleted(ctx context.Context, arg InsertOutboxDraftCompletedParams) error {
	_, err := q.db.ExecContext(ctx, insertOutboxDraftCompleted, arg.ID, arg.DraftID, arg.Payload)
	return err
}

const insertOutboxDraftPaused = `-- name: InsertOutboxDraftPaused :exec
INSERT INTO draft_outbox (id, draft_id, event_type, payload)
VALUES ($1, $2, 'DraftPaused', $3)
`

type InsertOutboxDraftPausedParams struct {
	ID      uuid.UUID       `json:"id"`
	DraftID uuid.UUID       `json:"draft_id"`
	Payload json.RawMessage `json:"payload"`
}

func (q *Queries) InsertOutboxDraftPaused(ctx context.Context, arg InsertOutboxDraftPausedParams) error {
	_, err := q.db.ExecContext(ctx, insertOutboxDraftPaused, arg.ID, arg.DraftID, arg.Payload)
	return err
}

const insertOutboxDraftResumed = `-- name: InsertOutboxDraftResumed :exec
INSERT INTO draft_outbox (id, draft_id, event_type, payload)
VALUES ($1, $2, 'DraftResumed', $3)
`

type InsertOutboxDraftResumedParams struct {
	ID      uuid.UUID       `json:"id"`
	DraftID uuid.UUID       `json:"draft_id"`
	Payload json.RawMessage `json:"payload"`
}

func (q *Queries) InsertOutboxDraftResumed(ctx context.Context, arg InsertOutboxDraftResumedParams) error {
	_, err := q.db.ExecContext(ctx, insertOutboxDraftResumed, arg.ID, arg.DraftID, arg.Payload)
	return err
}

const insertOutboxDraftStarted = `-- name: InsertOutboxDraftStarted :exec
INSERT INTO draft_outbox (id, draft_id, event_type, payload)
VALUES ($1, $2, 'DraftStarted', $3)
`

type InsertOutboxDraftStartedParams struct {
	ID      uuid.UUID       `json:"id"`
	DraftID uuid.UUID       `json:"draft_id"`
	Payload json.RawMessage `json:"payload"`
}

func (q *Queries) InsertOutboxDraftStarted(ctx context.Context, arg InsertOutboxDraftStartedParams) error {
	_, err := q.db.ExecContext(ctx, insertOutboxDraftStarted, arg.ID, arg.DraftID, arg.Payload)
	return err
}

const insertOutboxPickMade = `-- name: InsertOutboxPickMade :exec
INSERT INTO draft_outbox (id, draft_id, event_type, payload)
VALUES ($1, $2, 'PickMade', $3)
`

type InsertOutboxPickMadeParams struct {
	ID      uuid.UUID       `json:"id"`
	DraftID uuid.UUID       `json:"draft_id"`
	Payload json.RawMessage `json:"payload"`
}

func (q *Queries) InsertOutboxPickMade(ctx context.Context, arg InsertOutboxPickMadeParams) error {
	_, err := q.db.ExecContext(ctx, insertOutboxPickMade, arg.ID, arg.DraftID, arg.Payload)
	return err
}

const insertOutboxPickStarted = `-- name: InsertOutboxPickStarted :exec
INSERT INTO draft_outbox (id, draft_id, event_type, payload)
VALUES ($1, $2, 'PickStarted', $3)
`

type InsertOutboxPickStartedParams struct {
	ID      uuid.UUID       `json:"id"`
	DraftID uuid.UUID       `json:"draft_id"`
	Payload json.RawMessage `json:"payload"`
}

func (q *Queries) InsertOutboxPickStarted(ctx context.Context, arg InsertOutboxPickStartedParams) error {
	_, err := q.db.ExecContext(ctx, insertOutboxPickStarted, arg.ID, arg.DraftID, arg.Payload)
	return err
}

const markOutboxSent = `-- name: MarkOutboxSent :exec
UPDATE draft_outbox
SET sent_at = NOW()
WHERE id = $1
`

func (q *Queries) MarkOutboxSent(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, markOutboxSent, id)
	return err
}
