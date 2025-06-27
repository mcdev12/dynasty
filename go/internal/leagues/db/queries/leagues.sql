-- name: CreateLeague :one
INSERT INTO leagues (
    name,
    sport_id,
    league_type,
    commissioner_id,
    league_settings,
    status,
    season
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7
) RETURNING *;

-- name: GetLeague :one
SELECT * FROM leagues WHERE id = $1;

-- name: GetLeaguesByCommissioner :many
SELECT * FROM leagues WHERE commissioner_id = $1 ORDER BY created_at DESC;

-- name: UpdateLeague :one
UPDATE leagues SET
    name = $2,
    sport_id = $3,
    league_type = $4,
    commissioner_id = $5,
    league_settings = $6,
    status = $7,
    season = $8,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateLeagueStatus :one
UPDATE leagues SET
    status = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateLeagueSettings :one
UPDATE leagues SET
    league_settings = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteLeague :exec
DELETE FROM leagues WHERE id = $1;