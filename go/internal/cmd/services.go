package main

import (
	"database/sql"
	"github.com/mcdev12/dynasty/go/internal/fantasyteam"
	"github.com/mcdev12/dynasty/go/internal/leagues"
	"github.com/mcdev12/dynasty/go/internal/users"

	fantasyteamdb "github.com/mcdev12/dynasty/go/internal/fantasyteam/db"
	leaguedb "github.com/mcdev12/dynasty/go/internal/leagues/db"
	"github.com/mcdev12/dynasty/go/internal/player"
	playerdb "github.com/mcdev12/dynasty/go/internal/player/db"
	"github.com/mcdev12/dynasty/go/internal/sports/base"
	"github.com/mcdev12/dynasty/go/internal/teams"
	teamsdb "github.com/mcdev12/dynasty/go/internal/teams/db"
	usersdb "github.com/mcdev12/dynasty/go/internal/users/db"
)

type Services struct {
	Teams       *teams.Service
	Players     *player.Service
	Users       *users.Service
	League      *leagues.Service
	FantasyTeam *fantasyteam.Service
}

func setupServices(database *sql.DB, plugins map[string]base.SportPlugin) *Services {
	// Wire up dependency injection chain
	// Database layer → Repository layer → App layer → Service layer

	// Teams
	queries := teamsdb.New(database)
	teamsRepo := teams.NewRepository(queries)
	teamsApp := teams.NewApp(teamsRepo, plugins)
	teamsService := teams.NewService(teamsApp)

	// Players
	playerQueries := playerdb.New(database)
	playerRepo := player.NewRepository(playerQueries, database)
	playerApp := player.NewApp(playerRepo, plugins, teamsApp)
	playerService := player.NewService(playerApp)

	// Users
	userQueries := usersdb.New(database)
	userRepo := users.NewRepository(userQueries)
	userApp := users.NewApp(userRepo)
	userService := users.NewService(userApp)

	// League
	leagueQueries := leaguedb.New(database)
	leagueRepo := leagues.NewRepository(leagueQueries)
	leagueApp := leagues.NewApp(leagueRepo, userApp)
	leagueService := leagues.NewService(leagueApp)

	// FantasyTeam
	fantasyTeamQueries := fantasyteamdb.New(database)
	fantasyTeamRepo := fantasyteam.NewRepository(fantasyTeamQueries)
	fantasyTeamApp := fantasyteam.NewApp(fantasyTeamRepo, userApp, leagueApp)
	fantasyTeamService := fantasyteam.NewService(fantasyTeamApp)

	return &Services{
		Teams:       teamsService,
		Players:     playerService,
		Users:       userService,
		League:      leagueService,
		FantasyTeam: fantasyTeamService,
	}
}
