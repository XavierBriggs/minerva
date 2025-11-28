package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/fortuna/minerva/internal/store"
)

// GameRepository handles game data access
type GameRepository struct {
	db *store.Database
}

// NewGameRepository creates a new game repository
func NewGameRepository(db *store.Database) *GameRepository {
	return &GameRepository{db: db}
}

// GetByID finds a game by ID
// GetByID finds a game by its database ID (integer)
func (r *GameRepository) GetByID(ctx context.Context, gameID int) (*store.Game, error) {
	query := `
		SELECT game_id, sport, season_id, external_id, game_date, game_time,
			home_team_id, away_team_id, home_score, away_score, status,
			period, clock, venue, attendance, metadata, created_at, updated_at
		FROM games
		WHERE game_id = $1
	`

	game := &store.Game{}
	err := r.db.DB().QueryRowContext(ctx, query, gameID).Scan(
		&game.GameID, &game.Sport, &game.SeasonID, &game.ExternalID, &game.GameDate, &game.GameTime,
		&game.HomeTeamID, &game.AwayTeamID, &game.HomeScore, &game.AwayScore, &game.Status,
		&game.Period, &game.Clock, &game.Venue, &game.Attendance, &game.Metadata,
		&game.CreatedAt, &game.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("game not found: %d", gameID)
	}
	if err != nil {
		return nil, fmt.Errorf("querying game: %w", err)
	}

	return game, nil
}

// GetByExternalID finds a game by its external ID (ESPN ID)
func (r *GameRepository) GetByExternalID(ctx context.Context, externalID string) (*store.Game, error) {
	query := `
		SELECT game_id, sport, season_id, external_id, game_date, game_time,
			home_team_id, away_team_id, home_score, away_score, status,
			period, clock, venue, attendance, metadata, created_at, updated_at
		FROM games
		WHERE external_id = $1
	`

	game := &store.Game{}
	err := r.db.DB().QueryRowContext(ctx, query, externalID).Scan(
		&game.GameID, &game.Sport, &game.SeasonID, &game.ExternalID, &game.GameDate, &game.GameTime,
		&game.HomeTeamID, &game.AwayTeamID, &game.HomeScore, &game.AwayScore, &game.Status,
		&game.Period, &game.Clock, &game.Venue, &game.Attendance, &game.Metadata,
		&game.CreatedAt, &game.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("game not found: %s", externalID)
	}
	if err != nil {
		return nil, fmt.Errorf("querying game: %w", err)
	}

	return game, nil
}

// GetByDate returns all games on a specific date
func (r *GameRepository) GetByDate(ctx context.Context, date time.Time) ([]*store.Game, error) {
	// Truncate to start of day and get the next day
	startOfDay := date.Truncate(24 * time.Hour)
	endOfDay := startOfDay.Add(24 * time.Hour)

	query := `
		SELECT game_id, sport, season_id, external_id, game_date, game_time,
			home_team_id, away_team_id, home_score, away_score, status,
			period, clock, venue, attendance, metadata, created_at, updated_at
		FROM games
		WHERE game_date >= $1 AND game_date < $2
		ORDER BY game_time
	`

	rows, err := r.db.DB().QueryContext(ctx, query, startOfDay, endOfDay)
	if err != nil {
		return nil, fmt.Errorf("querying games: %w", err)
	}
	defer rows.Close()

	return r.scanGames(rows)
}

// GetLiveGames returns all currently live games
// Only returns games from today (EST) to avoid stale data
func (r *GameRepository) GetLiveGames(ctx context.Context) ([]*store.Game, error) {
	// Get today's date range in EST
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		loc = time.UTC
	}
	nowEST := time.Now().In(loc)
	startOfDay := time.Date(nowEST.Year(), nowEST.Month(), nowEST.Day(), 0, 0, 0, 0, loc)
	endOfDay := startOfDay.Add(24 * time.Hour)

	query := `
		SELECT game_id, sport, season_id, external_id, game_date, game_time,
			home_team_id, away_team_id, home_score, away_score, status,
			period, clock, venue, attendance, metadata, created_at, updated_at
		FROM games
		WHERE status = 'in_progress' 
			AND game_date >= $1 AND game_date < $2
		ORDER BY updated_at DESC
	`

	rows, err := r.db.DB().QueryContext(ctx, query, startOfDay, endOfDay)
	if err != nil {
		return nil, fmt.Errorf("querying live games: %w", err)
	}
	defer rows.Close()

	return r.scanGames(rows)
}

// GetTodaysGames returns all games scheduled for today (any status)
// Uses Eastern Time since NBA games are scheduled in EST
func (r *GameRepository) GetTodaysGames(ctx context.Context) ([]*store.Game, error) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		loc = time.UTC
	}
	nowEST := time.Now().In(loc)
	startOfDay := time.Date(nowEST.Year(), nowEST.Month(), nowEST.Day(), 0, 0, 0, 0, loc)
	endOfDay := startOfDay.Add(24 * time.Hour)

	query := `
		SELECT game_id, sport, season_id, external_id, game_date, game_time,
			home_team_id, away_team_id, home_score, away_score, status,
			period, clock, venue, attendance, metadata, created_at, updated_at
		FROM games
		WHERE game_date >= $1 AND game_date < $2
		ORDER BY 
			CASE status 
				WHEN 'in_progress' THEN 1 
				WHEN 'scheduled' THEN 2 
				WHEN 'final' THEN 3 
				ELSE 4 
			END,
			game_time
	`

	rows, err := r.db.DB().QueryContext(ctx, query, startOfDay, endOfDay)
	if err != nil {
		return nil, fmt.Errorf("querying today's games: %w", err)
	}
	defer rows.Close()

	return r.scanGames(rows)
}

// GetUpcomingGames returns upcoming scheduled games
// Uses Eastern Time (America/New_York) since NBA games are scheduled in EST
func (r *GameRepository) GetUpcomingGames(ctx context.Context, limit int) ([]*store.Game, error) {
	// Get current date in EST for proper comparison
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		loc = time.UTC // Fallback to UTC if timezone not available
	}
	nowEST := time.Now().In(loc)
	todayEST := nowEST.Truncate(24 * time.Hour)

	query := `
		SELECT game_id, sport, season_id, external_id, game_date, game_time,
			home_team_id, away_team_id, home_score, away_score, status,
			period, clock, venue, attendance, metadata, created_at, updated_at
		FROM games
		WHERE status = 'scheduled' AND game_date >= $1
		ORDER BY game_date, game_time
		LIMIT $2
	`

	rows, err := r.db.DB().QueryContext(ctx, query, todayEST, limit)
	if err != nil {
		return nil, fmt.Errorf("querying upcoming games: %w", err)
	}
	defer rows.Close()

	return r.scanGames(rows)
}

// GetByTeam returns games for a specific team
func (r *GameRepository) GetByTeam(ctx context.Context, teamID int, seasonID int, limit int) ([]*store.Game, error) {
	query := `
		SELECT game_id, sport, season_id, external_id, game_date, game_time,
			home_team_id, away_team_id, home_score, away_score, status,
			period, clock, venue, attendance, metadata, created_at, updated_at
		FROM games
		WHERE (home_team_id = $1 OR away_team_id = $1)
			AND season_id = $2
		ORDER BY game_date DESC
		LIMIT $3
	`

	rows, err := r.db.DB().QueryContext(ctx, query, teamID, seasonID, limit)
	if err != nil {
		return nil, fmt.Errorf("querying team games: %w", err)
	}
	defer rows.Close()

	return r.scanGames(rows)
}

// GetBySeason returns all games in a season
func (r *GameRepository) GetBySeason(ctx context.Context, seasonID int) ([]*store.Game, error) {
	query := `
		SELECT game_id, sport, season_id, external_id, game_date, game_time,
			home_team_id, away_team_id, home_score, away_score, status,
			period, clock, venue, attendance, metadata, created_at, updated_at
		FROM games
		WHERE season_id = $1
		ORDER BY game_date, game_time
	`

	rows, err := r.db.DB().QueryContext(ctx, query, seasonID)
	if err != nil {
		return nil, fmt.Errorf("querying season games: %w", err)
	}
	defer rows.Close()

	return r.scanGames(rows)
}

// Upsert inserts or updates a game
func (r *GameRepository) Upsert(ctx context.Context, game *store.Game) error {
	query := `
		INSERT INTO games (sport, season_id, external_id, game_date, game_time,
			home_team_id, away_team_id, home_score, away_score, status,
			period, clock, venue, attendance, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (sport, external_id) DO UPDATE SET
			game_date = EXCLUDED.game_date,
			game_time = EXCLUDED.game_time,
			home_team_id = EXCLUDED.home_team_id,
			away_team_id = EXCLUDED.away_team_id,
			home_score = EXCLUDED.home_score,
			away_score = EXCLUDED.away_score,
			status = EXCLUDED.status,
			period = EXCLUDED.period,
			clock = EXCLUDED.clock,
			venue = EXCLUDED.venue,
			attendance = EXCLUDED.attendance,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
		RETURNING game_id
	`

	err := r.db.DB().QueryRowContext(ctx, query,
		game.Sport, game.SeasonID, game.ExternalID, game.GameDate, game.GameTime,
		game.HomeTeamID, game.AwayTeamID, game.HomeScore, game.AwayScore, game.Status,
		game.Period, game.Clock, game.Venue, game.Attendance, game.Metadata,
	).Scan(&game.GameID)

	if err != nil {
		return fmt.Errorf("upserting game: %w", err)
	}

	return nil
}

// CleanupStaleGames marks games older than 6 hours with "in_progress" status as "final"
// This fixes stuck games that never had their status updated
func (r *GameRepository) CleanupStaleGames(ctx context.Context) (int64, error) {
	// Any game that started more than 6 hours ago and is still "in_progress" is almost certainly finished
	// NBA games typically last about 2.5 hours
	staleThreshold := time.Now().Add(-6 * time.Hour)

	query := `
		UPDATE games 
		SET status = 'final', updated_at = NOW()
		WHERE status = 'in_progress' 
			AND game_time < $1
	`

	result, err := r.db.DB().ExecContext(ctx, query, staleThreshold)
	if err != nil {
		return 0, fmt.Errorf("cleaning up stale games: %w", err)
	}

	return result.RowsAffected()
}

// scanGames scans multiple game rows
func (r *GameRepository) scanGames(rows *sql.Rows) ([]*store.Game, error) {
	var games []*store.Game
	for rows.Next() {
		game := &store.Game{}
		err := rows.Scan(
			&game.GameID, &game.Sport, &game.SeasonID, &game.ExternalID, &game.GameDate, &game.GameTime,
			&game.HomeTeamID, &game.AwayTeamID, &game.HomeScore, &game.AwayScore, &game.Status,
			&game.Period, &game.Clock, &game.Venue, &game.Attendance, &game.Metadata,
			&game.CreatedAt, &game.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning game: %w", err)
		}
		games = append(games, game)
	}

	return games, rows.Err()
}
