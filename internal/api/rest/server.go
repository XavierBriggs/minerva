package rest

import (
	"context"
	"fmt"
	"net/http"

	"github.com/fortuna/minerva/internal/backfill"
	"github.com/fortuna/minerva/internal/store"
	"github.com/gorilla/mux"
)

// Server represents the REST API server
type Server struct {
	port    string
	server  *http.Server
	handler *Handler
}

// NewServer creates a new REST API server
func NewServer(port string, db *store.Database, backfillSvc *backfill.Service) *Server {
	handler := NewHandler(db)
	backfillHandler := NewBackfillHandler(backfillSvc)

	router := mux.NewRouter()

	// Apply middleware
	router.Use(RecoveryMiddleware)
	router.Use(LoggingMiddleware)
	router.Use(CORSMiddleware)

	// Health check
	router.HandleFunc("/health", handler.HealthCheck).Methods("GET")

	// API v1 routes
	api := router.PathPrefix("/api/v1").Subrouter()

	// Games
	api.HandleFunc("/games/live", handler.GetLiveGames).Methods("GET")
	api.HandleFunc("/games/today", handler.GetTodaysGames).Methods("GET")
	api.HandleFunc("/games/upcoming", handler.GetUpcomingGames).Methods("GET")
	api.HandleFunc("/games/cleanup", handler.CleanupStaleGames).Methods("POST")
	api.HandleFunc("/games", handler.GetGamesByDate).Methods("GET")
	api.HandleFunc("/games/{gameID}", handler.GetGame).Methods("GET")
	api.HandleFunc("/games/{gameID}/boxscore", handler.GetGameBoxScore).Methods("GET")

	// Players
	api.HandleFunc("/players/search", handler.SearchPlayers).Methods("GET")
	api.HandleFunc("/players/{playerID}", handler.GetPlayer).Methods("GET")
	api.HandleFunc("/players/{playerID}/stats", handler.GetPlayerStats).Methods("GET")
	api.HandleFunc("/players/{playerID}/averages", handler.GetPlayerSeasonAverages).Methods("GET")
	api.HandleFunc("/players/{playerID}/trend", handler.GetPlayerPerformanceTrend).Methods("GET")
	api.HandleFunc("/players/{playerID}/ml-features", handler.GetPlayerMLFeatures).Methods("GET")

	// Teams
	api.HandleFunc("/teams", handler.GetTeams).Methods("GET")
	api.HandleFunc("/teams/{teamID}", handler.GetTeam).Methods("GET")
	api.HandleFunc("/teams/{teamID}/roster", handler.GetTeamRoster).Methods("GET")
	api.HandleFunc("/teams/{teamID}/schedule", handler.GetTeamSchedule).Methods("GET")

	// Backfill operations
	api.HandleFunc("/backfill", backfillHandler.HandleBackfillRequest).Methods("POST")
	api.HandleFunc("/backfill/status", backfillHandler.HandleBackfillStatus).Methods("GET")

	return &Server{
		port:    port,
		handler: handler,
		server: &http.Server{
			Addr:    fmt.Sprintf(":%s", port),
			Handler: router,
		},
	}
}

// Start starts the REST API server
func (s *Server) Start() error {
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}
