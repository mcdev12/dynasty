// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0
// source: nfl_player_profile.sql

package db

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

const createNFLPlayerProfile = `-- name: CreateNFLPlayerProfile :one
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
) RETURNING player_id, position, status, college, jersey_number, experience, birth_date, height_cm, weight_kg, height_desc, weight_desc
`

type CreateNFLPlayerProfileParams struct {
	PlayerID     uuid.UUID      `json:"player_id"`
	Position     sql.NullString `json:"position"`
	Status       sql.NullString `json:"status"`
	College      sql.NullString `json:"college"`
	JerseyNumber sql.NullInt16  `json:"jersey_number"`
	Experience   sql.NullInt16  `json:"experience"`
	BirthDate    sql.NullTime   `json:"birth_date"`
	HeightCm     sql.NullInt32  `json:"height_cm"`
	WeightKg     sql.NullInt32  `json:"weight_kg"`
	HeightDesc   sql.NullString `json:"height_desc"`
	WeightDesc   sql.NullString `json:"weight_desc"`
}

func (q *Queries) CreateNFLPlayerProfile(ctx context.Context, arg CreateNFLPlayerProfileParams) (NflPlayerProfile, error) {
	row := q.db.QueryRowContext(ctx, createNFLPlayerProfile,
		arg.PlayerID,
		arg.Position,
		arg.Status,
		arg.College,
		arg.JerseyNumber,
		arg.Experience,
		arg.BirthDate,
		arg.HeightCm,
		arg.WeightKg,
		arg.HeightDesc,
		arg.WeightDesc,
	)
	var i NflPlayerProfile
	err := row.Scan(
		&i.PlayerID,
		&i.Position,
		&i.Status,
		&i.College,
		&i.JerseyNumber,
		&i.Experience,
		&i.BirthDate,
		&i.HeightCm,
		&i.WeightKg,
		&i.HeightDesc,
		&i.WeightDesc,
	)
	return i, err
}

const deleteNFLPlayerProfile = `-- name: DeleteNFLPlayerProfile :exec
DELETE FROM nfl_player_profiles WHERE player_id = $1
`

func (q *Queries) DeleteNFLPlayerProfile(ctx context.Context, playerID uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, deleteNFLPlayerProfile, playerID)
	return err
}

const getNFLPlayerProfile = `-- name: GetNFLPlayerProfile :one
SELECT player_id, position, status, college, jersey_number, experience, birth_date, height_cm, weight_kg, height_desc, weight_desc FROM nfl_player_profiles WHERE player_id = $1
`

func (q *Queries) GetNFLPlayerProfile(ctx context.Context, playerID uuid.UUID) (NflPlayerProfile, error) {
	row := q.db.QueryRowContext(ctx, getNFLPlayerProfile, playerID)
	var i NflPlayerProfile
	err := row.Scan(
		&i.PlayerID,
		&i.Position,
		&i.Status,
		&i.College,
		&i.JerseyNumber,
		&i.Experience,
		&i.BirthDate,
		&i.HeightCm,
		&i.WeightKg,
		&i.HeightDesc,
		&i.WeightDesc,
	)
	return i, err
}

const getNFLPlayerProfileByExternalID = `-- name: GetNFLPlayerProfileByExternalID :one
SELECT npp.player_id, npp.position, npp.status, npp.college, npp.jersey_number, npp.experience, npp.birth_date, npp.height_cm, npp.weight_kg, npp.height_desc, npp.weight_desc FROM nfl_player_profiles npp
JOIN players p ON npp.player_id = p.id
WHERE p.sport_id = $1 AND p.external_id = $2
`

type GetNFLPlayerProfileByExternalIDParams struct {
	SportID    string `json:"sport_id"`
	ExternalID string `json:"external_id"`
}

func (q *Queries) GetNFLPlayerProfileByExternalID(ctx context.Context, arg GetNFLPlayerProfileByExternalIDParams) (NflPlayerProfile, error) {
	row := q.db.QueryRowContext(ctx, getNFLPlayerProfileByExternalID, arg.SportID, arg.ExternalID)
	var i NflPlayerProfile
	err := row.Scan(
		&i.PlayerID,
		&i.Position,
		&i.Status,
		&i.College,
		&i.JerseyNumber,
		&i.Experience,
		&i.BirthDate,
		&i.HeightCm,
		&i.WeightKg,
		&i.HeightDesc,
		&i.WeightDesc,
	)
	return i, err
}

const updateNFLPlayerProfile = `-- name: UpdateNFLPlayerProfile :one
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
RETURNING player_id, position, status, college, jersey_number, experience, birth_date, height_cm, weight_kg, height_desc, weight_desc
`

type UpdateNFLPlayerProfileParams struct {
	PlayerID     uuid.UUID      `json:"player_id"`
	Position     sql.NullString `json:"position"`
	Status       sql.NullString `json:"status"`
	College      sql.NullString `json:"college"`
	JerseyNumber sql.NullInt16  `json:"jersey_number"`
	Experience   sql.NullInt16  `json:"experience"`
	BirthDate    sql.NullTime   `json:"birth_date"`
	HeightCm     sql.NullInt32  `json:"height_cm"`
	WeightKg     sql.NullInt32  `json:"weight_kg"`
	HeightDesc   sql.NullString `json:"height_desc"`
	WeightDesc   sql.NullString `json:"weight_desc"`
}

func (q *Queries) UpdateNFLPlayerProfile(ctx context.Context, arg UpdateNFLPlayerProfileParams) (NflPlayerProfile, error) {
	row := q.db.QueryRowContext(ctx, updateNFLPlayerProfile,
		arg.PlayerID,
		arg.Position,
		arg.Status,
		arg.College,
		arg.JerseyNumber,
		arg.Experience,
		arg.BirthDate,
		arg.HeightCm,
		arg.WeightKg,
		arg.HeightDesc,
		arg.WeightDesc,
	)
	var i NflPlayerProfile
	err := row.Scan(
		&i.PlayerID,
		&i.Position,
		&i.Status,
		&i.College,
		&i.JerseyNumber,
		&i.Experience,
		&i.BirthDate,
		&i.HeightCm,
		&i.WeightKg,
		&i.HeightDesc,
		&i.WeightDesc,
	)
	return i, err
}
