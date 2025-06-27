-- name: CreateFantasyTeam :one
INSERT INTO fantasy_teams (
    id,
    league_id,
    owner_id,
    name,
    logo_url,
    created_at
) VALUES (
    gen_random_uuid(),
    $1,
    $2,
    $3,
    $4,
    NOW()
) RETURNING *;

-- name: GetFantasyTeam :one
SELECT * FROM fantasy_teams WHERE id = $1;

-- name: GetFantasyTeamsByLeague :many
SELECT * FROM fantasy_teams WHERE league_id = $1;

-- name: GetFantasyTeamsByOwner :many
SELECT * FROM fantasy_teams WHERE owner_id = $1;

-- name: GetFantasyTeamByLeagueAndOwner :one
SELECT * FROM fantasy_teams WHERE owner_id = $1 and league_id = $2;

-- name: UpdateFantasyTeam :one
UPDATE fantasy_teams SET
    name = $2,
    logo_url = $3
WHERE id = $1
RETURNING *;

-- name: DeleteFantasyTeam :exec
DELETE FROM fantasy_teams WHERE id = $1;