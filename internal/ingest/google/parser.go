package google

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/fortuna/minerva/internal/store"
)

// LiveGame represents a live game scraped from Google
type LiveGame struct {
	HomeTeam      string
	AwayTeam      string
	HomeScore     int
	AwayScore     int
	GameStatus    string
	Period        int
	TimeRemaining string
	IsLive        bool
}

// ParseLiveGames extracts live NBA games from Google search results
func ParseLiveGames(doc *goquery.Document) ([]LiveGame, error) {
	var games []LiveGame
	
	// Google Sports uses various selectors depending on the page structure
	// We'll try multiple strategies to extract game data
	
	// Strategy 1: Look for sports card widgets
	doc.Find("div.imso_mh__lv-m-stl-cont").Each(func(i int, s *goquery.Selection) {
		game := parseSportsCard(s)
		if game != nil {
			games = append(games, *game)
		}
	})
	
	// Strategy 2: Look for game result divs
	if len(games) == 0 {
		doc.Find("div[class*='sports']").Each(func(i int, s *goquery.Selection) {
			game := parseSportsDiv(s)
			if game != nil {
				games = append(games, *game)
			}
		})
	}
	
	log.Printf("Parsed %d live games from Google", len(games))
	return games, nil
}

// parseSportsCard extracts game info from a Google sports card widget
func parseSportsCard(s *goquery.Selection) *LiveGame {
	game := &LiveGame{}
	
	// Extract team names
	s.Find("div.imso_mh__first-tn-ed").Each(func(i int, team *goquery.Selection) {
		teamName := strings.TrimSpace(team.Text())
		if i == 0 {
			game.HomeTeam = teamName
		} else if i == 1 {
			game.AwayTeam = teamName
		}
	})
	
	// Extract scores
	s.Find("div.imso_mh__l-tm-sc").Each(func(i int, score *goquery.Selection) {
		scoreText := strings.TrimSpace(score.Text())
		scoreVal, err := strconv.Atoi(scoreText)
		if err == nil {
			if i == 0 {
				game.HomeScore = scoreVal
			} else if i == 1 {
				game.AwayScore = scoreVal
			}
		}
	})
	
	// Extract game status (Live, Final, etc.)
	statusText := s.Find("span.imso_mh__ft-mtch").Text()
	game.GameStatus = strings.TrimSpace(statusText)
	game.IsLive = strings.Contains(strings.ToLower(game.GameStatus), "live") ||
		strings.Contains(strings.ToLower(game.GameStatus), "q1") ||
		strings.Contains(strings.ToLower(game.GameStatus), "q2") ||
		strings.Contains(strings.ToLower(game.GameStatus), "q3") ||
		strings.Contains(strings.ToLower(game.GameStatus), "q4")
	
	// Extract period and time
	game.Period, game.TimeRemaining = parseGameClock(game.GameStatus)
	
	// Only return if we have valid team names
	if game.HomeTeam != "" && game.AwayTeam != "" {
		return game
	}
	
	return nil
}

// parseSportsDiv is a fallback parser for alternate Google structures
func parseSportsDiv(s *goquery.Selection) *LiveGame {
	// This is a simplified fallback parser
	// Google's HTML structure can vary, so we may need to adjust this
	
	text := s.Text()
	if !strings.Contains(strings.ToLower(text), "nba") {
		return nil
	}
	
	// Look for score patterns like "Lakers 105 - 98 Celtics"
	scorePattern := regexp.MustCompile(`(\w+)\s+(\d+)\s*-\s*(\d+)\s+(\w+)`)
	matches := scorePattern.FindStringSubmatch(text)
	
	if len(matches) == 5 {
		awayScore, _ := strconv.Atoi(matches[2])
		homeScore, _ := strconv.Atoi(matches[3])
		
		return &LiveGame{
			AwayTeam:   matches[1],
			HomeTeam:   matches[4],
			AwayScore:  awayScore,
			HomeScore:  homeScore,
			GameStatus: "Unknown",
			IsLive:     false,
		}
	}
	
	return nil
}

// parseGameClock extracts period and time remaining from status text
func parseGameClock(statusText string) (int, string) {
	statusLower := strings.ToLower(statusText)
	
	// Match patterns like "Q4 2:30", "3rd 5:45", "4th Quarter 1:23"
	periodMap := map[string]int{
		"q1": 1, "1st": 1, "first": 1,
		"q2": 2, "2nd": 2, "second": 2,
		"q3": 3, "3rd": 3, "third": 3,
		"q4": 4, "4th": 4, "fourth": 4,
		"ot": 5, "overtime": 5,
	}
	
	for key, period := range periodMap {
		if strings.Contains(statusLower, key) {
			// Try to extract time
			timePattern := regexp.MustCompile(`(\d{1,2}:\d{2})`)
			if matches := timePattern.FindStringSubmatch(statusText); len(matches) > 0 {
				return period, matches[1]
			}
			return period, ""
		}
	}
	
	// Check for halftime
	if strings.Contains(statusLower, "half") {
		return 2, "Halftime"
	}
	
	return 0, ""
}

// ConvertToStoreGame converts a Google LiveGame to a store.Game
func ConvertToStoreGame(liveGame LiveGame, seasonID int) *store.Game {
	game := &store.Game{
		Sport:      "basketball_nba",
		ExternalID: generateGameID(liveGame),
		SeasonID:   seasonID,
		GameDate:   time.Now(), // Use current date for live games
		GameTime:   sql.NullTime{Time: time.Now(), Valid: true},
		HomeScore:  sql.NullInt32{Int32: int32(liveGame.HomeScore), Valid: true},
		AwayScore:  sql.NullInt32{Int32: int32(liveGame.AwayScore), Valid: true},
		Status:     parseGameStatus(liveGame),
	}
	
	if liveGame.Period > 0 {
		game.Period = sql.NullInt32{Int32: int32(liveGame.Period), Valid: true}
	}
	
	if liveGame.TimeRemaining != "" {
		game.Clock = sql.NullString{String: liveGame.TimeRemaining, Valid: true}
	}
	
	// Team IDs would need to be looked up from team names
	// For now, set as placeholders
	game.HomeTeamID = -1
	game.AwayTeamID = -1
	
	return game
}

// generateGameID creates a unique game ID from team names and date
func generateGameID(game LiveGame) string {
	dateStr := time.Now().Format("20060102")
	homeTeam := strings.ReplaceAll(strings.ToLower(game.HomeTeam), " ", "")
	awayTeam := strings.ReplaceAll(strings.ToLower(game.AwayTeam), " ", "")
	return fmt.Sprintf("google_%s_%s_%s", dateStr, awayTeam, homeTeam)
}

// parseGameStatus converts Google game status to our standard format
func parseGameStatus(game LiveGame) string {
	if game.IsLive {
		return "in_progress"
	}
	
	statusLower := strings.ToLower(game.GameStatus)
	if strings.Contains(statusLower, "final") {
		return "final"
	}
	if strings.Contains(statusLower, "scheduled") {
		return "scheduled"
	}
	if strings.Contains(statusLower, "postponed") {
		return "postponed"
	}
	if strings.Contains(statusLower, "cancelled") {
		return "cancelled"
	}
	
	// Default to scheduled if unclear
	return "scheduled"
}

// TeamNameToAbbreviation maps common team names to abbreviations
// This should be replaced with a database lookup in production
var TeamNameToAbbreviation = map[string]string{
	"lakers":     "LAL",
	"celtics":    "BOS",
	"warriors":   "GSW",
	"nets":       "BKN",
	"knicks":     "NYK",
	"heat":       "MIA",
	"bucks":      "MIL",
	"bulls":      "CHI",
	"cavaliers":  "CLE",
	"mavericks":  "DAL",
	"nuggets":    "DEN",
	"rockets":    "HOU",
	"clippers":   "LAC",
	"grizzlies":  "MEM",
	"timberwolves": "MIN",
	"pelicans":   "NOP",
	"thunder":    "OKC",
	"magic":      "ORL",
	"76ers":      "PHI",
	"suns":       "PHX",
	"blazers":    "POR",
	"kings":      "SAC",
	"spurs":      "SAS",
	"raptors":    "TOR",
	"jazz":       "UTA",
	"wizards":    "WAS",
	"hawks":      "ATL",
	"hornets":    "CHA",
	"pistons":    "DET",
	"pacers":     "IND",
}

// GetTeamAbbreviation returns team abbreviation from full name
func GetTeamAbbreviation(teamName string) string {
	nameLower := strings.ToLower(strings.TrimSpace(teamName))
	
	// Try exact match first
	if abbr, ok := TeamNameToAbbreviation[nameLower]; ok {
		return abbr
	}
	
	// Try partial match
	for key, abbr := range TeamNameToAbbreviation {
		if strings.Contains(nameLower, key) {
			return abbr
		}
	}
	
	// Return original if no match
	return teamName
}

