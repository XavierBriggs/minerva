package main

import (
	"database/sql"
	"log"

	"github.com/fortuna/minerva/internal/ingest/google"
	"github.com/fortuna/minerva/internal/reconciliation"
	"github.com/fortuna/minerva/internal/store"
)

// Test utility for reconciliation engine
func main() {
	log.Println("Testing Reconciliation Engine")
	log.Println("==============================")

	// Create test data
	espnGame := createTestESPNGame()
	googleGame := createTestGoogleGame()

	// Test all strategies
	strategies := []reconciliation.ReconciliationStrategy{
		reconciliation.SmartMerge,
		reconciliation.PreferLatest,
		reconciliation.PreferAuthoritative,
	}

	for _, strategy := range strategies {
		log.Printf("\n--- Testing Strategy: %s ---", strategy)
		
		engine := reconciliation.NewEngine(strategy)
		
		// Test 1: Both sources available
		log.Println("\nTest 1: Both sources available (Live game)")
		merged, err := engine.ReconcileGame(espnGame, googleGame)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
		displayGame(merged)
		
		// Test 2: Only ESPN available
		log.Println("\nTest 2: Only ESPN available")
		merged, err = engine.ReconcileGame(espnGame, nil)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
		displayGame(merged)
		
		// Test 3: Only Google available
		log.Println("\nTest 3: Only Google available")
		merged, err = engine.ReconcileGame(nil, googleGame)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
		displayGame(merged)
		
		// Display metrics
		metrics := engine.GetMetrics()
		log.Printf("\nMetrics:")
		log.Printf("  Total: %d", metrics.TotalReconciliations)
		log.Printf("  Conflicts: %d", metrics.Conflicts)
		log.Printf("  Google Preferred: %d", metrics.GooglePreferred)
		log.Printf("  ESPN Preferred: %d", metrics.ESPNPreferred)
	}

	// Test conflict detection
	log.Println("\n--- Testing Conflict Detection ---")
	testConflictDetection()

	// Test team matching
	log.Println("\n--- Testing Team Matching ---")
	testTeamMatching()

	log.Println("\n==============================")
	log.Println("✓ All Reconciliation Tests Complete")
}

func createTestESPNGame() *store.Game {
	return &store.Game{
		GameID:     "401584894",
		SeasonID:   "2024-25",
		HomeTeamID: 13, // Lakers
		AwayTeamID: 2,  // Celtics
		HomeScore:  sql.NullInt32{Int32: 105, Valid: true},
		AwayScore:  sql.NullInt32{Int32: 98, Valid: true},
		GameStatus: "in_progress",
		Period:     sql.NullInt32{Int32: 4, Valid: true},
		TimeRemaining: sql.NullString{String: "2:30", Valid: true},
	}
}

func createTestGoogleGame() *google.LiveGame {
	return &google.LiveGame{
		HomeTeam:      "Lakers",
		AwayTeam:      "Celtics",
		HomeScore:     107,  // Slightly different from ESPN
		AwayScore:     100,  // Slightly different from ESPN
		GameStatus:    "Q4 2:15",
		Period:        4,
		TimeRemaining: "2:15",
		IsLive:        true,
	}
}

func displayGame(game *store.Game) {
	log.Printf("  Game ID: %s", game.GameID)
	log.Printf("  Season: %s", game.SeasonID)
	if game.HomeScore.Valid && game.AwayScore.Valid {
		log.Printf("  Score: %d - %d", game.AwayScore.Int32, game.HomeScore.Int32)
	}
	log.Printf("  Status: %s", game.GameStatus)
	if game.Period.Valid {
		log.Printf("  Period: %d", game.Period.Int32)
	}
	if game.TimeRemaining.Valid {
		log.Printf("  Time: %s", game.TimeRemaining.String)
	}
}

func testConflictDetection() {
	engine := reconciliation.NewEngine(reconciliation.SmartMerge)

	// Test 1: Large score discrepancy
	log.Println("\nTest 1: Large score discrepancy")
	espnGame := createTestESPNGame()
	googleGame := createTestGoogleGame()
	googleGame.HomeScore = 130 // 25 point difference!
	
	merged, err := engine.ReconcileGame(espnGame, googleGame)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		log.Printf("Merged score: %d - %d", merged.AwayScore.Int32, merged.HomeScore.Int32)
	}
	log.Printf("Conflicts detected: %d", engine.GetMetrics().Conflicts)

	// Test 2: Status conflict
	log.Println("\nTest 2: Status conflict")
	engine.ResetMetrics()
	espnGame2 := createTestESPNGame()
	espnGame2.GameStatus = "final"
	googleGame2 := createTestGoogleGame()
	googleGame2.IsLive = true
	googleGame2.GameStatus = "Q4 2:30"
	
	merged, err = engine.ReconcileGame(espnGame2, googleGame2)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		log.Printf("Merged status: %s", merged.GameStatus)
	}
	log.Printf("Conflicts detected: %d", engine.GetMetrics().Conflicts)
}

func testTeamMatching() {
	// Create test teams
	teams := []*store.Team{
		{TeamID: 13, TeamAbbreviation: "LAL", TeamName: "Los Angeles Lakers"},
		{TeamID: 2, TeamAbbreviation: "BOS", TeamName: "Boston Celtics"},
		{TeamID: 9, TeamAbbreviation: "GSW", TeamName: "Golden State Warriors"},
	}

	matcher := reconciliation.NewMatcher(teams)

	// Test ESPN game
	espnGame := createTestESPNGame()
	espnGame.HomeTeamID = 13 // Lakers
	espnGame.AwayTeamID = 2  // Celtics

	// Test Google games with variations
	googleGames := []google.LiveGame{
		{HomeTeam: "Lakers", AwayTeam: "Celtics"},                    // Simple names
		{HomeTeam: "Los Angeles Lakers", AwayTeam: "Boston Celtics"}, // Full names
		{HomeTeam: "LA Lakers", AwayTeam: "Celtics"},                 // Variant
		{HomeTeam: "Warriors", AwayTeam: "Nets"},                     // Different game
	}

	for i, googleGame := range googleGames {
		log.Printf("\nTest %d: %s vs %s", i+1, googleGame.AwayTeam, googleGame.HomeTeam)
		match := matcher.FindMatchingGoogleGame(espnGame, googleGames)
		if match != nil {
			log.Printf("  ✓ Match found: %s vs %s", match.AwayTeam, match.HomeTeam)
		} else {
			log.Printf("  ✗ No match")
		}
	}
}

