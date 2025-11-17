package espn

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/fortuna/minerva/internal/store"
	"github.com/fortuna/minerva/internal/store/repository"
)

// Ingester handles the ingestion of ESPN data into the database.
type Ingester struct {
	client    *Client
	db        *store.Database
	gameRepo  *repository.GameRepository
	statsRepo *repository.StatsRepository
	teamRepo  *repository.TeamRepository
	playerRepo *repository.PlayerRepository

	mu        sync.Mutex
	teamCache *teamLookup
	playerIDs sync.Map // espn_player_id -> int
}

type teamLookup struct {
	byAbbr map[string]int
	byESPN map[string]int
}

// NewIngester creates a new ESPN data ingester using the default API base.
func NewIngester(db *store.Database) *Ingester {
	return NewIngesterWithBaseURL(db, "")
}

// NewIngesterWithBaseURL creates an ingester overriding the ESPN base URL.
func NewIngesterWithBaseURL(db *store.Database, baseURL string) *Ingester {
	var client *Client
	if strings.TrimSpace(baseURL) != "" {
		log.Printf("[ingester] Creating ESPN client with baseURL: %s", baseURL)
		client = New(baseURL)
	} else {
		log.Printf("[ingester] Creating ESPN client with default baseURL")
		client = NewClient()
	}

	return &Ingester{
		client:     client,
		db:         db,
		gameRepo:   repository.NewGameRepository(db),
		statsRepo:  repository.NewStatsRepository(db),
		teamRepo:   repository.NewTeamRepository(db),
		playerRepo: repository.NewPlayerRepository(db),
	}
}

// IngestTodaysGames fetches and stores games for the current day.
// Uses Eastern Time (America/New_York) since NBA games are scheduled in US timezones.
func (i *Ingester) IngestTodaysGames(ctx context.Context, seasonID int) error {
	// Load Eastern Time location
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		log.Printf("Warning: Failed to load America/New_York timezone, falling back to UTC: %v", err)
		loc = time.UTC
	}
	
	// Get current time in Eastern Time
	now := time.Now().In(loc)
	_, err = i.IngestGamesByDate(ctx, seasonID, now)
	return err
}

// IngestGamesByDate fetches and stores games (and stats) for a specific date.
func (i *Ingester) IngestGamesByDate(ctx context.Context, seasonID int, date time.Time) ([]*store.Game, error) {
	log.Printf("[ingest] Fetching scoreboard for %s", date.Format("2006-01-02"))

	if err := i.ensureTeamLookup(ctx); err != nil {
		return nil, err
	}

	scoreboard, err := i.client.FetchScoreboard(ctx, BasketballNBA, date)
	if err != nil {
		return nil, fmt.Errorf("fetch scoreboard: %w", err)
	}

	parsedGames, err := ParseScoreboardGamesDetailed(scoreboard, seasonID)
	if err != nil {
		return nil, fmt.Errorf("parse scoreboard: %w", err)
	}

	var ingested []*store.Game
	for _, parsed := range parsedGames {
		game, err := i.persistParsedGame(ctx, parsed)
		if err != nil {
			log.Printf("[ingest] Error upserting game %d: %v", parsed.Game.GameID, err)
			continue
		}

		if err := i.ingestStatsForGameByID(ctx, game.GameID, parsed.Game.ExternalID); err != nil {
			log.Printf("[ingest] Error ingesting stats for game %d (ESPN ID %s): %v", game.GameID, parsed.Game.ExternalID, err)
		}

		ingested = append(ingested, game)
	}

	log.Printf("[ingest] ✓ Processed %d games for %s", len(ingested), date.Format("2006-01-02"))
	return ingested, nil
}

// IngestGameByID fetches and stores a single game by ESPN event ID.
func (i *Ingester) IngestGameByID(ctx context.Context, seasonID int, gameID string) (*store.Game, error) {
	if err := i.ensureTeamLookup(ctx); err != nil {
		return nil, err
	}

	summary, err := i.client.FetchGameSummary(ctx, BasketballNBA, gameID)
	if err != nil {
		return nil, fmt.Errorf("fetch game summary: %w", err)
	}

	event := buildEventFromSummary(summary)
	if event == nil {
		return nil, fmt.Errorf("summary missing header data for event %s", gameID)
	}

	parsed, err := parseGameFromEventDetailed(event, seasonID)
	if err != nil {
		return nil, err
	}

	game, err := i.persistParsedGame(ctx, parsed)
	if err != nil {
		return nil, err
	}

	if err := i.ingestStatsForGameByID(ctx, game.GameID, parsed.Game.ExternalID); err != nil {
		return nil, err
	}

	return game, nil
}

func (i *Ingester) ingestStatsForGameByID(ctx context.Context, dbGameID int, espnGameID string) error {
	summary, err := i.client.FetchGameSummary(ctx, BasketballNBA, espnGameID)
	if err != nil {
		return fmt.Errorf("fetch game summary: %w", err)
	}
	return i.ingestStatsFromSummary(ctx, dbGameID, espnGameID, summary)
}

func (i *Ingester) ingestStatsFromSummary(ctx context.Context, dbGameID int, espnGameID string, summary map[string]interface{}) error {
	parsedStats, err := ParseBoxScoreDetailed(summary, espnGameID)
	if err != nil {
		return fmt.Errorf("parse box score: %w", err)
	}

	for _, parsed := range parsedStats {
		teamID, err := i.lookupTeamID(parsed.TeamAbbr, "")
		if err != nil {
			log.Printf("[ingest] Unknown team %s for player %s", parsed.TeamAbbr, parsed.PlayerName)
			continue
		}

		playerID, err := i.resolvePlayerID(ctx, parsed, teamID)
		if err != nil {
			log.Printf("[ingest] Unable to resolve player %s: %v", parsed.PlayerName, err)
			continue
		}

		stats := parsed.Stats
		stats.GameID = dbGameID
		stats.TeamID = teamID
		stats.PlayerID = playerID

		if err := i.statsRepo.UpsertPlayerStats(ctx, stats); err != nil {
			log.Printf("[ingest] Failed to upsert stats for player %d in game %d: %v", playerID, dbGameID, err)
		}
	}

	// Ingest team stats
	if err := i.ingestTeamStatsFromSummary(ctx, dbGameID, espnGameID, summary); err != nil {
		log.Printf("[ingest] Failed to ingest team stats for game %d: %v", dbGameID, err)
		// Don't return error - team stats are supplementary
	}

	return nil
}

func (i *Ingester) ingestTeamStatsFromSummary(ctx context.Context, dbGameID int, espnGameID string, summary map[string]interface{}) error {
	parsedTeamStats, err := ParseTeamStats(summary, espnGameID)
	if err != nil {
		return fmt.Errorf("parse team stats: %w", err)
	}

	// Get game to determine home/away
	game, err := i.gameRepo.GetByID(ctx, dbGameID)
	if err != nil {
		return fmt.Errorf("fetch game: %w", err)
	}

	for _, parsed := range parsedTeamStats {
		teamID, err := i.lookupTeamID(parsed.TeamAbbr, "")
		if err != nil {
			log.Printf("[ingest] Unknown team %s for team stats", parsed.TeamAbbr)
			continue
		}

		stats := parsed.Stats
		stats.GameID = dbGameID
		stats.TeamID = teamID
		stats.IsHome = (teamID == game.HomeTeamID)

		if err := i.statsRepo.UpsertTeamStats(ctx, stats); err != nil {
			log.Printf("[ingest] Failed to upsert team stats for team %d in game %d: %v", teamID, dbGameID, err)
		}
	}

	return nil
}

func (i *Ingester) persistParsedGame(ctx context.Context, parsed *ParsedGame) (*store.Game, error) {
	homeID, err := i.lookupTeamID(parsed.HomeTeam.Abbreviation, parsed.HomeTeam.ESPNID)
	if err != nil {
		return nil, fmt.Errorf("lookup home team: %w", err)
	}
	awayID, err := i.lookupTeamID(parsed.AwayTeam.Abbreviation, parsed.AwayTeam.ESPNID)
	if err != nil {
		return nil, fmt.Errorf("lookup away team: %w", err)
	}

	parsed.Game.HomeTeamID = homeID
	parsed.Game.AwayTeamID = awayID

	// SeasonType is no longer a field in the Game struct (v2 schema)
	// Season type is managed through the seasons table

	if err := i.gameRepo.Upsert(ctx, parsed.Game); err != nil {
		return nil, err
	}

	return parsed.Game, nil
}

func (i *Ingester) lookupTeamID(abbr string, espnID string) (int, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.teamCache == nil {
		return 0, fmt.Errorf("team cache not initialized")
	}

	if espnID != "" {
		if id, ok := i.teamCache.byESPN[espnID]; ok {
			return id, nil
		}
	}

	if abbr != "" {
		// Normalize the abbreviation
		normalizedAbbr := normalizeTeamAbbreviation(abbr)
		if id, ok := i.teamCache.byAbbr[strings.ToUpper(normalizedAbbr)]; ok {
			return id, nil
		}
	}

	return 0, fmt.Errorf("team not found (abbr=%s espn=%s)", abbr, espnID)
}

// normalizeTeamAbbreviation handles ESPN's inconsistent abbreviations
func normalizeTeamAbbreviation(abbr string) string {
	abbr = strings.ToUpper(strings.TrimSpace(abbr))
	
	// ESPN sometimes uses shortened versions of team abbreviations
	// Map them to our database's standard abbreviations
	abbreviationMap := map[string]string{
		"GS":   "GSW",  // Golden State Warriors
		"SA":   "SAS",  // San Antonio Spurs
		"NO":   "NOP",  // New Orleans Pelicans
		"NY":   "NYK",  // New York Knicks
		"UTAH": "UTA",  // Utah Jazz
		"WSH":  "WAS",  // Washington Wizards
	}
	
	if normalized, ok := abbreviationMap[abbr]; ok {
		return normalized
	}
	
	return abbr
}

func (i *Ingester) ensureTeamLookup(ctx context.Context) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.teamCache != nil {
		return nil
	}

	teams, err := i.teamRepo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("load teams: %w", err)
	}

	lookup := &teamLookup{
		byAbbr: make(map[string]int),
		byESPN: make(map[string]int),
	}

	for _, team := range teams {
		lookup.byAbbr[strings.ToUpper(team.Abbreviation)] = team.TeamID
		if team.ExternalID != "" {
			lookup.byESPN[team.ExternalID] = team.TeamID
		}
	}

	i.teamCache = lookup
	return nil
}

func (i *Ingester) resolvePlayerID(ctx context.Context, parsed *ParsedPlayerStats, teamID int) (int, error) {
	if parsed.ESPNPlayerID != "" {
		if cached, ok := i.playerIDs.Load(parsed.ESPNPlayerID); ok {
			return cached.(int), nil
		}

		if player, err := i.playerRepo.GetByExternalID(ctx, parsed.ESPNPlayerID); err == nil {
			i.playerIDs.Store(parsed.ESPNPlayerID, player.PlayerID)
			return player.PlayerID, nil
		}
	}

	// Parse name into first/last (simple split on last space)
	firstName, lastName := "", parsed.PlayerName
	if idx := len(parsed.PlayerName) - 1; idx > 0 {
		for i := idx; i >= 0; i-- {
			if parsed.PlayerName[i] == ' ' {
				firstName = parsed.PlayerName[:i]
				lastName = parsed.PlayerName[i+1:]
				break
			}
		}
	}

	player := &store.Player{
		Sport:        "basketball_nba",
		ExternalID:   sql.NullString{String: parsed.ESPNPlayerID, Valid: parsed.ESPNPlayerID != ""},
		FirstName:    sql.NullString{String: firstName, Valid: firstName != ""},
		LastName:     lastName,
		FullName:     parsed.PlayerName,
		DisplayName:  sql.NullString{String: parsed.PlayerName, Valid: true},
		Position:     sql.NullString{String: parsed.Position, Valid: parsed.Position != ""},
		JerseyNumber: sql.NullString{String: parsed.Jersey, Valid: parsed.Jersey != ""},
		Height:       sql.NullString{String: parsed.Height, Valid: parsed.Height != ""},
		Status:       sql.NullString{String: "active", Valid: true},
	}

	if parsed.Weight > 0 {
		player.Weight = sql.NullInt32{Int32: int32(parsed.Weight), Valid: true}
	}

	if parsed.BirthDate != nil {
		player.BirthDate = sql.NullTime{Time: *parsed.BirthDate, Valid: true}
	}

	if err := i.playerRepo.Upsert(ctx, player); err != nil {
		return 0, err
	}

	if parsed.ESPNPlayerID != "" {
		i.playerIDs.Store(parsed.ESPNPlayerID, player.PlayerID)
	}

	return player.PlayerID, nil
}

func buildEventFromSummary(summary map[string]interface{}) map[string]interface{} {
	header := extractMap(summary, "header")
	if len(header) == 0 {
		return nil
	}

	competitions := extractArray(header, "competitions")
	if len(competitions) == 0 {
		return nil
	}

	comp := competitions[0].(map[string]interface{})

	event := map[string]interface{}{
		"id":           extractString(comp, "id"),
		"date":         extractString(comp, "date"),
		"status":       extractMap(comp, "status"),
		"competitions": []interface{}{comp},
		"season":       extractMap(comp, "season"),
	}
	return event
}

