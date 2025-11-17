package espn

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fortuna/minerva/internal/store"
)

// ESPN stat labels for dynamic parsing (more robust than hardcoded indices)
// This approach works across different sports and API updates
const (
	statLabelMinutes   = "MIN"
	statLabelPoints    = "PTS"
	statLabelOffReb    = "OREB"
	statLabelDefReb    = "DREB"
	statLabelReb       = "REB"
	statLabelAst       = "AST"
	statLabelStl       = "STL"
	statLabelBlk       = "BLK"
	statLabelTO        = "TO"
	statLabelFG        = "FG"   // Format: "X-Y"
	statLabel3PT       = "3PT"  // Format: "X-Y"
	statLabelFT        = "FT"   // Format: "X-Y"
	statLabelPF        = "PF"
	statLabelPlusMinus = "+/-"
)

// ParseScoreboardGames extracts games without metadata (legacy helper).
func ParseScoreboardGames(scoreboardData map[string]interface{}, seasonID int) ([]*store.Game, error) {
	detailed, err := ParseScoreboardGamesDetailed(scoreboardData, seasonID)
	if err != nil {
		return nil, err
	}

	games := make([]*store.Game, 0, len(detailed))
	for _, parsed := range detailed {
		games = append(games, parsed.Game)
	}
	return games, nil
}

// ParseScoreboardGamesDetailed returns parsed games plus team metadata.
func ParseScoreboardGamesDetailed(scoreboardData map[string]interface{}, seasonID int) ([]*ParsedGame, error) {
	events := extractArray(scoreboardData, "events")
	if len(events) == 0 {
		// No games on this date - this is normal, not an error
		return []*ParsedGame{}, nil
	}

	var games []*ParsedGame
	for _, eventInterface := range events {
		event := eventInterface.(map[string]interface{})
		game, err := parseGameFromEventDetailed(event, seasonID)
		if err != nil {
			// Log parsing errors instead of silently skipping
			gameID := extractString(event, "id")
			fmt.Printf("[parser] Warning: Skipping game %s: %v\n", gameID, err)
			continue
		}
		games = append(games, game)
	}

	return games, nil
}

func parseGameFromEventDetailed(event map[string]interface{}, seasonID int) (*ParsedGame, error) {
	game := &store.Game{
		Sport:      "basketball_nba",
		ExternalID: extractString(event, "id"),
		SeasonID:   seasonID,
	}

	if dateStr := extractString(event, "date"); dateStr != "" {
		// Try RFC3339 first, then fallback to ESPN's shortened format (no seconds)
		var gameTime time.Time
		var err error
		
		gameTime, err = time.Parse(time.RFC3339, dateStr)
		if err != nil {
			// ESPN sometimes omits seconds: "2025-11-15T01:00Z"
			gameTime, err = time.Parse("2006-01-02T15:04Z", dateStr)
		}
		
		if err == nil {
			// ESPN gives UTC time, convert to EST for storage
			est, _ := time.LoadLocation("America/New_York")
			gameTimeEST := gameTime.In(est)
			game.GameDate = gameTimeEST
			game.GameTime = sql.NullTime{Time: gameTimeEST, Valid: true}
		} else {
			fmt.Printf("[parser] Warning: Failed to parse date '%s' for game %s: %v\n", dateStr, game.ExternalID, err)
		}
	} else {
		fmt.Printf("[parser] Warning: No date field found for game %s\n", game.ExternalID)
	}

	status := extractMap(event, "status")
	game.Status = normalizeGameStatus(parseGameStatus(status))

	if period := extractInt(status, "period"); period > 0 {
		game.Period = sql.NullInt32{Int32: int32(period), Valid: true}
	}
	if clock := extractString(status, "displayClock"); clock != "" {
		game.Clock = sql.NullString{String: clock, Valid: true}
	}

	competitions := extractArray(event, "competitions")
	if len(competitions) == 0 {
		return nil, fmt.Errorf("no competitions found for game %s", game.GameID)
	}

	comp := competitions[0].(map[string]interface{})
	competitors := extractArray(comp, "competitors")
	if len(competitors) < 2 {
		return nil, fmt.Errorf("insufficient competitors for game %s", game.GameID)
	}

	var homeMeta, awayMeta TeamMeta
	for _, compInterface := range competitors {
		competitor := compInterface.(map[string]interface{})
		homeAway := extractString(competitor, "homeAway")
		team := extractMap(competitor, "team")
		meta := TeamMeta{
			Abbreviation: strings.ToUpper(extractString(team, "abbreviation")),
			ESPNID:       extractString(team, "id"),
			DisplayName:  extractString(team, "displayName"),
		}

		score := extractInt(competitor, "score")

		if homeAway == "home" {
			homeMeta = meta
			game.HomeTeamID = -1
			if score > 0 {
				game.HomeScore = sql.NullInt32{Int32: int32(score), Valid: true}
			}
		} else if homeAway == "away" {
			awayMeta = meta
			game.AwayTeamID = -1
			if score > 0 {
				game.AwayScore = sql.NullInt32{Int32: int32(score), Valid: true}
			}
		}
	}

	venue := extractMap(comp, "venue")
	if venueName := extractString(venue, "fullName"); venueName != "" {
		game.Venue = sql.NullString{String: venueName, Valid: true}
	}
	if attendance := extractInt(comp, "attendance"); attendance > 0 {
		game.Attendance = sql.NullInt32{Int32: int32(attendance), Valid: true}
	}

	// SeasonType is no longer stored in Game struct (v2 schema)
	// It's managed through the seasons table
	var seasonType string
	if season := extractMap(event, "season"); len(season) > 0 {
		if phase := extractInt(season, "type"); phase > 0 {
			seasonType = seasonTypeFromCode(phase)
		}
	}

	return &ParsedGame{
		Game:       game,
		HomeTeam:   homeMeta,
		AwayTeam:   awayMeta,
		SeasonType: seasonType,
	}, nil
}

// ParseBoxScore returns player stats without metadata (legacy helper).
func ParseBoxScore(summaryData map[string]interface{}, gameID string) ([]*store.PlayerGameStats, error) {
	detailed, err := ParseBoxScoreDetailed(summaryData, gameID)
	if err != nil {
		return nil, err
	}

	stats := make([]*store.PlayerGameStats, 0, len(detailed))
	for _, parsed := range detailed {
		stats = append(stats, parsed.Stats)
	}
	return stats, nil
}

// ParseBoxScoreDetailed returns player stats and metadata.
func ParseBoxScoreDetailed(summaryData map[string]interface{}, gameID string) ([]*ParsedPlayerStats, error) {
	boxscore := extractMap(summaryData, "boxscore")
	if len(boxscore) == 0 {
		return nil, fmt.Errorf("no boxscore data found")
	}

	// ESPN API uses either "players" or "teams" depending on the endpoint/version
	// Try both for robustness
	playersData := extractArray(boxscore, "players")
	if len(playersData) == 0 {
		playersData = extractArray(boxscore, "teams")
	}
	if len(playersData) == 0 {
		return nil, fmt.Errorf("no players/teams data in boxscore")
	}

	var allStats []*ParsedPlayerStats
	for _, teamDataInterface := range playersData {
		teamData := teamDataInterface.(map[string]interface{})
		team := extractMap(teamData, "team")
		teamAbbr := strings.ToUpper(extractString(team, "abbreviation"))

		statistics := extractArray(teamData, "statistics")
		if len(statistics) == 0 {
			continue
		}

		statGroup := statistics[0].(map[string]interface{})
		
		// Build stat name -> index mapping for dynamic parsing
		statNames := extractArray(statGroup, "names")
		statIndexMap := make(map[string]int)
		for i, nameInterface := range statNames {
			if name, ok := nameInterface.(string); ok {
				statIndexMap[name] = i
			}
		}

		athletes := extractArray(statGroup, "athletes")

		for _, athleteInterface := range athletes {
			athleteData := athleteInterface.(map[string]interface{})

			if didNotPlay, ok := athleteData["didNotPlay"].(bool); ok && didNotPlay {
				continue
			}

			playerStat, err := parsePlayerStatsDetailed(athleteData, gameID, teamAbbr, statIndexMap)
			if err != nil {
				continue
			}

			allStats = append(allStats, playerStat)
		}
	}

	return allStats, nil
}

func parsePlayerStatsDetailed(athleteData map[string]interface{}, gameID string, teamAbbr string, statIndexMap map[string]int) (*ParsedPlayerStats, error) {
	athlete := extractMap(athleteData, "athlete")

	playerStats := &store.PlayerGameStats{
		// GameID will be set by the ingester after game is persisted
		PlayerID: -1,
		TeamID:   -1,
	}

	parsed := &ParsedPlayerStats{
		Stats:        playerStats,
		TeamAbbr:     teamAbbr,
		ESPNPlayerID: extractString(athlete, "id"),
		PlayerName:   fallbackString(extractString(athlete, "displayName"), extractString(athlete, "shortName")),
		Jersey:       extractString(athlete, "jersey"),
		Height:       extractString(athlete, "height"),
	}

	if weightStr := fmt.Sprint(athlete["weight"]); weightStr != "" {
		if w, err := strconv.Atoi(weightStr); err == nil && w > 0 {
			parsed.Weight = w
		}
	}

	if position := extractMap(athlete, "position"); len(position) > 0 {
		parsed.Position = extractString(position, "abbreviation")
	}

	if dob := extractString(athlete, "dateOfBirth"); dob != "" {
		if ts, err := time.Parse(time.RFC3339, dob); err == nil {
			parsed.BirthDate = &ts
		}
	}

	stats := extractArray(athleteData, "stats")
	if len(stats) == 0 {
		return nil, fmt.Errorf("no stats array for player")
	}

	// Helper to safely get stat by label
	getStat := func(label string) interface{} {
		if idx, ok := statIndexMap[label]; ok && idx < len(stats) {
			return stats[idx]
		}
		return nil
	}

	// Parse stats using dynamic labels (robust to API changes)
	if minStat := getStat(statLabelMinutes); minStat != nil {
		playerStats.MinutesPlayed = sql.NullFloat64{Float64: parseMinutes(fmt.Sprint(minStat)), Valid: true}
	}
	
	if ptsStat := getStat(statLabelPoints); ptsStat != nil {
		playerStats.Points = parseInt(ptsStat)
	}
	
	if orebStat := getStat(statLabelOffReb); orebStat != nil {
		playerStats.OffensiveRebounds = parseInt(orebStat)
	}
	
	if drebStat := getStat(statLabelDefReb); drebStat != nil {
		playerStats.DefensiveRebounds = parseInt(drebStat)
	}
	
	if rebStat := getStat(statLabelReb); rebStat != nil {
		playerStats.Rebounds = parseInt(rebStat)
	}
	
	if astStat := getStat(statLabelAst); astStat != nil {
		playerStats.Assists = parseInt(astStat)
	}
	
	if stlStat := getStat(statLabelStl); stlStat != nil {
		playerStats.Steals = parseInt(stlStat)
	}
	
	if blkStat := getStat(statLabelBlk); blkStat != nil {
		playerStats.Blocks = parseInt(blkStat)
	}
	
	if toStat := getStat(statLabelTO); toStat != nil {
		playerStats.Turnovers = parseInt(toStat)
	}

	if fgStat := getStat(statLabelFG); fgStat != nil {
		fgMadeAttempted := parseShotFormat(fmt.Sprint(fgStat))
		playerStats.FieldGoalsMade = fgMadeAttempted[0]
		playerStats.FieldGoalsAttempted = fgMadeAttempted[1]
	}

	if threePtStat := getStat(statLabel3PT); threePtStat != nil {
		threeMadeAttempted := parseShotFormat(fmt.Sprint(threePtStat))
		playerStats.ThreePointersMade = threeMadeAttempted[0]
		playerStats.ThreePointersAttempted = threeMadeAttempted[1]
	}

	if ftStat := getStat(statLabelFT); ftStat != nil {
		ftMadeAttempted := parseShotFormat(fmt.Sprint(ftStat))
		playerStats.FreeThrowsMade = ftMadeAttempted[0]
		playerStats.FreeThrowsAttempted = ftMadeAttempted[1]
	}

	if pfStat := getStat(statLabelPF); pfStat != nil {
		playerStats.PersonalFouls = parseInt(pfStat)
	}

	if plusMinusStat := getStat(statLabelPlusMinus); plusMinusStat != nil {
		if plusMinus := parsePlusMinus(fmt.Sprint(plusMinusStat)); plusMinus != 0 {
			playerStats.PlusMinus = sql.NullInt32{Int32: int32(plusMinus), Valid: true}
		}
	}

	playerStats.TrueShootingPct = calculateTrueShootingPct(playerStats)
	playerStats.EffectiveFGPct = calculateEffectiveFGPct(playerStats)

	if starter, ok := athleteData["starter"].(bool); ok {
		playerStats.Starter = starter
	}

	return parsed, nil
}

// Helper functions

func extractString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if str, ok := v.(string); ok {
			return str
		}
	}
	return ""
}

func fallbackString(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func extractInt(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		return parseInt(v)
	}
	return 0
}

func extractMap(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key]; ok {
		if mapVal, ok := v.(map[string]interface{}); ok {
			return mapVal
		}
	}
	return map[string]interface{}{}
}

func extractArray(m map[string]interface{}, key string) []interface{} {
	if v, ok := m[key]; ok {
		if arrVal, ok := v.([]interface{}); ok {
			return arrVal
		}
	}
	return []interface{}{}
}

func parseInt(v interface{}) int {
	switch val := v.(type) {
	case float64:
		return int(val)
	case string:
		i, _ := strconv.Atoi(val)
		return i
	case int:
		return val
	default:
		return 0
	}
}

func parseMinutes(minutesStr string) float64 {
	if minutesStr == "" || minutesStr == "0" {
		return 0.0
	}

	if strings.Contains(minutesStr, ":") {
		parts := strings.Split(minutesStr, ":")
		mins, _ := strconv.Atoi(parts[0])
		secs := 0
		if len(parts) > 1 {
			secs, _ = strconv.Atoi(parts[1])
		}
		return float64(mins) + (float64(secs) / 60.0)
	}

	f, _ := strconv.ParseFloat(minutesStr, 64)
	return f
}

func parsePlusMinus(pmStr string) int {
	if pmStr == "" || pmStr == "0" {
		return 0
	}
	pmStr = strings.Replace(pmStr, "+", "", -1)
	i, _ := strconv.Atoi(pmStr)
	return i
}

func parseShotFormat(shotStr string) [2]int {
	parts := strings.Split(shotStr, "-")
	if len(parts) != 2 {
		return [2]int{0, 0}
	}
	made, _ := strconv.Atoi(parts[0])
	attempted, _ := strconv.Atoi(parts[1])
	return [2]int{made, attempted}
}

func parseGameStatus(status map[string]interface{}) string {
	statusType := extractMap(status, "type")

	if completed, ok := statusType["completed"].(bool); ok && completed {
		return "final"
	}

	if state, ok := statusType["state"].(string); ok {
		switch state {
		case "in":
			return "live"
		case "pre":
			return "scheduled"
		case "post":
			return "final"
		}
	}

	return "scheduled"
}

func normalizeGameStatus(status string) string {
	if status == "live" {
		return "in_progress"
	}
	return status
}

func seasonTypeFromCode(code int) string {
	switch code {
	case 1:
		return "preseason"
	case 2:
		return "regular"
	case 3:
		return "postseason"
	default:
		return "regular"
	}
}

func calculateTrueShootingPct(stats *store.PlayerGameStats) sql.NullFloat64 {
	if stats.FieldGoalsAttempted == 0 && stats.FreeThrowsAttempted == 0 {
		return sql.NullFloat64{Valid: false}
	}

	denominator := 2.0 * (float64(stats.FieldGoalsAttempted) + (0.44 * float64(stats.FreeThrowsAttempted)))
	if denominator == 0 {
		return sql.NullFloat64{Valid: false}
	}

	ts := float64(stats.Points) / denominator
	return sql.NullFloat64{Float64: ts, Valid: true}
}

func calculateEffectiveFGPct(stats *store.PlayerGameStats) sql.NullFloat64 {
	if stats.FieldGoalsAttempted == 0 {
		return sql.NullFloat64{Valid: false}
	}

	efg := (float64(stats.FieldGoalsMade) + (0.5 * float64(stats.ThreePointersMade))) / float64(stats.FieldGoalsAttempted)
	return sql.NullFloat64{Float64: efg, Valid: true}
}

// ParsedTeamStats holds team stats with metadata for ingestion
type ParsedTeamStats struct {
	Stats    *store.TeamGameStats
	TeamAbbr string
}

// ParseTeamStats extracts team-level statistics from ESPN game summary
// ESPN provides team stats in a different format than player stats - they're in a flat array
func ParseTeamStats(summaryData map[string]interface{}, gameID string) ([]*ParsedTeamStats, error) {
	boxscore := extractMap(summaryData, "boxscore")
	if len(boxscore) == 0 {
		return nil, fmt.Errorf("no boxscore data found")
	}

	// Get teams data - ESPN has team stats at .boxscore.teams[]
	teamsData := extractArray(boxscore, "teams")
	if len(teamsData) == 0 {
		return nil, fmt.Errorf("no teams data in boxscore")
	}

	var teamStats []*ParsedTeamStats
	for _, teamDataInterface := range teamsData {
		teamData := teamDataInterface.(map[string]interface{})
		team := extractMap(teamData, "team")
		teamAbbr := strings.ToUpper(extractString(team, "abbreviation"))

		// ESPN provides team stats as a flat array of stat objects
		// Each stat has: {name, displayValue, label}
		statistics := extractArray(teamData, "statistics")
		if len(statistics) == 0 {
			continue
		}

		parsed, err := parseTeamStatsFromStatArray(statistics, teamAbbr)
		if err == nil {
			teamStats = append(teamStats, parsed)
		}
	}

	return teamStats, nil
}

// parseTeamStatsFromStatArray parses ESPN's flat stat array format
func parseTeamStatsFromStatArray(statistics []interface{}, teamAbbr string) (*ParsedTeamStats, error) {
	stats := &store.TeamGameStats{
		GameID: -1,
		TeamID: -1,
	}

	// Helper to find stat by name or label
	getStat := func(names ...string) string {
		for _, statInterface := range statistics {
			statObj, ok := statInterface.(map[string]interface{})
			if !ok {
				continue
			}
			
			statName := extractString(statObj, "name")
			statLabel := extractString(statObj, "label")
			
			for _, searchName := range names {
				if statName == searchName || statLabel == searchName {
					return extractString(statObj, "displayValue")
				}
			}
		}
		return ""
	}

	// Field Goals - ESPN format: "49-88"
	if fgStr := getStat("fieldGoalsMade-fieldGoalsAttempted", "FG"); fgStr != "" {
		parts := strings.Split(fgStr, "-")
		if len(parts) == 2 {
			if made, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil {
				stats.FieldGoalsMade = made
			}
			if att, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
				stats.FieldGoalsAttempted = att
			}
		}
	}

	// 3-Pointers - ESPN format: "12-31"
	if tpStr := getStat("threePointFieldGoalsMade-threePointFieldGoalsAttempted", "3PT"); tpStr != "" {
		parts := strings.Split(tpStr, "-")
		if len(parts) == 2 {
			if made, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil {
				stats.ThreePointersMade = made
			}
			if att, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
				stats.ThreePointersAttempted = att
			}
		}
	}

	// Free Throws - ESPN format: "16-17"
	if ftStr := getStat("freeThrowsMade-freeThrowsAttempted", "FT"); ftStr != "" {
		parts := strings.Split(ftStr, "-")
		if len(parts) == 2 {
			if made, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil {
				stats.FreeThrowsMade = made
			}
			if att, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
				stats.FreeThrowsAttempted = att
			}
		}
	}

	// Rebounds
	if reb := getStat("totalRebounds", "REB"); reb != "" {
		if v, err := strconv.Atoi(reb); err == nil {
			stats.Rebounds = v
		}
	}
	if oreb := getStat("offensiveRebounds", "OR", "OREB"); oreb != "" {
		if v, err := strconv.Atoi(oreb); err == nil {
			stats.OffensiveRebounds = v
		}
	}
	if dreb := getStat("defensiveRebounds", "DR", "DREB"); dreb != "" {
		if v, err := strconv.Atoi(dreb); err == nil {
			stats.DefensiveRebounds = v
		}
	}

	// Other stats
	if ast := getStat("assists", "AST"); ast != "" {
		if v, err := strconv.Atoi(ast); err == nil {
			stats.Assists = v
		}
	}
	if stl := getStat("steals", "STL"); stl != "" {
		if v, err := strconv.Atoi(stl); err == nil {
			stats.Steals = v
		}
	}
	if blk := getStat("blocks", "BLK"); blk != "" {
		if v, err := strconv.Atoi(blk); err == nil {
			stats.Blocks = v
		}
	}
	if to := getStat("turnovers", "TO"); to != "" {
		if v, err := strconv.Atoi(to); err == nil {
			stats.Turnovers = v
		}
	}
	if pf := getStat("fouls", "PF"); pf != "" {
		if v, err := strconv.Atoi(pf); err == nil {
			stats.PersonalFouls = v
		}
	}

	// Calculate points from scoring if not directly available
	// Points = (FGM * 2) + (3PM * 3) + FTM - but we need to subtract the 3PM from FGM first
	// Actually: Points = ((FGM - 3PM) * 2) + (3PM * 3) + FTM
	if stats.FieldGoalsMade > 0 {
		twoPointMade := stats.FieldGoalsMade - stats.ThreePointersMade
		stats.Points = (twoPointMade * 2) + (stats.ThreePointersMade * 3) + stats.FreeThrowsMade
	}

	return &ParsedTeamStats{
		Stats:    stats,
		TeamAbbr: teamAbbr,
	}, nil
}

func parseTeamStatsDetailed(teamData map[string]interface{}, teamAbbr string, statIndexMap map[string]int) (*ParsedTeamStats, error) {
	stats := &store.TeamGameStats{
		// GameID and TeamID will be set by ingester
		GameID: -1,
		TeamID: -1,
	}

	statValues := extractArray(teamData, "stats")
	if len(statValues) == 0 {
		return nil, fmt.Errorf("no stats array for team")
	}

	// Helper to safely get stat by label
	getStat := func(label string) string {
		if idx, ok := statIndexMap[label]; ok && idx < len(statValues) {
			return fmt.Sprint(statValues[idx])
		}
		return ""
	}

	// Parse basic stats
	if pts := getStat("PTS"); pts != "" {
		if v, err := strconv.Atoi(pts); err == nil {
			stats.Points = v
		}
	}

	// Field Goals
	if fgStr := getStat("FG"); fgStr != "" {
		parts := strings.Split(fgStr, "-")
		if len(parts) == 2 {
			if made, err := strconv.Atoi(parts[0]); err == nil {
				stats.FieldGoalsMade = made
			}
			if att, err := strconv.Atoi(parts[1]); err == nil {
				stats.FieldGoalsAttempted = att
			}
		}
	}

	// 3-Pointers
	if tpStr := getStat("3PT"); tpStr != "" {
		parts := strings.Split(tpStr, "-")
		if len(parts) == 2 {
			if made, err := strconv.Atoi(parts[0]); err == nil {
				stats.ThreePointersMade = made
			}
			if att, err := strconv.Atoi(parts[1]); err == nil {
				stats.ThreePointersAttempted = att
			}
		}
	}

	// Free Throws
	if ftStr := getStat("FT"); ftStr != "" {
		parts := strings.Split(ftStr, "-")
		if len(parts) == 2 {
			if made, err := strconv.Atoi(parts[0]); err == nil {
				stats.FreeThrowsMade = made
			}
			if att, err := strconv.Atoi(parts[1]); err == nil {
				stats.FreeThrowsAttempted = att
			}
		}
	}

	// Rebounds
	if reb := getStat("REB"); reb != "" {
		if v, err := strconv.Atoi(reb); err == nil {
			stats.Rebounds = v
		}
	}
	if oreb := getStat("OREB"); oreb != "" {
		if v, err := strconv.Atoi(oreb); err == nil {
			stats.OffensiveRebounds = v
		}
	}
	if dreb := getStat("DREB"); dreb != "" {
		if v, err := strconv.Atoi(dreb); err == nil {
			stats.DefensiveRebounds = v
		}
	}

	// Other stats
	if ast := getStat("AST"); ast != "" {
		if v, err := strconv.Atoi(ast); err == nil {
			stats.Assists = v
		}
	}
	if stl := getStat("STL"); stl != "" {
		if v, err := strconv.Atoi(stl); err == nil {
			stats.Steals = v
		}
	}
	if blk := getStat("BLK"); blk != "" {
		if v, err := strconv.Atoi(blk); err == nil {
			stats.Blocks = v
		}
	}
	if to := getStat("TO"); to != "" {
		if v, err := strconv.Atoi(to); err == nil {
			stats.Turnovers = v
		}
	}
	if pf := getStat("PF"); pf != "" {
		if v, err := strconv.Atoi(pf); err == nil {
			stats.PersonalFouls = v
		}
	}

	return &ParsedTeamStats{
		Stats:    stats,
		TeamAbbr: teamAbbr,
	}, nil
}

// calculateTeamTotalsFromPlayers sums up all player stats to get team totals
func calculateTeamTotalsFromPlayers(athletes []interface{}, statIndexMap map[string]int) map[string]interface{} {
	totals := make(map[string]interface{})
	statValues := make([]interface{}, len(statIndexMap))
	
	// Initialize all stats to 0
	for i := range statValues {
		statValues[i] = "0"
	}
	
	// Sum up all player stats
	for _, athleteInterface := range athletes {
		athleteData := athleteInterface.(map[string]interface{})
		
		// Skip if this is already a totals row
		athlete := extractMap(athleteData, "athlete")
		athleteName := extractString(athlete, "displayName")
		if strings.Contains(strings.ToLower(athleteName), "total") || 
		   strings.Contains(strings.ToLower(athleteName), "team") {
			continue
		}
		
		playerStats := extractArray(athleteData, "stats")
		if len(playerStats) == 0 {
			continue
		}
		
		// Add each stat
		for statName, idx := range statIndexMap {
			if idx >= len(playerStats) || idx >= len(statValues) {
				continue
			}
			
			// For counting stats (not percentages), sum them up
			if !strings.Contains(statName, "%") && !strings.Contains(statName, "PCT") {
				currentVal := parseStatValue(fmt.Sprint(statValues[idx]))
				playerVal := parseStatValue(fmt.Sprint(playerStats[idx]))
				statValues[idx] = fmt.Sprint(currentVal + playerVal)
			}
		}
	}
	
	totals["stats"] = statValues
	return totals
}

// parseStatValue extracts numeric value from stat strings like "5-10" or "15"
func parseStatValue(s string) int {
	// Handle "made-attempted" format
	if strings.Contains(s, "-") {
		parts := strings.Split(s, "-")
		if len(parts) > 0 {
			if v, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil {
				return v
			}
		}
		return 0
	}
	
	// Handle simple number
	if v, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
		return v
	}
	
	return 0
}
