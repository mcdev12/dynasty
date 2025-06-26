package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mcdev12/dynasty/go/internal/dbconfig"
)

// Player uses a string for CreatedAt to match your JSON layout
type Player struct {
	ID         uuid.UUID  `json:"id"`
	SportID    string     `json:"sport_id"`
	ExternalID string     `json:"external_id"`
	FullName   string     `json:"full_name"`
	TeamID     *uuid.UUID `json:"team_id"`
	CreatedAt  string     `json:"created_at"`
}

// NFLPlayerProfile uses a string pointer for BirthDate
type NFLPlayerProfile struct {
	PlayerID     uuid.UUID `json:"player_id"`
	Position     string    `json:"position"`
	Status       string    `json:"status"`
	College      string    `json:"college"`
	JerseyNumber int       `json:"jersey_number"`
	Experience   int       `json:"experience"`
	BirthDate    *string   `json:"birth_date"`
	HeightCm     int       `json:"height_cm"`
	WeightKg     int       `json:"weight_kg"`
	HeightDesc   string    `json:"height_desc"`
	WeightDesc   string    `json:"weight_desc"`
}

func main() {
	ctx := context.Background()

	// 1) Load players.json
	pData, err := os.ReadFile("go/internal/assets/players.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "read players.json: %v\n", err)
		os.Exit(1)
	}
	var players []Player
	if err := json.Unmarshal(pData, &players); err != nil {
		fmt.Fprintf(os.Stderr, "unmarshal players: %v\n", err)
		os.Exit(1)
	}

	// 2) Load player_profiles.json
	profData, err := os.ReadFile("go/internal/assets/nfl_player_profiles.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "read player_profiles.json: %v\n", err)
		os.Exit(1)
	}
	var profiles []NFLPlayerProfile
	if err := json.Unmarshal(profData, &profiles); err != nil {
		fmt.Fprintf(os.Stderr, "unmarshal profiles: %v\n", err)
		os.Exit(1)
	}

	// 3) Connect to DB
	cfg := dbconfig.NewConfigFromEnv()
	pool, err := pgxpool.New(ctx, cfg.DSN())
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect error: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	// 4) Seed players
	total, inserted, skipped, errs := len(players), 0, 0, 0
	for _, p := range players {
		tag, err := pool.Exec(ctx, `
            INSERT INTO players (
              id, sport_id, external_id, full_name, team_id, created_at
            ) VALUES ($1,$2,$3,$4,$5,$6)
            ON CONFLICT (sport_id, external_id) DO NOTHING
        `, p.ID, p.SportID, p.ExternalID, p.FullName, p.TeamID, p.CreatedAt)
		if err != nil {
			errs++
			continue
		}
		if tag.RowsAffected() == 1 {
			inserted++
		} else {
			skipped++
		}
	}
	fmt.Printf(
		"Players seed: total=%d inserted=%d skipped=%d errors=%d\n",
		total, inserted, skipped, errs,
	)

	// 5) Seed profiles
	total, inserted, skipped, errs = len(profiles), 0, 0, 0
	for _, prof := range profiles {
		tag, err := pool.Exec(ctx, `
            INSERT INTO nfl_player_profiles (
              player_id, position, status, college,
              jersey_number, experience, birth_date,
              height_cm, weight_kg, height_desc, weight_desc
            ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
            ON CONFLICT (player_id) DO NOTHING
        `,
			prof.PlayerID, prof.Position, prof.Status, prof.College,
			prof.JerseyNumber, prof.Experience, prof.BirthDate,
			prof.HeightCm, prof.WeightKg, prof.HeightDesc, prof.WeightDesc,
		)
		if err != nil {
			errs++
			continue
		}
		if tag.RowsAffected() == 1 {
			inserted++
		} else {
			skipped++
		}
	}
	fmt.Printf(
		"NFL profiles seed: total=%d inserted=%d skipped=%d errors=%d\n",
		total, inserted, skipped, errs,
	)
}
