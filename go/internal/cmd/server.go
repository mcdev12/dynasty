package main

import (
	"fmt"
	"log"
	"net/http"

	"connectrpc.com/grpcreflect"

	"github.com/mcdev12/dynasty/go/internal/genproto/fantasyteam/v1/fantasyteamv1connect"
	"github.com/mcdev12/dynasty/go/internal/genproto/league/v1/leaguev1connect"
	"github.com/mcdev12/dynasty/go/internal/genproto/player/v1/playerv1connect"
	"github.com/mcdev12/dynasty/go/internal/genproto/roster/v1/rosterv1connect"
	"github.com/mcdev12/dynasty/go/internal/genproto/team/v1/teamv1connect"
	"github.com/mcdev12/dynasty/go/internal/genproto/user/v1/userv1connect"
	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func setupServer(services *Services) *http.Server {
	mux := http.NewServeMux()

	// Setup CORS middleware
	c := cors.New(cors.Options{
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowedOrigins: []string{"*"},
		AllowedHeaders: []string{"*"},
	})

	// Register services
	registerServices(mux, services)

	// Setup reflection for grpcui/grpcurl
	setupReflection(mux)

	// Add health check endpoint
	setupHealthCheck(mux)

	// Wrap with CORS
	handler := c.Handler(mux)

	// Setup HTTP/2 server
	return &http.Server{
		Addr:    fmt.Sprintf(":%s", getEnv("PORT", "8080")),
		Handler: h2c.NewHandler(handler, &http2.Server{}),
	}
}

func registerServices(mux *http.ServeMux, services *Services) {
	// Register team service
	teamServicePath, teamServiceHandler := teamv1connect.NewTeamServiceHandler(services.Teams)
	mux.Handle(teamServicePath, teamServiceHandler)

	// Register player service
	playerServicePath, playerServiceHandler := playerv1connect.NewPlayerServiceHandler(services.Players)
	mux.Handle(playerServicePath, playerServiceHandler)

	// Register user service
	userServicePath, userServiceHandler := userv1connect.NewUserServiceHandler(services.Users)
	mux.Handle(userServicePath, userServiceHandler)

	// Register league service
	leagueServicePath, leagueServiceHandler := leaguev1connect.NewLeagueServiceHandler(services.League)
	mux.Handle(leagueServicePath, leagueServiceHandler)

	// Register fantasy team service
	fantasyTeamServicePath, fantasyTeamServiceHandler := fantasyteamv1connect.NewFantasyTeamServiceHandler(services.FantasyTeam)
	mux.Handle(fantasyTeamServicePath, fantasyTeamServiceHandler)

	// Register roster service
	rosterServicePath, rosterServiceHandler := rosterv1connect.NewRosterServiceHandler(services.Roster)
	mux.Handle(rosterServicePath, rosterServiceHandler)
}

func setupReflection(mux *http.ServeMux) {
	reflector := grpcreflect.NewStaticReflector(
		teamv1connect.TeamServiceName,
		playerv1connect.PlayerServiceName,
		userv1connect.UserServiceName,
		leaguev1connect.LeagueServiceName,
		fantasyteamv1connect.FantasyTeamServiceName,
		rosterv1connect.RosterServiceName,
	)
	mux.Handle(grpcreflect.NewHandlerV1(reflector))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))
}

func setupHealthCheck(mux *http.ServeMux) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			log.Printf("Failed to write health check response: %v", err)
		}
	})
}
