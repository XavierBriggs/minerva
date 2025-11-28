package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/fortuna/minerva/internal/service"
	"github.com/fortuna/minerva/internal/store"
	"github.com/fortuna/minerva/internal/store/repository"
	"github.com/gorilla/mux"
)

// Handler contains dependencies for HTTP handlers
type Handler struct {
	db               *store.Database
	gameService      *service.GameService
	playerService    *service.PlayerService
	statsService     *service.StatsService
	analyticsService *service.AnalyticsService
}

// NewHandler creates a new handler
func NewHandler(db *store.Database) *Handler {
	return &Handler{
		db:               db,
		gameService:      service.NewGameService(db),
		playerService:    service.NewPlayerService(db),
		statsService:     service.NewStatsService(db),
		analyticsService: service.NewAnalyticsService(db),
	}
}

// HealthCheck handles health check requests
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "minerva",
		"version": "2.0.0",
	})
}

// GetLiveGames returns all currently live games
func (h *Handler) GetLiveGames(w http.ResponseWriter, r *http.Request) {
	games, err := h.gameService.GetLiveGames(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch live games", err)
		return
	}

	respondJSON(w, http.StatusOK, games)
}

// CleanupStaleGames marks old "in_progress" games as "final"
func (h *Handler) CleanupStaleGames(w http.ResponseWriter, r *http.Request) {
	count, err := h.gameService.CleanupStaleGames(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to cleanup stale games", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":       "Stale games cleaned up",
		"games_updated": count,
	})
}

// GetGamesByDate returns all games on a specific date
func (h *Handler) GetGamesByDate(w http.ResponseWriter, r *http.Request) {
	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid date format (use YYYY-MM-DD)", err)
		return
	}

	games, err := h.gameService.GetGamesByDate(r.Context(), date)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch games", err)
		return
	}

	respondJSON(w, http.StatusOK, games)
}

// GetUpcomingGames returns upcoming scheduled games
func (h *Handler) GetUpcomingGames(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 10 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	games, err := h.gameService.GetUpcomingGames(r.Context(), limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch upcoming games", err)
		return
	}

	respondJSON(w, http.StatusOK, games)
}

// GetTodaysGames returns all games for today (live, scheduled, final)
func (h *Handler) GetTodaysGames(w http.ResponseWriter, r *http.Request) {
	games, err := h.gameService.GetTodaysGames(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch today's games", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"games": games,
		"count": len(games),
	})
}

// GetGame returns a specific game by ID
func (h *Handler) GetGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameID"]

	game, err := h.gameService.GetGame(r.Context(), gameID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Game not found", err)
		return
	}

	respondJSON(w, http.StatusOK, game)
}

// GetGameBoxScore returns the box score for a game
func (h *Handler) GetGameBoxScore(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameID"]

	boxScore, err := h.statsService.GetGameBoxScore(r.Context(), gameID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Box score not found", err)
		return
	}

	respondJSON(w, http.StatusOK, boxScore)
}

// GetPlayer returns a player by ID
func (h *Handler) GetPlayer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	playerIDStr := vars["playerID"]

	playerID, err := strconv.Atoi(playerIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid player ID", err)
		return
	}

	player, err := h.playerService.GetPlayer(r.Context(), playerID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Player not found", err)
		return
	}

	respondJSON(w, http.StatusOK, player)
}

// SearchPlayers searches for players by name
func (h *Handler) SearchPlayers(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		respondError(w, http.StatusBadRequest, "Missing query parameter 'q'", nil)
		return
	}

	profiles, err := h.playerService.SearchPlayers(r.Context(), query)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to search players", err)
		return
	}

	// Extract just the player data for the response
	players := make([]*store.Player, 0, len(profiles))
	for _, profile := range profiles {
		if profile.Player != nil {
			// Add current_team_id if available
			if profile.Team != nil {
				profile.Player.CurrentTeamID = profile.Team.TeamID
			}
			players = append(players, profile.Player)
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"players": players})
}

// GetPlayerStats returns a player's recent game stats
func (h *Handler) GetPlayerStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	playerIDStr := vars["playerID"]

	playerID, err := strconv.Atoi(playerIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid player ID", err)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 10 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}

	stats, err := h.playerService.GetPlayerStats(r.Context(), playerID, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch player stats", err)
		return
	}

	respondJSON(w, http.StatusOK, stats)
}

// GetPlayerSeasonAverages returns a player's season averages
func (h *Handler) GetPlayerSeasonAverages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	playerIDStr := vars["playerID"]

	playerID, err := strconv.Atoi(playerIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid player ID", err)
		return
	}

	seasonID := r.URL.Query().Get("season")
	if seasonID == "" {
		seasonID = "2024-25" // default to current season
	}

	averages, err := h.playerService.GetPlayerSeasonAverages(r.Context(), playerID, seasonID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to calculate season averages", err)
		return
	}

	respondJSON(w, http.StatusOK, averages)
}

// GetTeams returns all teams
func (h *Handler) GetTeams(w http.ResponseWriter, r *http.Request) {
	teamRepo := repository.NewTeamRepository(h.db)
	teams, err := teamRepo.GetAll(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch teams", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"teams": teams})
}

// GetTeam returns a specific team by ID
func (h *Handler) GetTeam(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamIDStr := vars["teamID"]

	teamID, err := strconv.Atoi(teamIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid team ID", err)
		return
	}

	teamRepo := repository.NewTeamRepository(h.db)
	team, err := teamRepo.GetByID(r.Context(), teamID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch team", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"team": team})
}

// GetTeamRoster returns a team's current roster
func (h *Handler) GetTeamRoster(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamIDStr := vars["teamID"]

	teamID, err := strconv.Atoi(teamIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid team ID", err)
		return
	}

	roster, err := h.playerService.GetTeamRoster(r.Context(), teamID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch team roster", err)
		return
	}

	respondJSON(w, http.StatusOK, roster)
}

// GetTeamSchedule returns a team's schedule
func (h *Handler) GetTeamSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamIDStr := vars["teamID"]

	teamID, err := strconv.Atoi(teamIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid team ID", err)
		return
	}

	seasonYear := r.URL.Query().Get("season")
	if seasonYear == "" {
		seasonYear = "2025-26" // default to current season
	}

	// Lookup season_id from season_year
	seasonID, err := h.lookupSeasonID(r.Context(), seasonYear)
	if err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid season: %s", seasonYear), err)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 20 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	schedule, err := h.gameService.GetTeamSchedule(r.Context(), teamID, seasonID, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch team schedule", err)
		return
	}

	respondJSON(w, http.StatusOK, schedule)
}

// GetPlayerPerformanceTrend returns performance trends for a player
func (h *Handler) GetPlayerPerformanceTrend(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	playerIDStr := vars["playerID"]

	playerID, err := strconv.Atoi(playerIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid player ID", err)
		return
	}

	gamesStr := r.URL.Query().Get("games")
	games := 10 // default
	if gamesStr != "" {
		if g, err := strconv.Atoi(gamesStr); err == nil && g > 0 && g <= 50 {
			games = g
		}
	}

	trend, err := h.analyticsService.GetPlayerPerformanceTrend(r.Context(), playerID, games)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to calculate performance trend", err)
		return
	}

	respondJSON(w, http.StatusOK, trend)
}

// GetPlayerMLFeatures returns ML features for a player
func (h *Handler) GetPlayerMLFeatures(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	playerIDStr := vars["playerID"]

	playerID, err := strconv.Atoi(playerIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid player ID", err)
		return
	}

	seasonID := r.URL.Query().Get("season")
	if seasonID == "" {
		seasonID = "2024-25" // default to current season
	}

	features, err := h.analyticsService.GetPlayerMLFeatures(r.Context(), playerID, seasonID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate ML features", err)
		return
	}

	respondJSON(w, http.StatusOK, features)
}

// lookupSeasonID queries the database to get season_id (INT) from season_year (STRING)
func (h *Handler) lookupSeasonID(ctx context.Context, seasonYear string) (int, error) {
	query := `SELECT season_id FROM seasons WHERE season_year = $1 AND sport = 'basketball_nba' LIMIT 1`
	
	var seasonID int
	err := h.db.DB().QueryRowContext(ctx, query, seasonYear).Scan(&seasonID)
	if err != nil {
		return 0, fmt.Errorf("season '%s' not found in database: %w", seasonYear, err)
	}
	
	return seasonID, nil
}

// respondJSON writes a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError writes an error response
func respondError(w http.ResponseWriter, status int, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	
	response := map[string]interface{}{
		"error":   message,
		"status":  status,
	}
	
	if err != nil {
		response["details"] = err.Error()
	}
	
	json.NewEncoder(w).Encode(response)
}

