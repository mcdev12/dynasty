-- name: CreatePlayer :one
INSERT INTO players (
    id,
    sport_id,
    external_id,
    full_name,
    team_id
) VALUES (
    gen_random_uuid(),
    $1,
    $2,
    $3,
    $4
) RETURNING *;

-- name: GetPlayer :one
SELECT * FROM players WHERE id = $1;

-- name: GetPlayerByExternalID :one
SELECT * FROM players WHERE sport_id = $1 AND external_id = $2;

-- name: UpdatePlayer :one
UPDATE players SET
    full_name = $2,
    team_id = $3
WHERE id = $1
RETURNING *;

-- name: DeletePlayer :exec
DELETE FROM players WHERE id = $1;