package main

import (
	"database/sql"

	"github.com/mcdev12/dynasty/go/internal/sports/base"
	"github.com/mcdev12/dynasty/go/internal/teams"
	"github.com/mcdev12/dynasty/go/internal/teams/db"
)

type Services struct {
	Teams *teams.Service
}

func setupServices(database *sql.DB, plugins map[string]base.SportPlugin) *Services {
	// Wire up dependency injection chain
	// Database layer → Repository layer → App layer → Service layer

	// Teams
	queries := db.New(database)
	teamsRepo := teams.NewRepository(queries)
	teamsApp := teams.NewApp(teamsRepo, plugins)
	teamsService := teams.NewService(teamsApp)

	return &Services{
		Teams: teamsService,
	}
}
