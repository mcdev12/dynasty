-- name: CreateNFLPlayerProfile :one
INSERT INTO nfl_player_profiles (
    player_id,
    position,
    status,
    college,
    jersey_number,
    experience,
    birth_date,
    height_cm,
    weight_kg,
    height_desc,
    weight_desc
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9,
    $10,
    $11
) RETURNING *;

-- name: GetNFLPlayerProfile :one
SELECT * FROM nfl_player_profiles WHERE player_id = $1;

-- name: GetNFLPlayerProfileByExternalID :one
SELECT npp.* FROM nfl_player_profiles npp
JOIN players p ON npp.player_id = p.id
WHERE p.sport_id = $1 AND p.external_id = $2;

-- name: UpdateNFLPlayerProfile :one
UPDATE nfl_player_profiles SET
    position = $2,
    status = $3,
    college = $4,
    jersey_number = $5,
    experience = $6,
    birth_date = $7,
    height_cm = $8,
    weight_kg = $9,
    height_desc = $10,
    weight_desc = $11
WHERE player_id = $1
RETURNING *;

-- name: DeleteNFLPlayerProfile :exec
DELETE FROM nfl_player_profiles WHERE player_id = $1;