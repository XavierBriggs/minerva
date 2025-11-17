package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/fortuna/minerva/internal/store"
)

// PlayerRepository handles player data access
type PlayerRepository struct {
	db *store.Database
}

// NewPlayerRepository creates a new player repository
func NewPlayerRepository(db *store.Database) *PlayerRepository {
	return &PlayerRepository{db: db}
}

// GetByID finds a player by ID
func (r *PlayerRepository) GetByID(ctx context.Context, playerID int) (*store.Player, error) {
	query := `
		SELECT player_id, sport, external_id, first_name, last_name, full_name, display_name,
			birth_date, birth_city, birth_country, nationality,
			height, height_inches, weight, position, college, high_school,
			draft_year, draft_round, draft_pick, draft_team_id,
			headshot_url, jersey_number, status, metadata,
			created_at, updated_at
		FROM players
		WHERE player_id = $1
	`

	player := &store.Player{}
	err := r.db.DB().QueryRowContext(ctx, query, playerID).Scan(
		&player.PlayerID, &player.Sport, &player.ExternalID, &player.FirstName, &player.LastName,
		&player.FullName, &player.DisplayName, &player.BirthDate, &player.BirthCity, &player.BirthCountry,
		&player.Nationality, &player.Height, &player.HeightInches, &player.Weight, &player.Position,
		&player.College, &player.HighSchool, &player.DraftYear, &player.DraftRound, &player.DraftPick,
		&player.DraftTeamID, &player.HeadshotURL, &player.JerseyNumber, &player.Status, &player.Metadata,
		&player.CreatedAt, &player.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("player not found: %d", playerID)
	}
	if err != nil {
		return nil, fmt.Errorf("querying player: %w", err)
	}

	return player, nil
}

// GetByExternalID finds a player by external ID (e.g., ESPN ID)
func (r *PlayerRepository) GetByExternalID(ctx context.Context, externalID string) (*store.Player, error) {
	query := `
		SELECT player_id, sport, external_id, first_name, last_name, full_name, display_name,
			birth_date, birth_city, birth_country, nationality,
			height, height_inches, weight, position, college, high_school,
			draft_year, draft_round, draft_pick, draft_team_id,
			headshot_url, jersey_number, status, metadata,
			created_at, updated_at
		FROM players
		WHERE external_id = $1
	`

	player := &store.Player{}
	err := r.db.DB().QueryRowContext(ctx, query, externalID).Scan(
		&player.PlayerID, &player.Sport, &player.ExternalID, &player.FirstName, &player.LastName,
		&player.FullName, &player.DisplayName, &player.BirthDate, &player.BirthCity, &player.BirthCountry,
		&player.Nationality, &player.Height, &player.HeightInches, &player.Weight, &player.Position,
		&player.College, &player.HighSchool, &player.DraftYear, &player.DraftRound, &player.DraftPick,
		&player.DraftTeamID, &player.HeadshotURL, &player.JerseyNumber, &player.Status, &player.Metadata,
		&player.CreatedAt, &player.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("player not found: %s", externalID)
	}
	if err != nil {
		return nil, fmt.Errorf("querying player: %w", err)
	}

	return player, nil
}

// GetByName searches for players by name (case-insensitive partial match)
func (r *PlayerRepository) GetByName(ctx context.Context, name string) ([]*store.Player, error) {
	query := `
		SELECT player_id, sport, external_id, first_name, last_name, full_name, display_name,
			birth_date, birth_city, birth_country, nationality,
			height, height_inches, weight, position, college, high_school,
			draft_year, draft_round, draft_pick, draft_team_id,
			headshot_url, jersey_number, status, metadata,
			created_at, updated_at
		FROM players
		WHERE full_name ILIKE $1 OR display_name ILIKE $1
		ORDER BY full_name
		LIMIT 50
	`

	rows, err := r.db.DB().QueryContext(ctx, query, "%"+name+"%")
	if err != nil {
		return nil, fmt.Errorf("querying players: %w", err)
	}
	defer rows.Close()

	return r.scanPlayers(rows)
}

// GetAll returns all players
func (r *PlayerRepository) GetAll(ctx context.Context) ([]*store.Player, error) {
	query := `
		SELECT player_id, sport, external_id, first_name, last_name, full_name, display_name,
			birth_date, birth_city, birth_country, nationality,
			height, height_inches, weight, position, college, high_school,
			draft_year, draft_round, draft_pick, draft_team_id,
			headshot_url, jersey_number, status, metadata,
			created_at, updated_at
		FROM players
		ORDER BY full_name
	`

	rows, err := r.db.DB().QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying players: %w", err)
	}
	defer rows.Close()

	return r.scanPlayers(rows)
}

// Upsert inserts or updates a player
func (r *PlayerRepository) Upsert(ctx context.Context, player *store.Player) error {
	query := `
		INSERT INTO players (sport, external_id, first_name, last_name, full_name, display_name,
			birth_date, birth_city, birth_country, nationality,
			height, height_inches, weight, position, college, high_school,
			draft_year, draft_round, draft_pick, draft_team_id,
			headshot_url, jersey_number, status, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)
		ON CONFLICT (sport, external_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			full_name = EXCLUDED.full_name,
			display_name = EXCLUDED.display_name,
			birth_date = EXCLUDED.birth_date,
			birth_city = EXCLUDED.birth_city,
			birth_country = EXCLUDED.birth_country,
			nationality = EXCLUDED.nationality,
			height = EXCLUDED.height,
			height_inches = EXCLUDED.height_inches,
			weight = EXCLUDED.weight,
			position = EXCLUDED.position,
			college = EXCLUDED.college,
			high_school = EXCLUDED.high_school,
			draft_year = EXCLUDED.draft_year,
			draft_round = EXCLUDED.draft_round,
			draft_pick = EXCLUDED.draft_pick,
			draft_team_id = EXCLUDED.draft_team_id,
			headshot_url = EXCLUDED.headshot_url,
			jersey_number = EXCLUDED.jersey_number,
			status = EXCLUDED.status,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
		RETURNING player_id
	`

	err := r.db.DB().QueryRowContext(ctx, query,
		player.Sport, player.ExternalID, player.FirstName, player.LastName, player.FullName,
		player.DisplayName, player.BirthDate, player.BirthCity, player.BirthCountry, player.Nationality,
		player.Height, player.HeightInches, player.Weight, player.Position, player.College,
		player.HighSchool, player.DraftYear, player.DraftRound, player.DraftPick, player.DraftTeamID,
		player.HeadshotURL, player.JerseyNumber, player.Status, player.Metadata,
	).Scan(&player.PlayerID)

	if err != nil {
		return fmt.Errorf("upserting player: %w", err)
	}

	return nil
}

// GetCurrentTeamID returns the current team ID for a player from player_team_history
func (r *PlayerRepository) GetCurrentTeamID(ctx context.Context, playerID int) (int, error) {
	query := `
		SELECT team_id
		FROM player_team_history
		WHERE player_id = $1
		  AND (end_date IS NULL OR end_date > NOW())
		ORDER BY start_date DESC
		LIMIT 1
	`

	var teamID int
	err := r.db.DB().QueryRowContext(ctx, query, playerID).Scan(&teamID)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("no current team found for player %d", playerID)
	}
	if err != nil {
		return 0, fmt.Errorf("querying current team: %w", err)
	}

	return teamID, nil
}

// GetByCurrentTeam returns all players currently on a team
func (r *PlayerRepository) GetByCurrentTeam(ctx context.Context, teamID int) ([]*store.Player, error) {
	query := `
		SELECT DISTINCT p.player_id, p.sport, p.external_id, p.first_name, p.last_name, p.full_name, p.display_name,
			p.birth_date, p.birth_city, p.birth_country, p.nationality,
			p.height, p.height_inches, p.weight, p.position, p.college, p.high_school,
			p.draft_year, p.draft_round, p.draft_pick, p.draft_team_id,
			p.headshot_url, p.jersey_number, p.status, p.metadata,
			p.created_at, p.updated_at
		FROM players p
		INNER JOIN player_team_history pth ON p.player_id = pth.player_id
		WHERE pth.team_id = $1
		  AND (pth.end_date IS NULL OR pth.end_date > NOW())
		ORDER BY p.full_name
	`

	rows, err := r.db.DB().QueryContext(ctx, query, teamID)
	if err != nil {
		return nil, fmt.Errorf("querying players by team: %w", err)
	}
	defer rows.Close()

	return r.scanPlayers(rows)
}

// scanPlayers is a helper to scan multiple player rows
func (r *PlayerRepository) scanPlayers(rows *sql.Rows) ([]*store.Player, error) {
	var players []*store.Player
	for rows.Next() {
		player := &store.Player{}
		err := rows.Scan(
			&player.PlayerID, &player.Sport, &player.ExternalID, &player.FirstName, &player.LastName,
			&player.FullName, &player.DisplayName, &player.BirthDate, &player.BirthCity, &player.BirthCountry,
			&player.Nationality, &player.Height, &player.HeightInches, &player.Weight, &player.Position,
			&player.College, &player.HighSchool, &player.DraftYear, &player.DraftRound, &player.DraftPick,
			&player.DraftTeamID, &player.HeadshotURL, &player.JerseyNumber, &player.Status, &player.Metadata,
			&player.CreatedAt, &player.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning player: %w", err)
		}
		players = append(players, player)
	}

	return players, rows.Err()
}
