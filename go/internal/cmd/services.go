package main

import (
	"database/sql"

	"github.com/mcdev12/dynasty/go/internal/player"
	playerdb "github.com/mcdev12/dynasty/go/internal/player/db"
	"github.com/mcdev12/dynasty/go/internal/sports/base"
	"github.com/mcdev12/dynasty/go/internal/teams"
	"github.com/mcdev12/dynasty/go/internal/teams/db"
)

type Services struct {
	Teams   *teams.Service
	Players *player.Service
}

func setupServices(database *sql.DB, plugins map[string]base.SportPlugin) *Services {
	// Wire up dependency injection chain
	// Database layer → Repository layer → App layer → Service layer

	// Teams
	queries := db.New(database)
	teamsRepo := teams.NewRepository(queries)
	teamsApp := teams.NewApp(teamsRepo, plugins)
	teamsService := teams.NewService(teamsApp)

	// Players
	playerQueries := playerdb.New(database)
	playerRepo := player.NewRepository(playerQueries, database)
	playerApp := player.NewApp(playerRepo, plugins)
	playerService := player.NewService(playerApp)

	return &Services{
		Teams:   teamsService,
		Players: playerService,
	}
}
