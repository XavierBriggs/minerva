package service

import (
	"context"
	"fmt"

	"github.com/fortuna/minerva/internal/store"
	"github.com/fortuna/minerva/internal/store/repository"
)

// PlayerService handles player-related business logic
type PlayerService struct {
	playerRepo *repository.PlayerRepository
	statsRepo  *repository.StatsRepository
	teamRepo   *repository.TeamRepository
}

// NewPlayerService creates a new player service
func NewPlayerService(db *store.Database) *PlayerService {
	return &PlayerService{
		playerRepo: repository.NewPlayerRepository(db),
		statsRepo:  repository.NewStatsRepository(db),
		teamRepo:   repository.NewTeamRepository(db),
	}
}

// GetPlayer retrieves a player by ID with team details
func (s *PlayerService) GetPlayer(ctx context.Context, playerID int) (*PlayerProfile, error) {
	player, err := s.playerRepo.GetByID(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("fetching player: %w", err)
	}

	// Lookup current team from player_team_history table
	var team *store.Team
	if teamID, err := s.playerRepo.GetCurrentTeamID(ctx, playerID); err == nil {
		team, _ = s.teamRepo.GetByID(ctx, teamID)
	}

	return &PlayerProfile{
		Player: player,
		Team:   team,
	}, nil
}

// SearchPlayers searches for players by name
func (s *PlayerService) SearchPlayers(ctx context.Context, name string) ([]*PlayerProfile, error) {
	players, err := s.playerRepo.GetByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("searching players: %w", err)
	}

	profiles := make([]*PlayerProfile, 0, len(players))
	for _, player := range players {
		// Lookup current team from player_team_history table
		var team *store.Team
		if teamID, err := s.playerRepo.GetCurrentTeamID(ctx, player.PlayerID); err == nil {
			team, _ = s.teamRepo.GetByID(ctx, teamID)
		}

		profiles = append(profiles, &PlayerProfile{
			Player: player,
			Team:   team,
		})
	}

	return profiles, nil
}

// GetTeamRoster retrieves all players on a team
func (s *PlayerService) GetTeamRoster(ctx context.Context, teamID int) ([]*PlayerProfile, error) {
	players, err := s.playerRepo.GetByCurrentTeam(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("fetching team roster: %w", err)
	}

	team, err := s.teamRepo.GetByID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("fetching team: %w", err)
	}

	profiles := make([]*PlayerProfile, 0, len(players))
	for _, player := range players {
		profiles = append(profiles, &PlayerProfile{
			Player: player,
			Team:   team,
		})
	}

	return profiles, nil
}

// GetPlayerStats retrieves a player's recent game stats with enriched game context
func (s *PlayerService) GetPlayerStats(ctx context.Context, playerID int, limit int) ([]*repository.EnrichedPlayerStats, error) {
	stats, err := s.statsRepo.GetPlayerRecentStatsEnriched(ctx, playerID, limit)
	if err != nil {
		return nil, fmt.Errorf("fetching player stats: %w", err)
	}

	return stats, nil
}

// GetPlayerSeasonAverages retrieves a player's season averages
func (s *PlayerService) GetPlayerSeasonAverages(ctx context.Context, playerID int, seasonID string) (map[string]float64, error) {
	averages, err := s.statsRepo.GetPlayerSeasonAverages(ctx, playerID, seasonID)
	if err != nil {
		return nil, fmt.Errorf("calculating season averages: %w", err)
	}

	return averages, nil
}

// PlayerProfile contains player details with team information
type PlayerProfile struct {
	Player *store.Player `json:"player"`
	Team   *store.Team   `json:"team,omitempty"`
}
