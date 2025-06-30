package main

import (
	"database/sql"
	"github.com/mcdev12/dynasty/go/internal/draft"
	draftdb "github.com/mcdev12/dynasty/go/internal/draft/db"
	"github.com/mcdev12/dynasty/go/internal/draft/repository"
	"github.com/mcdev12/dynasty/go/internal/fantasyteam"
	fantasyteamdb "github.com/mcdev12/dynasty/go/internal/fantasyteam/db"
	"github.com/mcdev12/dynasty/go/internal/leagues"
	leaguedb "github.com/mcdev12/dynasty/go/internal/leagues/db"
	"github.com/mcdev12/dynasty/go/internal/player"
	playerdb "github.com/mcdev12/dynasty/go/internal/player/db"
	"github.com/mcdev12/dynasty/go/internal/roster"
	rosterdb "github.com/mcdev12/dynasty/go/internal/roster/db"
	"github.com/mcdev12/dynasty/go/internal/sports/base"
	"github.com/mcdev12/dynasty/go/internal/teams"
	teamsdb "github.com/mcdev12/dynasty/go/internal/teams/db"
	"github.com/mcdev12/dynasty/go/internal/users"
	usersdb "github.com/mcdev12/dynasty/go/internal/users/db"
)

type Services struct {
	Teams       *teams.Service
	Players     *player.Service
	Users       *users.Service
	League      *leagues.Service
	FantasyTeam *fantasyteam.Service
	Roster      *roster.Service
	Draft       *draft.Service
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

	// Roster players
	rosterQueries := rosterdb.New(database)
	rosterRepo := roster.NewRepository(rosterQueries)
	rosterApp := roster.NewApp(rosterRepo, fantasyTeamRepo, playerRepo)
	rosterService := roster.NewService(rosterApp)

	// Draft Service
	draftQueries := draftdb.New(database)
	draftRepo := repository.NewRepository(draftQueries)
	draftPickRepo := repository.NewDraftPickRepository(draftQueries, database)
	draftApp := draft.NewApp(draftRepo, draftPickRepo, leagueRepo)

	// 1) Create the RandomStrategy, injecting draftApp (which implements ListAvailable… & ClaimNext…)
	randStrat := draft.NewRandomStrategy(draftApp)
	draftOrchestrator := draft.NewOrchestrator(draftApp, randStrat, int32(100))
	draftService := draft.NewService(draftApp, draftOrchestrator)

	return &Services{
		Teams:       teamsService,
		Players:     playerService,
		Users:       userService,
		League:      leagueService,
		FantasyTeam: fantasyTeamService,
		Roster:      rosterService,
		Draft:       draftService,
	}
}
