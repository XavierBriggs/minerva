package reconciliation

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/fortuna/minerva/internal/ingest/espn"
	"github.com/fortuna/minerva/internal/ingest/google"
	"github.com/fortuna/minerva/internal/store"
)

// Engine reconciles data from multiple sources (ESPN + Google)
type Engine struct {
	strategy ReconciliationStrategy
	metrics  *Metrics
}

// ReconciliationStrategy defines how to merge conflicting data
type ReconciliationStrategy string

const (
	// PreferLatest prioritizes the most recent data
	PreferLatest ReconciliationStrategy = "prefer_latest"
	
	// PreferAuthoritative prioritizes ESPN over Google
	PreferAuthoritative ReconciliationStrategy = "prefer_authoritative"
	
	// SmartMerge uses context-aware logic (default)
	SmartMerge ReconciliationStrategy = "smart_merge"
)

// Metrics tracks reconciliation statistics
type Metrics struct {
	TotalReconciliations int
	Conflicts            int
	GooglePreferred      int
	ESPNPreferred        int
	LastReconciliation   time.Time
}

// NewEngine creates a new reconciliation engine
func NewEngine(strategy ReconciliationStrategy) *Engine {
	if strategy == "" {
		strategy = SmartMerge
	}
	
	return &Engine{
		strategy: strategy,
		metrics: &Metrics{
			LastReconciliation: time.Now(),
		},
	}
}

// ReconcileGame merges game data from ESPN and Google sources
// ESPN is the authoritative fallback when Google is unavailable
func (e *Engine) ReconcileGame(espnGame *store.Game, googleGame *google.LiveGame) (*store.Game, error) {
	e.metrics.TotalReconciliations++
	e.metrics.LastReconciliation = time.Now()
	
	if espnGame == nil && googleGame == nil {
		return nil, fmt.Errorf("both sources are nil")
	}
	
	// ESPN is the fallback - use it when Google is unavailable
	if googleGame == nil {
		log.Println("  Using ESPN data (Google unavailable - fallback to authoritative source)")
		e.metrics.ESPNPreferred++
		return espnGame, nil
	}
	
	// Google available but ESPN missing (rare case - new game not in ESPN yet)
	if espnGame == nil {
		log.Println("  ⚠️  Using Google data only (ESPN unavailable - unusual)")
		e.metrics.GooglePreferred++
		return google.ConvertToStoreGame(*googleGame, 1), nil
	}
	
	// Both sources available - apply strategy
	switch e.strategy {
	case PreferLatest:
		return e.reconcilePreferLatest(espnGame, googleGame)
	case PreferAuthoritative:
		return e.reconcilePreferAuthoritative(espnGame, googleGame)
	case SmartMerge:
		return e.reconcileSmartMerge(espnGame, googleGame)
	default:
		return e.reconcileSmartMerge(espnGame, googleGame)
	}
}

// reconcilePreferLatest always uses Google (more recent)
func (e *Engine) reconcilePreferLatest(espnGame *store.Game, googleGame *google.LiveGame) (*store.Game, error) {
	e.metrics.GooglePreferred++
	log.Println("  Strategy: Prefer Latest (Google)")
	
	merged := google.ConvertToStoreGame(*googleGame, espnGame.SeasonID)
	merged.GameID = espnGame.GameID  // Keep ESPN game ID
	merged.HomeTeamID = espnGame.HomeTeamID
	merged.AwayTeamID = espnGame.AwayTeamID
	
	return merged, nil
}

// reconcilePreferAuthoritative always uses ESPN (more accurate)
func (e *Engine) reconcilePreferAuthoritative(espnGame *store.Game, googleGame *google.LiveGame) (*store.Game, error) {
	e.metrics.ESPNPreferred++
	log.Println("  Strategy: Prefer Authoritative (ESPN)")
	return espnGame, nil
}

// reconcileSmartMerge uses context-aware logic
// ESPN is always the authoritative fallback
func (e *Engine) reconcileSmartMerge(espnGame *store.Game, googleGame *google.LiveGame) (*store.Game, error) {
	merged := &store.Game{}
	
	// Always use ESPN for structural data (IDs, teams, season)
	// ESPN is the authoritative source for game identity
	merged.GameID = espnGame.GameID
	merged.SeasonID = espnGame.SeasonID
	merged.HomeTeamID = espnGame.HomeTeamID
	merged.AwayTeamID = espnGame.AwayTeamID
	merged.GameDate = espnGame.GameDate
	merged.Venue = espnGame.Venue
	merged.Attendance = espnGame.Attendance
	
	// Game state determines which source to trust for live data
	gameState := determineGameState(espnGame, googleGame)
	
	switch gameState {
	case StatePreGame:
		// Pre-game: ESPN is authoritative (fallback: always ESPN)
		e.metrics.ESPNPreferred++
		log.Println("  Strategy: Smart Merge → Pre-game (ESPN - authoritative)")
		return espnGame, nil
		
	case StateLive:
		// Live game: Use Google for scores/time (fresher), ESPN for structure
		// If Google fails/missing, ESPN is the fallback
		e.metrics.GooglePreferred++
		log.Println("  Strategy: Smart Merge → Live (Google scores + ESPN structure, ESPN fallback)")
		
		merged.Status = "in_progress"
		
		// Use Google scores if available, otherwise fall back to ESPN
		if googleGame.HomeScore > 0 || googleGame.AwayScore > 0 {
			merged.HomeScore = sql.NullInt32{Int32: int32(googleGame.HomeScore), Valid: true}
			merged.AwayScore = sql.NullInt32{Int32: int32(googleGame.AwayScore), Valid: true}
		} else if espnGame.HomeScore.Valid || espnGame.AwayScore.Valid {
			merged.HomeScore = espnGame.HomeScore
			merged.AwayScore = espnGame.AwayScore
		}
		
		// Use Google period if available, otherwise fall back to ESPN
		if googleGame.Period > 0 {
			merged.Period = sql.NullInt32{Int32: int32(googleGame.Period), Valid: true}
		} else if espnGame.Period.Valid {
			merged.Period = espnGame.Period
		}
		
		// Use Google time if available, otherwise fall back to ESPN
		if googleGame.TimeRemaining != "" {
			merged.Clock = sql.NullString{String: googleGame.TimeRemaining, Valid: true}
		} else if espnGame.Clock.Valid {
			merged.Clock = espnGame.Clock
		}
		
		merged.GameTime = espnGame.GameTime
		
		return merged, nil
		
	case StateFinal:
		// Final: ESPN is authoritative for stats (fallback: always ESPN)
		e.metrics.ESPNPreferred++
		log.Println("  Strategy: Smart Merge → Final (ESPN - authoritative)")
		return espnGame, nil
		
	case StateConflict:
		// Conflict detected - always fall back to ESPN (authoritative)
		e.metrics.Conflicts++
		e.metrics.ESPNPreferred++
		log.Printf("  ⚠️  Conflict detected between sources (fallback to ESPN - authoritative)")
		return espnGame, nil
	}
	
	return merged, nil
}

// GameState represents the current state of a game
type GameState string

const (
	StatePreGame  GameState = "pre_game"
	StateLive     GameState = "live"
	StateFinal    GameState = "final"
	StateConflict GameState = "conflict"
)

// determineGameState analyzes both sources to determine game state
func determineGameState(espnGame *store.Game, googleGame *google.LiveGame) GameState {
	// Check for obvious conflicts
	if hasConflict(espnGame, googleGame) {
		return StateConflict
	}
	
	// Final game (both agree it's over)
	if espnGame.Status == "final" || espnGame.Status == "STATUS_FINAL" {
		return StateFinal
	}
	if !googleGame.IsLive && (googleGame.GameStatus == "Final" || googleGame.GameStatus == "final") {
		return StateFinal
	}
	
	// Live game (either source indicates live)
	if espnGame.Status == "in_progress" || espnGame.Status == "STATUS_IN_PROGRESS" {
		return StateLive
	}
	if googleGame.IsLive {
		return StateLive
	}
	
	// Pre-game (scheduled but not started)
	if espnGame.Status == "scheduled" || espnGame.Status == "STATUS_SCHEDULED" {
		return StatePreGame
	}
	
	// Default to pre-game if unclear
	return StatePreGame
}

// hasConflict detects obvious data inconsistencies
func hasConflict(espnGame *store.Game, googleGame *google.LiveGame) bool {
	// Check for major score discrepancies (> 20 points)
	if espnGame.HomeScore.Valid && googleGame.HomeScore > 0 {
		scoreDiff := abs(int(espnGame.HomeScore.Int32) - googleGame.HomeScore)
		if scoreDiff > 20 {
			log.Printf("  ⚠️  Large score discrepancy detected: ESPN=%d, Google=%d",
				espnGame.HomeScore.Int32, googleGame.HomeScore)
			return true
		}
	}
	
	if espnGame.AwayScore.Valid && googleGame.AwayScore > 0 {
		scoreDiff := abs(int(espnGame.AwayScore.Int32) - googleGame.AwayScore)
		if scoreDiff > 20 {
			log.Printf("  ⚠️  Large score discrepancy detected: ESPN=%d, Google=%d",
				espnGame.AwayScore.Int32, googleGame.AwayScore)
			return true
		}
	}
	
	// Check for status conflicts (one says final, other says live)
	espnFinal := espnGame.Status == "final" || espnGame.Status == "STATUS_FINAL"
	googleFinal := !googleGame.IsLive && (googleGame.GameStatus == "Final" || googleGame.GameStatus == "final")
	
	if espnFinal != googleFinal && googleGame.IsLive {
		log.Printf("  ⚠️  Status conflict: ESPN=%s, Google=%s (live=%v)",
			espnGame.Status, googleGame.GameStatus, googleGame.IsLive)
		return true
	}
	
	return false
}

// abs returns absolute value of an int
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// GetMetrics returns current reconciliation metrics
func (e *Engine) GetMetrics() *Metrics {
	return e.metrics
}

// ResetMetrics clears all metrics
func (e *Engine) ResetMetrics() {
	e.metrics = &Metrics{
		LastReconciliation: time.Now(),
	}
}

// ReconcileGames reconciles a list of games from both sources
func (e *Engine) ReconcileGames(espnGames []*store.Game, googleGames []google.LiveGame) ([]*store.Game, error) {
	// Create a map of Google games by team matchup for quick lookup
	googleMap := make(map[string]*google.LiveGame)
	for i := range googleGames {
		game := &googleGames[i]
		key := createMatchupKey(game.HomeTeam, game.AwayTeam)
		googleMap[key] = game
	}
	
	var reconciledGames []*store.Game
	
	for _, espnGame := range espnGames {
		// Try to find matching Google game
		// Note: This requires team name matching logic
		var googleGame *google.LiveGame
		// TODO: Implement team name lookup to find matching Google game
		
		reconciled, err := e.ReconcileGame(espnGame, googleGame)
		if err != nil {
			log.Printf("Error reconciling game %s: %v", espnGame.GameID, err)
			// Use ESPN game as fallback
			reconciledGames = append(reconciledGames, espnGame)
			continue
		}
		
		reconciledGames = append(reconciledGames, reconciled)
	}
	
	log.Printf("Reconciled %d games (Conflicts: %d, Google: %d, ESPN: %d)",
		e.metrics.TotalReconciliations,
		e.metrics.Conflicts,
		e.metrics.GooglePreferred,
		e.metrics.ESPNPreferred)
	
	return reconciledGames, nil
}

// createMatchupKey creates a normalized key for team matchups
func createMatchupKey(homeTeam, awayTeam string) string {
	return fmt.Sprintf("%s_vs_%s", normalizeTeamName(homeTeam), normalizeTeamName(awayTeam))
}

// normalizeTeamName converts team names to a standard format
func normalizeTeamName(teamName string) string {
	// Convert to lowercase and get abbreviation if possible
	abbr := google.GetTeamAbbreviation(teamName)
	if abbr != teamName {
		return abbr
	}
	return teamName
}

// ConvertESPNToStoreGame converts ESPN parsed data to store.Game
// This is a helper function to work with ESPN parser output
func ConvertESPNToStoreGame(espnData map[string]interface{}, seasonID int) (*store.Game, error) {
	// Use the existing ESPN parser
	games, err := espn.ParseScoreboardGames(map[string]interface{}{
		"events": []interface{}{espnData},
	}, seasonID)
	
	if err != nil || len(games) == 0 {
		return nil, fmt.Errorf("failed to parse ESPN data: %w", err)
	}
	
	return games[0], nil
}

