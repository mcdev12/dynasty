-- name: CreateDraftPick :one
INSERT INTO draft_picks (
    id,
    draft_id,
    round,
    pick,
    overall_pick,
    team_id,
    player_id,
    picked_at,
    auction_amount,
    keeper_pick
) VALUES (
    $1, -- id
    $2, -- draft_id
    $3, -- round
    $4, -- pick
    $5, -- overall_pick
    $6, -- team_id
    $7, -- player_id
    $8, -- picked_at
    $9, -- auction_amount
    $10 -- keeper_pick
) RETURNING *;

-- name: CreateDraftPickBatch :exec
INSERT INTO draft_picks (
    id,
    draft_id,
    round,
    pick,
    overall_pick,
    team_id
) 
SELECT 
    unnest($1::uuid[]) as id,
    unnest($2::uuid[]) as draft_id,
    unnest($3::integer[]) as round,
    unnest($4::integer[]) as pick,
    unnest($5::integer[]) as overall_pick,
    unnest($6::uuid[]) as team_id;

-- name: GetDraftPick :one
SELECT * FROM draft_picks WHERE id = $1;

-- name: GetDraftPicksByDraft :many
SELECT * FROM draft_picks 
WHERE draft_id = $1 
ORDER BY overall_pick;

-- name: GetDraftPicksByRound :many
SELECT * FROM draft_picks 
WHERE draft_id = $1 AND round = $2 
ORDER BY pick;

-- name: GetNextPickForDraft :one
SELECT * FROM draft_picks 
WHERE draft_id = $1 AND player_id IS NULL 
ORDER BY overall_pick 
LIMIT 1;

-- name: UpdateDraftPickPlayer :one
UPDATE draft_picks SET
    player_id = $2,
    picked_at = NOW(),
    auction_amount = $3,
    keeper_pick = $4
WHERE id = $1
RETURNING *;

-- name: DeleteDraftPicksByDraft :exec
DELETE FROM draft_picks WHERE draft_id = $1;

-- name: MakePick :execrows
UPDATE draft_picks
SET player_id = $2, picked_at = NOW()
WHERE id = $1
  AND player_id IS NULL;