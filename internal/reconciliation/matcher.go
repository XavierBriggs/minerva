package reconciliation

import (
	"strings"

	"github.com/fortuna/minerva/internal/ingest/google"
	"github.com/fortuna/minerva/internal/store"
)

// Matcher handles matching games across different data sources
type Matcher struct {
	teamAbbreviations map[int]string // teamID -> abbreviation
}

// NewMatcher creates a new game matcher
func NewMatcher(teams []*store.Team) *Matcher {
	abbrevMap := make(map[int]string)
	for _, team := range teams {
		abbrevMap[team.TeamID] = team.Abbreviation
	}
	
	return &Matcher{
		teamAbbreviations: abbrevMap,
	}
}

// FindMatchingGoogleGame finds a Google game that matches an ESPN game
func (m *Matcher) FindMatchingGoogleGame(espnGame *store.Game, googleGames []google.LiveGame) *google.LiveGame {
	// Get team abbreviations for ESPN game
	homeAbbr, ok := m.teamAbbreviations[espnGame.HomeTeamID]
	if !ok {
		return nil
	}
	
	awayAbbr, ok := m.teamAbbreviations[espnGame.AwayTeamID]
	if !ok {
		return nil
	}
	
	// Try to find matching game
	for i := range googleGames {
		game := &googleGames[i]
		
		// Normalize Google team names
		googleHomeAbbr := google.GetTeamAbbreviation(game.HomeTeam)
		googleAwayAbbr := google.GetTeamAbbreviation(game.AwayTeam)
		
		// Check if teams match
		if matchTeams(homeAbbr, googleHomeAbbr) && matchTeams(awayAbbr, googleAwayAbbr) {
			return game
		}
	}
	
	return nil
}

// FindMatchingESPNGame finds an ESPN game that matches a Google game
func (m *Matcher) FindMatchingESPNGame(googleGame *google.LiveGame, espnGames []*store.Game) *store.Game {
	// Normalize Google team names
	googleHomeAbbr := google.GetTeamAbbreviation(googleGame.HomeTeam)
	googleAwayAbbr := google.GetTeamAbbreviation(googleGame.AwayTeam)
	
	// Try to find matching game
	for _, espnGame := range espnGames {
		homeAbbr, ok := m.teamAbbreviations[espnGame.HomeTeamID]
		if !ok {
			continue
		}
		
		awayAbbr, ok := m.teamAbbreviations[espnGame.AwayTeamID]
		if !ok {
			continue
		}
		
		// Check if teams match
		if matchTeams(homeAbbr, googleHomeAbbr) && matchTeams(awayAbbr, googleAwayAbbr) {
			return espnGame
		}
	}
	
	return nil
}

// matchTeams checks if two team identifiers match
func matchTeams(abbr1, abbr2 string) bool {
	// Exact match
	if strings.EqualFold(abbr1, abbr2) {
		return true
	}
	
	// Handle special cases
	specialCases := map[string][]string{
		"LAL": {"Lakers", "Los Angeles Lakers", "LA Lakers"},
		"LAC": {"Clippers", "Los Angeles Clippers", "LA Clippers"},
		"GSW": {"Warriors", "Golden State Warriors", "GS Warriors"},
		"BKN": {"Nets", "Brooklyn Nets"},
		"NYK": {"Knicks", "New York Knicks", "NY Knicks"},
		"PHX": {"Suns", "Phoenix Suns"},
		"SAS": {"Spurs", "San Antonio Spurs", "SA Spurs"},
	}
	
	// Check if either abbr matches any variant
	for key, variants := range specialCases {
		if strings.EqualFold(abbr1, key) || strings.EqualFold(abbr2, key) {
			for _, variant := range variants {
				if strings.Contains(strings.ToLower(abbr1), strings.ToLower(variant)) ||
					strings.Contains(strings.ToLower(abbr2), strings.ToLower(variant)) {
					return true
				}
			}
		}
	}
	
	return false
}

// MatchAndReconcileAll matches and reconciles all games from both sources
func (m *Matcher) MatchAndReconcileAll(espnGames []*store.Game, googleGames []google.LiveGame, engine *Engine) ([]*store.Game, error) {
	var reconciledGames []*store.Game
	matchedGoogleGames := make(map[int]bool)
	
	// Process ESPN games and find matching Google games
	for _, espnGame := range espnGames {
		googleGame := m.FindMatchingGoogleGame(espnGame, googleGames)
		
		if googleGame != nil {
			// Mark as matched
			for i := range googleGames {
				if &googleGames[i] == googleGame {
					matchedGoogleGames[i] = true
					break
				}
			}
		}
		
		// Reconcile (googleGame may be nil if no match)
		reconciled, err := engine.ReconcileGame(espnGame, googleGame)
		if err != nil {
			// Fallback to ESPN data
			reconciledGames = append(reconciledGames, espnGame)
			continue
		}
		
		reconciledGames = append(reconciledGames, reconciled)
	}
	
	// Add any Google games that weren't matched
	// (These are games ESPN doesn't know about yet)
	for i, googleGame := range googleGames {
		if !matchedGoogleGames[i] {
			// Convert to store.Game format  
			// TODO: Get actual season_id dynamically instead of hardcoding
			game := google.ConvertToStoreGame(googleGame, 1)
			reconciledGames = append(reconciledGames, game)
		}
	}
	
	return reconciledGames, nil
}

