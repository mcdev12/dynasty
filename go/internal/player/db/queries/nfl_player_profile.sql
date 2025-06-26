-- name: CreateNFLPlayerProfile :one
INSERT INTO nfl_player_profiles (
    player_id,
    height_cm,
    weight_kg,
    group_role,
    position,
    age,
    height_desc,
    weight_desc,
    college,
    jersey_number,
    salary_desc,
    experience
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
    $11,
    $12
) RETURNING *;

-- name: GetNFLPlayerProfile :one
SELECT * FROM nfl_player_profiles WHERE player_id = $1;

-- name: GetNFLPlayerProfileByExternalID :one
SELECT npp.* FROM nfl_player_profiles npp
JOIN players p ON npp.player_id = p.id
WHERE p.sport_id = $1 AND p.external_id = $2;

-- name: DeleteNFLPlayerProfile :exec
DELETE FROM nfl_player_profiles WHERE player_id = $1;