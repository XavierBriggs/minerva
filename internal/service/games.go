package service

import (
	"context"
	"fmt"
	"time"

	"github.com/fortuna/minerva/internal/store"
	"github.com/fortuna/minerva/internal/store/repository"
)

// GameService handles game-related business logic
type GameService struct {
	gameRepo *repository.GameRepository
	teamRepo *repository.TeamRepository
}

// NewGameService creates a new game service
func NewGameService(db *store.Database) *GameService {
	return &GameService{
		gameRepo: repository.NewGameRepository(db),
		teamRepo: repository.NewTeamRepository(db),
	}
}

// GetGame retrieves a game by ID with team details
func (s *GameService) GetGame(ctx context.Context, gameID string) (*GameSummary, error) {
	game, err := s.gameRepo.GetByExternalID(ctx, gameID)
	if err != nil {
		return nil, fmt.Errorf("fetching game: %w", err)
	}

	homeTeam, err := s.teamRepo.GetByID(ctx, game.HomeTeamID)
	if err != nil {
		return nil, fmt.Errorf("fetching home team: %w", err)
	}

	awayTeam, err := s.teamRepo.GetByID(ctx, game.AwayTeamID)
	if err != nil {
		return nil, fmt.Errorf("fetching away team: %w", err)
	}

	return &GameSummary{
		Game:     game,
		HomeTeam: homeTeam,
		AwayTeam: awayTeam,
	}, nil
}

// GetLiveGames retrieves all currently live games
func (s *GameService) GetLiveGames(ctx context.Context) ([]*GameSummary, error) {
	games, err := s.gameRepo.GetLiveGames(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching live games: %w", err)
	}

	return s.enrichGamesWithTeams(ctx, games)
}

// GetGamesByDate retrieves all games on a specific date
func (s *GameService) GetGamesByDate(ctx context.Context, date time.Time) ([]*GameSummary, error) {
	games, err := s.gameRepo.GetByDate(ctx, date)
	if err != nil {
		return nil, fmt.Errorf("fetching games by date: %w", err)
	}

	return s.enrichGamesWithTeams(ctx, games)
}

// GetUpcomingGames retrieves upcoming scheduled games
func (s *GameService) GetUpcomingGames(ctx context.Context, limit int) ([]*GameSummary, error) {
	games, err := s.gameRepo.GetUpcomingGames(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("fetching upcoming games: %w", err)
	}

	return s.enrichGamesWithTeams(ctx, games)
}

// GetTodaysGames retrieves all games for today (live, scheduled, and final)
func (s *GameService) GetTodaysGames(ctx context.Context) ([]*GameSummary, error) {
	games, err := s.gameRepo.GetTodaysGames(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching today's games: %w", err)
	}

	return s.enrichGamesWithTeams(ctx, games)
}

// GetTeamSchedule retrieves games for a specific team
func (s *GameService) GetTeamSchedule(ctx context.Context, teamID int, seasonID int, limit int) ([]*GameSummary, error) {
	games, err := s.gameRepo.GetByTeam(ctx, teamID, seasonID, limit)
	if err != nil {
		return nil, fmt.Errorf("fetching team schedule: %w", err)
	}

	return s.enrichGamesWithTeams(ctx, games)
}

// CleanupStaleGames marks old "in_progress" games as "final"
func (s *GameService) CleanupStaleGames(ctx context.Context) (int64, error) {
	count, err := s.gameRepo.CleanupStaleGames(ctx)
	if err != nil {
		return 0, fmt.Errorf("cleaning up stale games: %w", err)
	}
	return count, nil
}

// enrichGamesWithTeams adds team details to games
func (s *GameService) enrichGamesWithTeams(ctx context.Context, games []*store.Game) ([]*GameSummary, error) {
	summaries := make([]*GameSummary, 0, len(games))

	for _, game := range games {
		homeTeam, err := s.teamRepo.GetByID(ctx, game.HomeTeamID)
		if err != nil {
			return nil, fmt.Errorf("fetching home team for game %s: %w", game.GameID, err)
		}

		awayTeam, err := s.teamRepo.GetByID(ctx, game.AwayTeamID)
		if err != nil {
			return nil, fmt.Errorf("fetching away team for game %s: %w", game.GameID, err)
		}

		summaries = append(summaries, &GameSummary{
			Game:     game,
			HomeTeam: homeTeam,
			AwayTeam: awayTeam,
		})
	}

	return summaries, nil
}

// GameSummary contains game details with team information
type GameSummary struct {
	Game     *store.Game `json:"game"`
	HomeTeam *store.Team `json:"home_team"`
	AwayTeam *store.Team `json:"away_team"`
}

