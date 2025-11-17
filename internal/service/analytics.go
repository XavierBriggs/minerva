package service

import (
	"context"
	"fmt"
	"math"

	"github.com/fortuna/minerva/internal/store"
	"github.com/fortuna/minerva/internal/store/repository"
)

// AnalyticsService handles advanced analytics and ML feature generation
type AnalyticsService struct {
	statsRepo  *repository.StatsRepository
	playerRepo *repository.PlayerRepository
	gameRepo   *repository.GameRepository
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(db *store.Database) *AnalyticsService {
	return &AnalyticsService{
		statsRepo:  repository.NewStatsRepository(db),
		playerRepo: repository.NewPlayerRepository(db),
		gameRepo:   repository.NewGameRepository(db),
	}
}

// GetPlayerPerformanceTrend calculates trending stats for a player
func (s *AnalyticsService) GetPlayerPerformanceTrend(ctx context.Context, playerID int, games int) (*PerformanceTrend, error) {
	recentStats, err := s.statsRepo.GetPlayerRecentStats(ctx, playerID, games)
	if err != nil {
		return nil, fmt.Errorf("fetching recent stats: %w", err)
	}

	if len(recentStats) == 0 {
		return nil, fmt.Errorf("no stats found for player %d", playerID)
	}

	// Calculate averages
	var totalPoints, totalRebounds, totalAssists, totalMinutes float64
	var totalFGM, totalFGA, total3PM, total3PA, totalFTM, totalFTA float64

	for _, stat := range recentStats {
		totalPoints += float64(stat.Points)
		totalRebounds += float64(stat.Rebounds)
		totalAssists += float64(stat.Assists)
		if stat.MinutesPlayed.Valid {
			totalMinutes += stat.MinutesPlayed.Float64
		}
		totalFGM += float64(stat.FieldGoalsMade)
		totalFGA += float64(stat.FieldGoalsAttempted)
		total3PM += float64(stat.ThreePointersMade)
		total3PA += float64(stat.ThreePointersAttempted)
		totalFTM += float64(stat.FreeThrowsMade)
		totalFTA += float64(stat.FreeThrowsAttempted)
	}

	gamesPlayed := float64(len(recentStats))

	trend := &PerformanceTrend{
		PlayerID:      playerID,
		GamesAnalyzed: len(recentStats),
		PPG:           totalPoints / gamesPlayed,
		RPG:           totalRebounds / gamesPlayed,
		APG:           totalAssists / gamesPlayed,
		MPG:           totalMinutes / gamesPlayed,
		FGPct:         safeDiv(totalFGM, totalFGA),
		ThreePct:      safeDiv(total3PM, total3PA),
		FTPct:         safeDiv(totalFTM, totalFTA),
	}

	// Calculate variance (consistency metric)
	var pointsVariance float64
	for _, stat := range recentStats {
		diff := float64(stat.Points) - trend.PPG
		pointsVariance += diff * diff
	}
	trend.PPGVariance = pointsVariance / gamesPlayed
	trend.PPGStdDev = math.Sqrt(trend.PPGVariance)

	return trend, nil
}

// GetPlayerMLFeatures generates ML features for a player's recent performance
func (s *AnalyticsService) GetPlayerMLFeatures(ctx context.Context, playerID int, seasonID string) (*MLFeatures, error) {
	// Get season averages
	seasonAvg, err := s.statsRepo.GetPlayerSeasonAverages(ctx, playerID, seasonID)
	if err != nil {
		return nil, fmt.Errorf("fetching season averages: %w", err)
	}

	// Get last 10 games for recent form
	recentStats, err := s.statsRepo.GetPlayerRecentStats(ctx, playerID, 10)
	if err != nil {
		return nil, fmt.Errorf("fetching recent stats: %w", err)
	}

	// Calculate recent form metrics
	var last10PPG, last10MPG, last10Usage float64
	if len(recentStats) > 0 {
		for _, stat := range recentStats {
			last10PPG += float64(stat.Points)
			if stat.MinutesPlayed.Valid {
				last10MPG += stat.MinutesPlayed.Float64
			}
			if stat.UsageRate.Valid {
				last10Usage += stat.UsageRate.Float64
			}
		}
		last10PPG /= float64(len(recentStats))
		last10MPG /= float64(len(recentStats))
		last10Usage /= float64(len(recentStats))
	}

	features := &MLFeatures{
		PlayerID: playerID,
		SeasonID: seasonID,

		// Season averages
		SeasonPPG: seasonAvg["ppg"],
		SeasonRPG: seasonAvg["rpg"],
		SeasonAPG: seasonAvg["apg"],
		SeasonMPG: seasonAvg["mpg"],
		SeasonFGPct: seasonAvg["fg_pct"],
		SeasonThreePct: seasonAvg["three_pct"],
		SeasonFTPct: seasonAvg["ft_pct"],

		// Recent form (last 10 games)
		Last10PPG: last10PPG,
		Last10MPG: last10MPG,
		Last10Usage: last10Usage,
		
		// Games played
		GamesPlayed: int(seasonAvg["games_played"]),
	}

	return features, nil
}

// PerformanceTrend contains trending performance metrics
type PerformanceTrend struct {
	PlayerID      int     `json:"player_id"`
	GamesAnalyzed int     `json:"games_analyzed"`
	PPG           float64 `json:"ppg"`
	RPG           float64 `json:"rpg"`
	APG           float64 `json:"apg"`
	MPG           float64 `json:"mpg"`
	FGPct         float64 `json:"fg_pct"`
	ThreePct      float64 `json:"three_pct"`
	FTPct         float64 `json:"ft_pct"`
	PPGVariance   float64 `json:"ppg_variance"`
	PPGStdDev     float64 `json:"ppg_std_dev"`
}

// MLFeatures contains machine learning features for a player
type MLFeatures struct {
	PlayerID       int     `json:"player_id"`
	SeasonID       string  `json:"season_id"`
	GamesPlayed    int     `json:"games_played"`
	
	// Season averages
	SeasonPPG      float64 `json:"season_ppg"`
	SeasonRPG      float64 `json:"season_rpg"`
	SeasonAPG      float64 `json:"season_apg"`
	SeasonMPG      float64 `json:"season_mpg"`
	SeasonFGPct    float64 `json:"season_fg_pct"`
	SeasonThreePct float64 `json:"season_three_pct"`
	SeasonFTPct    float64 `json:"season_ft_pct"`
	
	// Recent form
	Last10PPG      float64 `json:"last_10_ppg"`
	Last10MPG      float64 `json:"last_10_mpg"`
	Last10Usage    float64 `json:"last_10_usage"`
}

// safeDiv performs division with zero check
func safeDiv(numerator, denominator float64) float64 {
	if denominator == 0 {
		return 0
	}
	return numerator / denominator
}

