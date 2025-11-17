package service

import (
	"context"
	"fmt"

	"github.com/fortuna/minerva/internal/store"
	"github.com/fortuna/minerva/internal/store/repository"
)

// StatsService handles statistics-related business logic
type StatsService struct {
	statsRepo  *repository.StatsRepository
	playerRepo *repository.PlayerRepository
	teamRepo   *repository.TeamRepository
	gameRepo   *repository.GameRepository
}

// NewStatsService creates a new stats service
func NewStatsService(db *store.Database) *StatsService {
	return &StatsService{
		statsRepo:  repository.NewStatsRepository(db),
		playerRepo: repository.NewPlayerRepository(db),
		teamRepo:   repository.NewTeamRepository(db),
		gameRepo:   repository.NewGameRepository(db),
	}
}

// GetGameBoxScore retrieves the full box score for a game with player and team details
func (s *StatsService) GetGameBoxScore(ctx context.Context, gameID string) (*BoxScore, error) {
	// Get game details
	game, err := s.gameRepo.GetByExternalID(ctx, gameID)
	if err != nil {
		return nil, fmt.Errorf("fetching game: %w", err)
	}

	// Get all player stats for the game
	playerStats, err := s.statsRepo.GetGameBoxScore(ctx, gameID)
	if err != nil {
		return nil, fmt.Errorf("fetching box score: %w", err)
	}

	// Organize stats by team
	homeTeamStats := make([]*PlayerStatLine, 0)
	awayTeamStats := make([]*PlayerStatLine, 0)

	for _, stat := range playerStats {
		player, err := s.playerRepo.GetByID(ctx, stat.PlayerID)
		if err != nil {
			continue // Skip if player not found
		}

		statLine := &PlayerStatLine{
			Player: player,
			Stats:  stat,
		}

		if stat.TeamID == game.HomeTeamID {
			homeTeamStats = append(homeTeamStats, statLine)
		} else {
			awayTeamStats = append(awayTeamStats, statLine)
		}
	}

	homeTeam, err := s.teamRepo.GetByID(ctx, game.HomeTeamID)
	if err != nil {
		return nil, fmt.Errorf("fetching home team: %w", err)
	}

	awayTeam, err := s.teamRepo.GetByID(ctx, game.AwayTeamID)
	if err != nil {
		return nil, fmt.Errorf("fetching away team: %w", err)
	}

	return &BoxScore{
		Game:          game,
		HomeTeam:      homeTeam,
		AwayTeam:      awayTeam,
		HomeTeamStats: homeTeamStats,
		AwayTeamStats: awayTeamStats,
	}, nil
}

// GetPlayerGameStats retrieves stats for a specific player in a game
func (s *StatsService) GetPlayerGameStats(ctx context.Context, gameID string, playerID int) (*PlayerStatLine, error) {
	stats, err := s.statsRepo.GetPlayerGameStats(ctx, gameID, playerID)
	if err != nil {
		return nil, fmt.Errorf("fetching player game stats: %w", err)
	}

	player, err := s.playerRepo.GetByID(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("fetching player: %w", err)
	}

	return &PlayerStatLine{
		Player: player,
		Stats:  stats,
	}, nil
}

// BoxScore contains the complete box score for a game
type BoxScore struct {
	Game          *store.Game        `json:"game"`
	HomeTeam      *store.Team        `json:"home_team"`
	AwayTeam      *store.Team        `json:"away_team"`
	HomeTeamStats []*PlayerStatLine  `json:"home_team_stats"`
	AwayTeamStats []*PlayerStatLine  `json:"away_team_stats"`
}

// PlayerStatLine combines player info with their game stats
type PlayerStatLine struct {
	Player *store.Player          `json:"player"`
	Stats  *store.PlayerGameStats `json:"stats"`
}

