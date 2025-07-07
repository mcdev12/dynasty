package main

import (
	"database/sql"

	draftdraft "github.com/mcdev12/dynasty/go/internal/draft/draft"
	draftdb "github.com/mcdev12/dynasty/go/internal/draft/draft/db"
	"github.com/mcdev12/dynasty/go/internal/draft/outbox"
	outboxdb "github.com/mcdev12/dynasty/go/internal/draft/outbox/db"
	"github.com/mcdev12/dynasty/go/internal/draft/pick"
	pickdb "github.com/mcdev12/dynasty/go/internal/draft/pick/db"
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
	Teams             *teams.Service
	Players           *player.Service
	Users             *users.Service
	League            *leagues.Service
	FantasyTeam       *fantasyteam.Service
	Roster            *roster.Service
	DraftService      *draftdraft.Service
	DraftPickService  *pick.Service
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
	playerApp := player.NewApp(playerRepo, plugins)
	playerService := player.NewService(playerApp, teamsService)

	// Users
	userQueries := usersdb.New(database)
	userRepo := users.NewRepository(userQueries)
	userApp := users.NewApp(userRepo)
	userService := users.NewService(userApp)

	// League
	leagueQueries := leaguedb.New(database)
	leagueRepo := leagues.NewRepository(leagueQueries)
	leagueApp := leagues.NewApp(leagueRepo)
	leagueService := leagues.NewService(leagueApp, userService)

	// FantasyTeam
	fantasyTeamQueries := fantasyteamdb.New(database)
	fantasyTeamRepo := fantasyteam.NewRepository(fantasyTeamQueries)
	fantasyTeamApp := fantasyteam.NewApp(fantasyTeamRepo)
	fantasyTeamService := fantasyteam.NewService(fantasyTeamApp, userService, leagueService)

	// Roster players
	rosterQueries := rosterdb.New(database)
	rosterRepo := roster.NewRepository(rosterQueries)
	rosterApp := roster.NewApp(rosterRepo)
	rosterService := roster.NewService(rosterApp, fantasyTeamService, playerService)

	// Draft Services Setup (simplified for monolith - avoiding circular dependencies for now)
	draftQueries := draftdb.New(database)
	pickQueries := pickdb.New(database)
	outboxQueries := outboxdb.New(database)

	// Draft app and service
	draftRepo := draftdraft.NewRepository(draftQueries)
	draftApp := draftdraft.NewApp(draftRepo)

	// Outbox app
	outboxRepo := outbox.NewRepository(outboxQueries)
	outboxApp := outbox.NewApp(outboxRepo)

	// Create draft service with outbox app and league service
	draftService := draftdraft.NewService(draftApp, outboxApp, leagueService)

	// Draft pick app and service
	draftPickRepo := pick.NewRepository(pickQueries, database)
	pickApp := pick.NewApp(draftPickRepo)
	pickService := pick.NewService(pickApp, draftService, outboxApp)

	// NOTE: Orchestrator is now a separate binary - see go/internal/draft/orchestrator/cmd/main.go
	// It runs independently and subscribes to domain events via the message bus

	return &Services{
		Teams:            teamsService,
		Players:          playerService,
		Users:            userService,
		League:           leagueService,
		FantasyTeam:      fantasyTeamService,
		Roster:           rosterService,
		DraftService:     draftService,
		DraftPickService: pickService,
	}
}
