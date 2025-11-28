package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/fortuna/minerva/internal/store"
)

// StatsRepository handles player and team stats data access
type StatsRepository struct {
	db *store.Database
}

// NewStatsRepository creates a new stats repository
func NewStatsRepository(db *store.Database) *StatsRepository {
	return &StatsRepository{db: db}
}

// GetPlayerGameStats returns stats for a player in a specific game
func (r *StatsRepository) GetPlayerGameStats(ctx context.Context, gameID string, playerID int) (*store.PlayerGameStats, error) {
	query := `
		SELECT id, game_id, player_id, team_id, points, rebounds, assists, steals, blocks, turnovers,
			field_goals_made, field_goals_attempted, three_pointers_made, three_pointers_attempted,
			free_throws_made, free_throws_attempted, offensive_rebounds, defensive_rebounds,
			personal_fouls, minutes_played, plus_minus, starter, true_shooting_pct, effective_fg_pct,
			usage_pct, created_at, updated_at
		FROM player_game_stats
		WHERE game_id = $1 AND player_id = $2
	`

	stats := &store.PlayerGameStats{}
	err := r.db.DB().QueryRowContext(ctx, query, gameID, playerID).Scan(
		&stats.ID, &stats.GameID, &stats.PlayerID, &stats.TeamID, &stats.Points, &stats.Rebounds,
		&stats.Assists, &stats.Steals, &stats.Blocks, &stats.Turnovers, &stats.FieldGoalsMade,
		&stats.FieldGoalsAttempted, &stats.ThreePointersMade, &stats.ThreePointersAttempted,
		&stats.FreeThrowsMade, &stats.FreeThrowsAttempted, &stats.OffensiveRebounds,
		&stats.DefensiveRebounds, &stats.PersonalFouls, &stats.MinutesPlayed, &stats.PlusMinus,
		&stats.Starter, &stats.TrueShootingPct, &stats.EffectiveFGPct, &stats.UsageRate,
		&stats.CreatedAt, &stats.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("stats not found for game %s, player %d", gameID, playerID)
	}
	if err != nil {
		return nil, fmt.Errorf("querying player stats: %w", err)
	}

	return stats, nil
}

// GetGameBoxScore returns all player stats for a game
func (r *StatsRepository) GetGameBoxScore(ctx context.Context, gameID string) ([]*store.PlayerGameStats, error) {
	query := `
		SELECT id, game_id, player_id, team_id, points, rebounds, assists, steals, blocks, turnovers,
			field_goals_made, field_goals_attempted, three_pointers_made, three_pointers_attempted,
			free_throws_made, free_throws_attempted, offensive_rebounds, defensive_rebounds,
			personal_fouls, minutes_played, plus_minus, starter, true_shooting_pct, effective_fg_pct,
			usage_pct, created_at, updated_at
		FROM player_game_stats
		WHERE game_id = $1
		ORDER BY starter DESC, minutes_played DESC
	`

	rows, err := r.db.DB().QueryContext(ctx, query, gameID)
	if err != nil {
		return nil, fmt.Errorf("querying box score: %w", err)
	}
	defer rows.Close()

	return r.scanPlayerStats(rows)
}

// EnrichedPlayerStats includes player game stats with game context (date, opponent)
type EnrichedPlayerStats struct {
	*store.PlayerGameStats
	GameDate       string `json:"game_date"`
	OpponentTeamID int    `json:"opponent_team_id"`
	OpponentAbbr   string `json:"opponent_abbr"`
	OpponentName   string `json:"opponent_name"`
	IsHome         bool   `json:"is_home"`
	HomeScore      int    `json:"home_score"`
	AwayScore      int    `json:"away_score"`
	Result         string `json:"result"` // "W" or "L"
}

// GetPlayerRecentStats returns a player's stats for their last N games
func (r *StatsRepository) GetPlayerRecentStats(ctx context.Context, playerID int, limit int) ([]*store.PlayerGameStats, error) {
	query := `
		SELECT pgs.stat_id, pgs.game_id, pgs.player_id, pgs.team_id, pgs.points, pgs.rebounds, pgs.assists,
			pgs.steals, pgs.blocks, pgs.turnovers, pgs.field_goals_made, pgs.field_goals_attempted,
			pgs.three_pointers_made, pgs.three_pointers_attempted, pgs.free_throws_made,
			pgs.free_throws_attempted, pgs.offensive_rebounds, pgs.defensive_rebounds,
			pgs.personal_fouls, pgs.minutes_played, pgs.plus_minus, pgs.starter,
			pgs.true_shooting_pct, pgs.effective_fg_pct, pgs.usage_rate, pgs.created_at, pgs.updated_at
		FROM player_game_stats pgs
		JOIN games g ON pgs.game_id = g.game_id
		WHERE pgs.player_id = $1 AND g.status = 'final'
		ORDER BY g.game_date DESC
		LIMIT $2
	`

	rows, err := r.db.DB().QueryContext(ctx, query, playerID, limit)
	if err != nil {
		return nil, fmt.Errorf("querying recent stats: %w", err)
	}
	defer rows.Close()

	return r.scanPlayerStats(rows)
}

// GetPlayerRecentStatsEnriched returns a player's stats with full game context
func (r *StatsRepository) GetPlayerRecentStatsEnriched(ctx context.Context, playerID int, limit int) ([]*EnrichedPlayerStats, error) {
	query := `
		SELECT 
			pgs.stat_id, pgs.game_id, pgs.player_id, pgs.team_id, pgs.points, pgs.rebounds, pgs.assists,
			pgs.steals, pgs.blocks, pgs.turnovers, pgs.field_goals_made, pgs.field_goals_attempted,
			pgs.three_pointers_made, pgs.three_pointers_attempted, pgs.free_throws_made,
			pgs.free_throws_attempted, pgs.offensive_rebounds, pgs.defensive_rebounds,
			pgs.personal_fouls, pgs.minutes_played, pgs.plus_minus, pgs.starter,
			pgs.true_shooting_pct, pgs.effective_fg_pct, pgs.usage_rate, pgs.created_at, pgs.updated_at,
			g.game_date,
			g.home_team_id, g.away_team_id,
			COALESCE(g.home_score, 0) as home_score,
			COALESCE(g.away_score, 0) as away_score,
			CASE WHEN pgs.team_id = g.home_team_id THEN g.away_team_id ELSE g.home_team_id END as opponent_team_id,
			CASE WHEN pgs.team_id = g.home_team_id THEN true ELSE false END as is_home,
			opp.abbreviation as opponent_abbr,
			opp.full_name as opponent_name
		FROM player_game_stats pgs
		JOIN games g ON pgs.game_id = g.game_id
		LEFT JOIN teams opp ON opp.team_id = CASE WHEN pgs.team_id = g.home_team_id THEN g.away_team_id ELSE g.home_team_id END
		WHERE pgs.player_id = $1 AND g.status = 'final'
		ORDER BY g.game_date DESC
		LIMIT $2
	`

	rows, err := r.db.DB().QueryContext(ctx, query, playerID, limit)
	if err != nil {
		return nil, fmt.Errorf("querying enriched stats: %w", err)
	}
	defer rows.Close()

	var allStats []*EnrichedPlayerStats
	for rows.Next() {
		stats := &store.PlayerGameStats{}
		enriched := &EnrichedPlayerStats{PlayerGameStats: stats}
		var gameDate sql.NullTime
		var homeTeamID, awayTeamID int
		var oppAbbr, oppName sql.NullString

		err := rows.Scan(
			&stats.ID, &stats.GameID, &stats.PlayerID, &stats.TeamID, &stats.Points, &stats.Rebounds,
			&stats.Assists, &stats.Steals, &stats.Blocks, &stats.Turnovers, &stats.FieldGoalsMade,
			&stats.FieldGoalsAttempted, &stats.ThreePointersMade, &stats.ThreePointersAttempted,
			&stats.FreeThrowsMade, &stats.FreeThrowsAttempted, &stats.OffensiveRebounds,
			&stats.DefensiveRebounds, &stats.PersonalFouls, &stats.MinutesPlayed, &stats.PlusMinus,
			&stats.Starter, &stats.TrueShootingPct, &stats.EffectiveFGPct, &stats.UsageRate,
			&stats.CreatedAt, &stats.UpdatedAt,
			&gameDate,
			&homeTeamID, &awayTeamID,
			&enriched.HomeScore, &enriched.AwayScore,
			&enriched.OpponentTeamID, &enriched.IsHome,
			&oppAbbr, &oppName,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning enriched stats: %w", err)
		}

		if gameDate.Valid {
			enriched.GameDate = gameDate.Time.Format("2006-01-02")
		}
		if oppAbbr.Valid {
			enriched.OpponentAbbr = oppAbbr.String
		}
		if oppName.Valid {
			enriched.OpponentName = oppName.String
		}

		// Calculate result
		playerTeamScore := enriched.AwayScore
		oppTeamScore := enriched.HomeScore
		if enriched.IsHome {
			playerTeamScore = enriched.HomeScore
			oppTeamScore = enriched.AwayScore
		}
		if playerTeamScore > oppTeamScore {
			enriched.Result = "W"
		} else {
			enriched.Result = "L"
		}

		allStats = append(allStats, enriched)
	}

	return allStats, rows.Err()
}

// GetPlayerSeasonAverages calculates a player's season averages
// seasonYear is a string like "2024-25" which maps to a season_id in the seasons table
func (r *StatsRepository) GetPlayerSeasonAverages(ctx context.Context, playerID int, seasonYear string) (map[string]float64, error) {
	query := `
		SELECT
			COUNT(*) as games_played,
			AVG(points) as ppg,
			AVG(rebounds) as rpg,
			AVG(assists) as apg,
			AVG(steals) as spg,
			AVG(blocks) as bpg,
			AVG(turnovers) as tpg,
			AVG(minutes_played) as mpg,
			SUM(field_goals_made)::float / NULLIF(SUM(field_goals_attempted), 0) as fg_pct,
			SUM(three_pointers_made)::float / NULLIF(SUM(three_pointers_attempted), 0) as three_pct,
			SUM(free_throws_made)::float / NULLIF(SUM(free_throws_attempted), 0) as ft_pct
		FROM player_game_stats pgs
		JOIN games g ON pgs.game_id = g.game_id
		JOIN seasons s ON g.season_id = s.season_id
		WHERE pgs.player_id = $1 AND s.season_year = $2 AND g.status = 'final'
	`

	var gamesPlayed int
	var ppg, rpg, apg, spg, bpg, tpg, mpg sql.NullFloat64
	var fgPct, threePct, ftPct sql.NullFloat64

	err := r.db.DB().QueryRowContext(ctx, query, playerID, seasonYear).Scan(
		&gamesPlayed, &ppg, &rpg, &apg, &spg, &bpg, &tpg, &mpg, &fgPct, &threePct, &ftPct,
	)

	if err != nil {
		return nil, fmt.Errorf("calculating season averages: %w", err)
	}

	averages := map[string]float64{
		"games_played": float64(gamesPlayed),
	}

	if ppg.Valid {
		averages["ppg"] = ppg.Float64
	}
	if rpg.Valid {
		averages["rpg"] = rpg.Float64
	}
	if apg.Valid {
		averages["apg"] = apg.Float64
	}
	if spg.Valid {
		averages["spg"] = spg.Float64
	}
	if bpg.Valid {
		averages["bpg"] = bpg.Float64
	}
	if tpg.Valid {
		averages["tpg"] = tpg.Float64
	}
	if mpg.Valid {
		averages["mpg"] = mpg.Float64
	}
	if fgPct.Valid {
		averages["fg_pct"] = fgPct.Float64
	}
	if threePct.Valid {
		averages["three_pct"] = threePct.Float64
	}
	if ftPct.Valid {
		averages["ft_pct"] = ftPct.Float64
	}

	return averages, nil
}

// UpsertPlayerStats inserts or updates player game stats
func (r *StatsRepository) UpsertPlayerStats(ctx context.Context, stats *store.PlayerGameStats) error {
	query := `
		INSERT INTO player_game_stats (game_id, player_id, team_id, points, rebounds, assists,
			steals, blocks, turnovers, field_goals_made, field_goals_attempted,
			three_pointers_made, three_pointers_attempted, free_throws_made, free_throws_attempted,
			offensive_rebounds, defensive_rebounds, personal_fouls, minutes_played, plus_minus,
			starter, true_shooting_pct, effective_fg_pct, usage_rate)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)
		ON CONFLICT (game_id, player_id) DO UPDATE SET
			team_id = EXCLUDED.team_id,
			points = EXCLUDED.points,
			rebounds = EXCLUDED.rebounds,
			assists = EXCLUDED.assists,
			steals = EXCLUDED.steals,
			blocks = EXCLUDED.blocks,
			turnovers = EXCLUDED.turnovers,
			field_goals_made = EXCLUDED.field_goals_made,
			field_goals_attempted = EXCLUDED.field_goals_attempted,
			three_pointers_made = EXCLUDED.three_pointers_made,
			three_pointers_attempted = EXCLUDED.three_pointers_attempted,
			free_throws_made = EXCLUDED.free_throws_made,
			free_throws_attempted = EXCLUDED.free_throws_attempted,
			offensive_rebounds = EXCLUDED.offensive_rebounds,
			defensive_rebounds = EXCLUDED.defensive_rebounds,
			personal_fouls = EXCLUDED.personal_fouls,
			minutes_played = EXCLUDED.minutes_played,
			plus_minus = EXCLUDED.plus_minus,
			starter = EXCLUDED.starter,
			true_shooting_pct = EXCLUDED.true_shooting_pct,
			effective_fg_pct = EXCLUDED.effective_fg_pct,
			usage_rate = EXCLUDED.usage_rate,
			updated_at = NOW()
		RETURNING stat_id
	`

	err := r.db.DB().QueryRowContext(ctx, query,
		stats.GameID, stats.PlayerID, stats.TeamID, stats.Points, stats.Rebounds, stats.Assists,
		stats.Steals, stats.Blocks, stats.Turnovers, stats.FieldGoalsMade, stats.FieldGoalsAttempted,
		stats.ThreePointersMade, stats.ThreePointersAttempted, stats.FreeThrowsMade, stats.FreeThrowsAttempted,
		stats.OffensiveRebounds, stats.DefensiveRebounds, stats.PersonalFouls, stats.MinutesPlayed, stats.PlusMinus,
		stats.Starter, stats.TrueShootingPct, stats.EffectiveFGPct, stats.UsageRate,
	).Scan(&stats.ID)

	if err != nil {
		return fmt.Errorf("upserting player stats: %w", err)
	}

	return nil
}

// scanPlayerStats scans multiple player stats rows
func (r *StatsRepository) scanPlayerStats(rows *sql.Rows) ([]*store.PlayerGameStats, error) {
	var allStats []*store.PlayerGameStats
	for rows.Next() {
		stats := &store.PlayerGameStats{}
		err := rows.Scan(
			&stats.ID, &stats.GameID, &stats.PlayerID, &stats.TeamID, &stats.Points, &stats.Rebounds,
			&stats.Assists, &stats.Steals, &stats.Blocks, &stats.Turnovers, &stats.FieldGoalsMade,
			&stats.FieldGoalsAttempted, &stats.ThreePointersMade, &stats.ThreePointersAttempted,
			&stats.FreeThrowsMade, &stats.FreeThrowsAttempted, &stats.OffensiveRebounds,
			&stats.DefensiveRebounds, &stats.PersonalFouls, &stats.MinutesPlayed, &stats.PlusMinus,
			&stats.Starter, &stats.TrueShootingPct, &stats.EffectiveFGPct, &stats.UsageRate,
			&stats.CreatedAt, &stats.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning player stats: %w", err)
		}
		allStats = append(allStats, stats)
	}

	return allStats, rows.Err()
}

// UpsertTeamStats inserts or updates team game stats
func (r *StatsRepository) UpsertTeamStats(ctx context.Context, stats *store.TeamGameStats) error {
	query := `
		INSERT INTO team_game_stats (
			game_id, team_id, is_home, points,
			field_goals_made, field_goals_attempted,
			three_pointers_made, three_pointers_attempted,
			free_throws_made, free_throws_attempted,
			offensive_rebounds, defensive_rebounds, rebounds,
			assists, steals, blocks, turnovers, personal_fouls
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		ON CONFLICT (game_id, team_id) DO UPDATE SET
			is_home = EXCLUDED.is_home,
			points = EXCLUDED.points,
			field_goals_made = EXCLUDED.field_goals_made,
			field_goals_attempted = EXCLUDED.field_goals_attempted,
			three_pointers_made = EXCLUDED.three_pointers_made,
			three_pointers_attempted = EXCLUDED.three_pointers_attempted,
			free_throws_made = EXCLUDED.free_throws_made,
			free_throws_attempted = EXCLUDED.free_throws_attempted,
			offensive_rebounds = EXCLUDED.offensive_rebounds,
			defensive_rebounds = EXCLUDED.defensive_rebounds,
			rebounds = EXCLUDED.rebounds,
			assists = EXCLUDED.assists,
			steals = EXCLUDED.steals,
			blocks = EXCLUDED.blocks,
			turnovers = EXCLUDED.turnovers,
			personal_fouls = EXCLUDED.personal_fouls,
			updated_at = NOW()
		RETURNING stat_id
	`

	err := r.db.DB().QueryRowContext(ctx, query,
		stats.GameID, stats.TeamID, stats.IsHome, stats.Points,
		stats.FieldGoalsMade, stats.FieldGoalsAttempted,
		stats.ThreePointersMade, stats.ThreePointersAttempted,
		stats.FreeThrowsMade, stats.FreeThrowsAttempted,
		stats.OffensiveRebounds, stats.DefensiveRebounds, stats.Rebounds,
		stats.Assists, stats.Steals, stats.Blocks, stats.Turnovers, stats.PersonalFouls,
	).Scan(&stats.ID)

	if err != nil {
		return fmt.Errorf("upserting team stats: %w", err)
	}

	return nil
}
