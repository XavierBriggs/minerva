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
	HomeRecord    string // W-L record e.g. "11-4"
	AwayRecord    string // W-L record e.g. "10-9"
	HomeLogoURL   string // Team logo URL from Google CDN
	AwayLogoURL   string // Team logo URL from Google CDN
	GameStatus    string
	Period        int
	TimeRemaining string
	IsLive        bool
	IsScheduled   bool
	IsFinal       bool
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

	// Extract first team (usually away team in Google's layout)
	firstTeam := s.Find("div.imso_mh__first-tn-ed")
	if firstTeam.Length() > 0 {
		// Team name from .imso_mh__tm-nm or nested span
		teamNameEl := firstTeam.Find("div.imso_mh__tm-nm")
		if teamNameEl.Length() > 0 {
			game.AwayTeam = strings.TrimSpace(teamNameEl.Find("span").First().Text())
			if game.AwayTeam == "" {
				game.AwayTeam = strings.TrimSpace(teamNameEl.Text())
			}
		}

		// Team record from .imso_mh__tm-wlr - format: "(11 - 4)"
		recordEl := firstTeam.Find("div.imso_mh__tm-wlr")
		if recordEl.Length() > 0 {
			game.AwayRecord = parseTeamRecord(recordEl.Text())
		}

		// Team logo URL
		logoEl := firstTeam.Find("img")
		if src, exists := logoEl.Attr("src"); exists {
			game.AwayLogoURL = normalizeLogoURL(src)
		}
	}

	// Extract second team (usually home team)
	secondTeam := s.Find("div.imso_mh__second-tn-ed")
	if secondTeam.Length() > 0 {
		teamNameEl := secondTeam.Find("div.imso_mh__tm-nm")
		if teamNameEl.Length() > 0 {
			game.HomeTeam = strings.TrimSpace(teamNameEl.Find("span").First().Text())
			if game.HomeTeam == "" {
				game.HomeTeam = strings.TrimSpace(teamNameEl.Text())
			}
		}

		// Team record
		recordEl := secondTeam.Find("div.imso_mh__tm-wlr")
		if recordEl.Length() > 0 {
			game.HomeRecord = parseTeamRecord(recordEl.Text())
		}

		// Team logo URL
		logoEl := secondTeam.Find("img")
		if src, exists := logoEl.Attr("src"); exists {
			game.HomeLogoURL = normalizeLogoURL(src)
		}
	}

	// Extract scores - left score is away, right score is home
	leftScore := s.Find("div.imso_mh__l-tm-sc")
	if leftScore.Length() > 0 {
		scoreText := strings.TrimSpace(leftScore.Text())
		if scoreVal, err := strconv.Atoi(scoreText); err == nil {
			game.AwayScore = scoreVal
		}
	}

	rightScore := s.Find("div.imso_mh__r-tm-sc")
	if rightScore.Length() > 0 {
		scoreText := strings.TrimSpace(rightScore.Text())
		if scoreVal, err := strconv.Atoi(scoreText); err == nil {
			game.HomeScore = scoreVal
		}
	}

	// Extract game status from multiple possible locations
	// Try .imso_mh__stts-l first (live games have clock here)
	statusContainer := s.Find("div.imso_mh__stts-l")
	if statusContainer.Length() > 0 {
		// Look for quarter/time display like "Q4 - 00:47"
		statusText := strings.TrimSpace(statusContainer.Text())
		game.GameStatus = statusText
	}

	// Fallback to .imso_mh__ft-mtch for final games
	if game.GameStatus == "" {
		statusText := s.Find("span.imso_mh__ft-mtch").Text()
		game.GameStatus = strings.TrimSpace(statusText)
	}

	// Determine game state
	statusLower := strings.ToLower(game.GameStatus)
	game.IsLive = strings.Contains(statusLower, "live") ||
		strings.Contains(statusLower, "q1") ||
		strings.Contains(statusLower, "q2") ||
		strings.Contains(statusLower, "q3") ||
		strings.Contains(statusLower, "q4") ||
		strings.Contains(statusLower, "ot") ||
		strings.Contains(statusLower, "half") ||
		regexp.MustCompile(`\d{1,2}:\d{2}`).MatchString(game.GameStatus) // Has game clock

	game.IsFinal = strings.Contains(statusLower, "final")
	game.IsScheduled = !game.IsLive && !game.IsFinal

	// Extract period and time
	game.Period, game.TimeRemaining = parseGameClock(game.GameStatus)

	// Only return if we have valid team names
	if game.HomeTeam != "" && game.AwayTeam != "" {
		return game
	}

	return nil
}

// parseTeamRecord extracts W-L record from format like "(11 - 4)"
func parseTeamRecord(text string) string {
	text = strings.TrimSpace(text)
	// Remove parentheses
	text = strings.TrimPrefix(text, "(")
	text = strings.TrimSuffix(text, ")")
	// Normalize spacing: "11 - 4" -> "11-4"
	text = strings.ReplaceAll(text, " ", "")
	// Handle various formats: "11-4", "11 - 4", "11 and 4"
	text = strings.ReplaceAll(text, "and", "-")
	return text
}

// normalizeLogoURL ensures logo URL has proper protocol
func normalizeLogoURL(url string) string {
	if strings.HasPrefix(url, "//") {
		return "https:" + url
	}
	return url
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
// teamLookup is optional - if provided, it will try to resolve team IDs
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

	// Team IDs - try to resolve from abbreviation lookup
	// These will be set properly during reconciliation with ESPN data
	// or looked up from the database by the caller
	game.HomeTeamID = 0 // 0 = unresolved, will be matched during reconciliation
	game.AwayTeamID = 0

	return game
}

// ConvertToStoreGameWithTeams converts a Google LiveGame to a store.Game with team ID lookup
func ConvertToStoreGameWithTeams(liveGame LiveGame, seasonID int, teamLookup map[string]int) *store.Game {
	game := ConvertToStoreGame(liveGame, seasonID)

	// Try to resolve team IDs from the lookup map
	if teamLookup != nil {
		homeAbbr := GetTeamAbbreviation(liveGame.HomeTeam)
		awayAbbr := GetTeamAbbreviation(liveGame.AwayTeam)

		if id, ok := teamLookup[homeAbbr]; ok {
			game.HomeTeamID = id
		}
		if id, ok := teamLookup[awayAbbr]; ok {
			game.AwayTeamID = id
		}
	}

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
	// Use explicit flags first (most reliable)
	if game.IsLive {
		return "in_progress"
	}
	if game.IsFinal {
		return "final"
	}
	if game.IsScheduled {
		return "scheduled"
	}

	// Fallback to text parsing
	statusLower := strings.ToLower(game.GameStatus)
	if strings.Contains(statusLower, "final") {
		return "final"
	}
	if strings.Contains(statusLower, "live") ||
		strings.Contains(statusLower, "q1") ||
		strings.Contains(statusLower, "q2") ||
		strings.Contains(statusLower, "q3") ||
		strings.Contains(statusLower, "q4") ||
		strings.Contains(statusLower, "ot") ||
		strings.Contains(statusLower, "half") {
		return "in_progress"
	}
	if strings.Contains(statusLower, "postponed") {
		return "postponed"
	}
	if strings.Contains(statusLower, "cancelled") || strings.Contains(statusLower, "canceled") {
		return "cancelled"
	}

	// Default to scheduled if unclear
	return "scheduled"
}

// TeamNameToAbbreviation maps common team names to abbreviations
// This should be replaced with a database lookup in production
var TeamNameToAbbreviation = map[string]string{
	"lakers":       "LAL",
	"celtics":      "BOS",
	"warriors":     "GSW",
	"nets":         "BKN",
	"knicks":       "NYK",
	"heat":         "MIA",
	"bucks":        "MIL",
	"bulls":        "CHI",
	"cavaliers":    "CLE",
	"mavericks":    "DAL",
	"nuggets":      "DEN",
	"rockets":      "HOU",
	"clippers":     "LAC",
	"grizzlies":    "MEM",
	"timberwolves": "MIN",
	"pelicans":     "NOP",
	"thunder":      "OKC",
	"magic":        "ORL",
	"76ers":        "PHI",
	"suns":         "PHX",
	"blazers":      "POR",
	"kings":        "SAC",
	"spurs":        "SAS",
	"raptors":      "TOR",
	"jazz":         "UTA",
	"wizards":      "WAS",
	"hawks":        "ATL",
	"hornets":      "CHA",
	"pistons":      "DET",
	"pacers":       "IND",
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
