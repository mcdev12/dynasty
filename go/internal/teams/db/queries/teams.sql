-- name: CreateTeam :one
INSERT INTO teams (
    id,
    sport_id,
    external_id,
    name,
    code,
    city,
    coach,
    owner,
    stadium,
    established_year
) VALUES (
    gen_random_uuid(),
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9
) RETURNING *;

-- name: GetTeam :one
SELECT * FROM teams WHERE id = $1;

-- name: GetTeamByExternalID :one
SELECT * FROM teams WHERE sport_id = $1 AND external_id = $2;

-- name: GetTeamBySportIdAndAlias :one
SELECT * FROM teams WHERE sport_id = $1 AND code = $2;

-- name: ListTeamsBySport :many
SELECT * FROM teams WHERE sport_id = $1 ORDER BY name;

-- name: ListAllTeams :many
SELECT * FROM teams ORDER BY sport_id, name;

-- name: UpdateTeam :one
UPDATE teams SET
    name = $2,
    code = $3,
    city = $4,
    coach = $5,
    owner = $6,
    stadium = $7,
    established_year = $8
WHERE id = $1
RETURNING *;

-- name: DeleteTeam :exec
DELETE FROM teams WHERE id = $1;