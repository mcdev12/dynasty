package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mcdev12/dynasty/go/internal/dbconfig"
)

// Team mirrors your JSON structure
type Team struct {
	ID              string `json:"id"`
	SportID         string `json:"sport_id"`
	ExternalID      string `json:"external_id"`
	Name            string `json:"name"`
	Code            string `json:"code"`
	City            string `json:"city"`
	Coach           string `json:"coach"`
	Owner           string `json:"owner"`
	Stadium         string `json:"stadium"`
	EstablishedYear int    `json:"established_year"`
	CreatedAt       string `json:"created_at"`
}

func main() {
	// 1) Load the JSON snapshot
	data, err := os.ReadFile("go/internal/assets/teams.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "read JSON: %v\n", err)
		os.Exit(1)
	}
	var teams []Team
	if err := json.Unmarshal(data, &teams); err != nil {
		fmt.Fprintf(os.Stderr, "unmarshal JSON: %v\n", err)
		os.Exit(1)
	}

	// 2) Connect using shared dbconfig
	cfg := dbconfig.NewConfigFromEnv()
	pool, err := pgxpool.New(context.Background(), cfg.DSN())
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	// 3) Upsert and count
	var (
		total    = len(teams)
		inserted int
		skipped  int
		errs     int
	)

	for _, t := range teams {
		cmdTag, err := pool.Exec(context.Background(), `
            INSERT INTO teams (
              id, sport_id, external_id, name, code, city,
              coach, owner, stadium, established_year, created_at
            ) VALUES (
              $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11
            )
            ON CONFLICT (sport_id, external_id) DO NOTHING
        `,
			t.ID, t.SportID, t.ExternalID, t.Name, t.Code, t.City,
			t.Coach, t.Owner, t.Stadium, t.EstablishedYear, t.CreatedAt,
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error inserting team %s: %v\n", t.ID, err)
			errs++
			continue
		}
		if cmdTag.RowsAffected() == 1 {
			inserted++
		} else {
			skipped++
		}
	}

	// 4) Print summary
	fmt.Printf(
		"Teams seed complete: %d total, %d inserted, %d skipped, %d errors\n",
		total, inserted, skipped, errs,
	)
}
