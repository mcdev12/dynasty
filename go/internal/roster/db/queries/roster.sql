-- name: CreateRoster :one
INSERT INTO roster (
    id,
    fantasy_team_id,
    player_id,
    position,
    acquired_at,
    acquisition_type,
    keeper_data
) VALUES (
    gen_random_uuid(),
    $1,
    $2,
    $3,
    NOW(),
    $4,
    $5
) RETURNING *;

-- name: GetRoster :one
SELECT * FROM roster WHERE id = $1;

-- name: GetRosterPlayersByFantasyTeam :many
SELECT * FROM roster WHERE fantasy_team_id = $1
ORDER BY position, acquired_at;

-- name: GetRosterPlayersByFantasyTeamAndPosition :many
SELECT * FROM roster 
WHERE fantasy_team_id = $1 AND position = $2
ORDER BY acquired_at;

-- name: GetPlayerOnRoster :one
SELECT * FROM roster 
WHERE fantasy_team_id = $1 AND player_id = $2;

-- name: GetStartingRosterPlayers :many
SELECT * FROM roster 
WHERE fantasy_team_id = $1 AND position = 'STARTER'
ORDER BY acquired_at;

-- name: GetBenchRosterPlayers :many
SELECT * FROM roster 
WHERE fantasy_team_id = $1 AND position = 'BENCH'
ORDER BY acquired_at;

-- name: GetRosterPlayersByAcquisitionType :many
SELECT * FROM roster 
WHERE fantasy_team_id = $1 AND acquisition_type = $2
ORDER BY acquired_at;

-- name: UpdateRosterPlayerPosition :one
UPDATE roster SET
    position = $2
WHERE id = $1
RETURNING *;

-- name: UpdateRosterPlayerKeeperData :one
UPDATE roster SET
    keeper_data = $2
WHERE id = $1
RETURNING *;

-- name: UpdateRosterPositionAndKeeperData :one
UPDATE roster SET
    position = $2,
    keeper_data = $3
WHERE id = $1
RETURNING *;

-- name: DeleteRosterEntry :exec
DELETE FROM roster WHERE id = $1;

-- name: DeletePlayerFromRoster :exec
DELETE FROM roster 
WHERE fantasy_team_id = $1 AND player_id = $2;

-- name: DeleteTeamRoster :exec
DELETE FROM roster WHERE fantasy_team_id = $1;