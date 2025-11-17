package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/fortuna/minerva/internal/store"
)

// TeamRepository handles team data access
type TeamRepository struct {
	db *store.Database
}

// NewTeamRepository creates a new team repository
func NewTeamRepository(db *store.Database) *TeamRepository {
	return &TeamRepository{db: db}
}

// GetAll returns all NBA teams
func (r *TeamRepository) GetAll(ctx context.Context) ([]*store.Team, error) {
	query := `
		SELECT team_id, sport, external_id, abbreviation, full_name, short_name, 
			city, state, conference, division, venue_name, venue_capacity, 
			founded_year, logo_url, colors, social_media, metadata, is_active,
			created_at, updated_at
		FROM teams
		WHERE is_active = true
		ORDER BY abbreviation
	`

	rows, err := r.db.DB().QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying teams: %w", err)
	}
	defer rows.Close()

	var teams []*store.Team
	for rows.Next() {
		team := &store.Team{}
		err := rows.Scan(
			&team.TeamID, &team.Sport, &team.ExternalID, &team.Abbreviation, 
			&team.FullName, &team.ShortName, &team.City, &team.State,
			&team.Conference, &team.Division, &team.VenueName, &team.VenueCapacity,
			&team.FoundedYear, &team.LogoURL, &team.Colors, &team.SocialMedia, 
			&team.Metadata, &team.IsActive, &team.CreatedAt, &team.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning team: %w", err)
		}
		teams = append(teams, team)
	}

	return teams, rows.Err()
}

// GetByID finds a team by ID
func (r *TeamRepository) GetByID(ctx context.Context, teamID int) (*store.Team, error) {
	query := `
		SELECT team_id, sport, external_id, abbreviation, full_name, short_name, 
			city, state, conference, division, venue_name, venue_capacity, 
			founded_year, logo_url, colors, social_media, metadata, is_active,
			created_at, updated_at
		FROM teams
		WHERE team_id = $1
	`

	team := &store.Team{}
	err := r.db.DB().QueryRowContext(ctx, query, teamID).Scan(
		&team.TeamID, &team.Sport, &team.ExternalID, &team.Abbreviation, 
		&team.FullName, &team.ShortName, &team.City, &team.State,
		&team.Conference, &team.Division, &team.VenueName, &team.VenueCapacity,
		&team.FoundedYear, &team.LogoURL, &team.Colors, &team.SocialMedia, 
		&team.Metadata, &team.IsActive, &team.CreatedAt, &team.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("team not found: %d", teamID)
	}
	if err != nil {
		return nil, fmt.Errorf("querying team: %w", err)
	}

	return team, nil
}

// GetByAbbreviation finds a team by abbreviation (e.g., "LAL", "BOS")
func (r *TeamRepository) GetByAbbreviation(ctx context.Context, abbr string) (*store.Team, error) {
	query := `
		SELECT team_id, sport, external_id, abbreviation, full_name, short_name, 
			city, state, conference, division, venue_name, venue_capacity, 
			founded_year, logo_url, colors, social_media, metadata, is_active,
			created_at, updated_at
		FROM teams
		WHERE abbreviation = $1
	`

	team := &store.Team{}
	err := r.db.DB().QueryRowContext(ctx, query, abbr).Scan(
		&team.TeamID, &team.Sport, &team.ExternalID, &team.Abbreviation, 
		&team.FullName, &team.ShortName, &team.City, &team.State,
		&team.Conference, &team.Division, &team.VenueName, &team.VenueCapacity,
		&team.FoundedYear, &team.LogoURL, &team.Colors, &team.SocialMedia, 
		&team.Metadata, &team.IsActive, &team.CreatedAt, &team.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("team not found: %s", abbr)
	}
	if err != nil {
		return nil, fmt.Errorf("querying team: %w", err)
	}

	return team, nil
}

// GetByESPNID finds a team by ESPN team ID (external_id)
func (r *TeamRepository) GetByESPNID(ctx context.Context, espnID string) (*store.Team, error) {
	query := `
		SELECT team_id, sport, external_id, abbreviation, full_name, short_name, 
			city, state, conference, division, venue_name, venue_capacity, 
			founded_year, logo_url, colors, social_media, metadata, is_active,
			created_at, updated_at
		FROM teams
		WHERE external_id = $1
	`

	team := &store.Team{}
	err := r.db.DB().QueryRowContext(ctx, query, espnID).Scan(
		&team.TeamID, &team.Sport, &team.ExternalID, &team.Abbreviation, 
		&team.FullName, &team.ShortName, &team.City, &team.State,
		&team.Conference, &team.Division, &team.VenueName, &team.VenueCapacity,
		&team.FoundedYear, &team.LogoURL, &team.Colors, &team.SocialMedia, 
		&team.Metadata, &team.IsActive, &team.CreatedAt, &team.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("team not found with ESPN ID: %s", espnID)
	}
	if err != nil {
		return nil, fmt.Errorf("querying team: %w", err)
	}

	return team, nil
}

// GetByConference returns all teams in a conference
func (r *TeamRepository) GetByConference(ctx context.Context, conference string) ([]*store.Team, error) {
	query := `
		SELECT team_id, sport, external_id, abbreviation, full_name, short_name, 
			city, state, conference, division, venue_name, venue_capacity, 
			founded_year, logo_url, colors, social_media, metadata, is_active,
			created_at, updated_at
		FROM teams
		WHERE conference = $1 AND is_active = true
		ORDER BY division, abbreviation
	`

	rows, err := r.db.DB().QueryContext(ctx, query, conference)
	if err != nil {
		return nil, fmt.Errorf("querying teams: %w", err)
	}
	defer rows.Close()

	var teams []*store.Team
	for rows.Next() {
		team := &store.Team{}
		err := rows.Scan(
			&team.TeamID, &team.Sport, &team.ExternalID, &team.Abbreviation, 
			&team.FullName, &team.ShortName, &team.City, &team.State,
			&team.Conference, &team.Division, &team.VenueName, &team.VenueCapacity,
			&team.FoundedYear, &team.LogoURL, &team.Colors, &team.SocialMedia, 
			&team.Metadata, &team.IsActive, &team.CreatedAt, &team.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning team: %w", err)
		}
		teams = append(teams, team)
	}

	return teams, rows.Err()
}

