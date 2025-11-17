package store

import (
	"database/sql"
	"time"
)

// Season represents an NBA season (v2 schema)
type Season struct {
	SeasonID    int            `json:"season_id" db:"season_id"`
	Sport       string         `json:"sport" db:"sport"`
	SeasonYear  string         `json:"season_year" db:"season_year"`
	SeasonType  string         `json:"season_type" db:"season_type"`
	StartDate   time.Time      `json:"start_date" db:"start_date"`
	EndDate     time.Time      `json:"end_date" db:"end_date"`
	IsActive    bool           `json:"is_active" db:"is_active"`
	TotalGames  sql.NullInt32  `json:"total_games,omitempty" db:"total_games"`
	Metadata    sql.NullString `json:"metadata,omitempty" db:"metadata"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" db:"updated_at"`
}

// Team represents an NBA franchise (v2 schema)
type Team struct {
	TeamID        int            `json:"team_id" db:"team_id"`
	Sport         string         `json:"sport" db:"sport"`
	ExternalID    string         `json:"external_id" db:"external_id"`
	Abbreviation  string         `json:"abbreviation" db:"abbreviation"`
	FullName      string         `json:"full_name" db:"full_name"`
	ShortName     string         `json:"short_name" db:"short_name"`
	City          sql.NullString `json:"city,omitempty" db:"city"`
	State         sql.NullString `json:"state,omitempty" db:"state"`
	Conference    sql.NullString `json:"conference,omitempty" db:"conference"`
	Division      sql.NullString `json:"division,omitempty" db:"division"`
	VenueName     sql.NullString `json:"venue_name,omitempty" db:"venue_name"`
	VenueCapacity sql.NullInt32  `json:"venue_capacity,omitempty" db:"venue_capacity"`
	FoundedYear   sql.NullInt32  `json:"founded_year,omitempty" db:"founded_year"`
	LogoURL       sql.NullString `json:"logo_url,omitempty" db:"logo_url"`
	Colors        sql.NullString `json:"colors,omitempty" db:"colors"`
	SocialMedia   sql.NullString `json:"social_media,omitempty" db:"social_media"`
	Metadata      sql.NullString `json:"metadata,omitempty" db:"metadata"`
	IsActive      bool           `json:"is_active" db:"is_active"`
	CreatedAt     time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at" db:"updated_at"`
}

// Player represents a player (v2 schema)
type Player struct {
	PlayerID      int            `json:"player_id" db:"player_id"`
	Sport         string         `json:"sport" db:"sport"`
	ExternalID    sql.NullString `json:"external_id,omitempty" db:"external_id"`
	FirstName     sql.NullString `json:"first_name,omitempty" db:"first_name"`
	LastName      string         `json:"last_name" db:"last_name"`
	FullName      string         `json:"full_name" db:"full_name"`
	DisplayName   sql.NullString `json:"display_name,omitempty" db:"display_name"`
	BirthDate     sql.NullTime   `json:"birth_date,omitempty" db:"birth_date"`
	BirthCity     sql.NullString `json:"birth_city,omitempty" db:"birth_city"`
	BirthCountry  sql.NullString `json:"birth_country,omitempty" db:"birth_country"`
	Nationality   sql.NullString `json:"nationality,omitempty" db:"nationality"`
	Height        sql.NullString `json:"height,omitempty" db:"height"`
	HeightInches  sql.NullInt32  `json:"height_inches,omitempty" db:"height_inches"`
	Weight        sql.NullInt32  `json:"weight,omitempty" db:"weight"`
	Position      sql.NullString `json:"position,omitempty" db:"position"`
	College       sql.NullString `json:"college,omitempty" db:"college"`
	HighSchool    sql.NullString `json:"high_school,omitempty" db:"high_school"`
	DraftYear     sql.NullInt32  `json:"draft_year,omitempty" db:"draft_year"`
	DraftRound    sql.NullInt32  `json:"draft_round,omitempty" db:"draft_round"`
	DraftPick     sql.NullInt32  `json:"draft_pick,omitempty" db:"draft_pick"`
	DraftTeamID   sql.NullInt32  `json:"draft_team_id,omitempty" db:"draft_team_id"`
	HeadshotURL   sql.NullString `json:"headshot_url,omitempty" db:"headshot_url"`
	JerseyNumber  sql.NullString `json:"jersey_number,omitempty" db:"jersey_number"`
	Status        sql.NullString `json:"status,omitempty" db:"status"`
	Metadata      sql.NullString `json:"metadata,omitempty" db:"metadata"`
	CreatedAt     time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at" db:"updated_at"`
	
	// Not in database - populated from player_team_history for API responses
	CurrentTeamID int `json:"current_team_id,omitempty" db:"-"`
}

// PlayerSeason represents a player's participation in a season
type PlayerSeason struct {
	ID            int             `json:"id" db:"id"`
	PlayerID      int             `json:"player_id" db:"player_id"`
	SeasonID      string          `json:"season_id" db:"season_id"`
	TeamID        sql.NullInt32   `json:"team_id,omitempty" db:"team_id"`
	WasActive     bool            `json:"was_active" db:"was_active"`
	GamesPlayed   int             `json:"games_played" db:"games_played"`
	SeasonPPG     sql.NullFloat64 `json:"season_ppg,omitempty" db:"season_ppg"`
	SeasonRPG     sql.NullFloat64 `json:"season_rpg,omitempty" db:"season_rpg"`
	SeasonAPG     sql.NullFloat64 `json:"season_apg,omitempty" db:"season_apg"`
	SeasonMinutes sql.NullFloat64 `json:"season_minutes,omitempty" db:"season_minutes"`
	SeasonFGPct   sql.NullFloat64 `json:"season_fg_pct,omitempty" db:"season_fg_pct"`
	Season3PPct   sql.NullFloat64 `json:"season_3p_pct,omitempty" db:"season_3p_pct"`
	SeasonFTPct   sql.NullFloat64 `json:"season_ft_pct,omitempty" db:"season_ft_pct"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`
}

// Game represents an NBA game (v2 schema)
type Game struct {
	GameID        int            `json:"game_id" db:"game_id"`
	Sport         string         `json:"sport" db:"sport"`
	SeasonID      int            `json:"season_id" db:"season_id"`
	ExternalID    string         `json:"external_id" db:"external_id"`
	GameDate      time.Time      `json:"game_date" db:"game_date"`
	GameTime      sql.NullTime   `json:"game_time,omitempty" db:"game_time"`
	HomeTeamID    int            `json:"home_team_id" db:"home_team_id"`
	AwayTeamID    int            `json:"away_team_id" db:"away_team_id"`
	HomeScore     sql.NullInt32  `json:"home_score,omitempty" db:"home_score"`
	AwayScore     sql.NullInt32  `json:"away_score,omitempty" db:"away_score"`
	Status        string         `json:"status" db:"status"`
	Period        sql.NullInt32  `json:"period,omitempty" db:"period"`
	Clock         sql.NullString `json:"clock,omitempty" db:"clock"`
	Venue         sql.NullString `json:"venue,omitempty" db:"venue"`
	Attendance    sql.NullInt32  `json:"attendance,omitempty" db:"attendance"`
	Metadata      sql.NullString `json:"metadata,omitempty" db:"metadata"`
	CreatedAt     time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at" db:"updated_at"`
}

// PlayerGameStats represents player stats for a single game
type PlayerGameStats struct {
	ID                    int             `json:"id" db:"id"`
	GameID                int             `json:"game_id" db:"game_id"`
	PlayerID              int             `json:"player_id" db:"player_id"`
	TeamID                int             `json:"team_id" db:"team_id"`
	Points                int             `json:"points" db:"points"`
	Rebounds              int             `json:"rebounds" db:"rebounds"`
	Assists               int             `json:"assists" db:"assists"`
	Steals                int             `json:"steals" db:"steals"`
	Blocks                int             `json:"blocks" db:"blocks"`
	Turnovers             int             `json:"turnovers" db:"turnovers"`
	FieldGoalsMade        int             `json:"field_goals_made" db:"field_goals_made"`
	FieldGoalsAttempted   int             `json:"field_goals_attempted" db:"field_goals_attempted"`
	ThreePointersMade     int             `json:"three_pointers_made" db:"three_pointers_made"`
	ThreePointersAttempted int            `json:"three_pointers_attempted" db:"three_pointers_attempted"`
	FreeThrowsMade        int             `json:"free_throws_made" db:"free_throws_made"`
	FreeThrowsAttempted   int             `json:"free_throws_attempted" db:"free_throws_attempted"`
	OffensiveRebounds     int             `json:"offensive_rebounds" db:"offensive_rebounds"`
	DefensiveRebounds     int             `json:"defensive_rebounds" db:"defensive_rebounds"`
	PersonalFouls         int             `json:"personal_fouls" db:"personal_fouls"`
	MinutesPlayed         sql.NullFloat64 `json:"minutes_played,omitempty" db:"minutes_played"`
	PlusMinus             sql.NullInt32   `json:"plus_minus,omitempty" db:"plus_minus"`
	Starter               bool            `json:"starter" db:"starter"`
	TrueShootingPct       sql.NullFloat64 `json:"true_shooting_pct,omitempty" db:"true_shooting_pct"`
	EffectiveFGPct        sql.NullFloat64 `json:"effective_fg_pct,omitempty" db:"effective_fg_pct"`
	UsageRate             sql.NullFloat64 `json:"usage_rate,omitempty" db:"usage_rate"`
	CreatedAt             time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at" db:"updated_at"`
}

// TeamGameStats represents team stats for a single game
type TeamGameStats struct {
	ID                     int             `json:"id" db:"id"`
	GameID                 int             `json:"game_id" db:"game_id"`
	TeamID                 int             `json:"team_id" db:"team_id"`
	IsHome                 bool            `json:"is_home" db:"is_home"`
	Points                 int             `json:"points" db:"points"`
	FieldGoalsMade         int             `json:"field_goals_made" db:"field_goals_made"`
	FieldGoalsAttempted    int             `json:"field_goals_attempted" db:"field_goals_attempted"`
	ThreePointersMade      int             `json:"three_pointers_made" db:"three_pointers_made"`
	ThreePointersAttempted int             `json:"three_pointers_attempted" db:"three_pointers_attempted"`
	FreeThrowsMade         int             `json:"free_throws_made" db:"free_throws_made"`
	FreeThrowsAttempted    int             `json:"free_throws_attempted" db:"free_throws_attempted"`
	OffensiveRebounds      int             `json:"offensive_rebounds" db:"offensive_rebounds"`
	DefensiveRebounds      int             `json:"defensive_rebounds" db:"defensive_rebounds"`
	Rebounds               int             `json:"rebounds" db:"rebounds"`
	Assists                int             `json:"assists" db:"assists"`
	Steals                 int             `json:"steals" db:"steals"`
	Blocks                 int             `json:"blocks" db:"blocks"`
	Turnovers              int             `json:"turnovers" db:"turnovers"`
	PersonalFouls          int             `json:"personal_fouls" db:"personal_fouls"`
	TrueShootingPct        sql.NullFloat64 `json:"true_shooting_pct,omitempty" db:"true_shooting_pct"`
	EffectiveFGPct         sql.NullFloat64 `json:"effective_fg_pct,omitempty" db:"effective_fg_pct"`
	TurnoverPct            sql.NullFloat64 `json:"turnover_pct,omitempty" db:"turnover_pct"`
	OffensiveReboundPct    sql.NullFloat64 `json:"offensive_rebound_pct,omitempty" db:"offensive_rebound_pct"`
	DefensiveReboundPct    sql.NullFloat64 `json:"defensive_rebound_pct,omitempty" db:"defensive_rebound_pct"`
	FreeThrowRate          sql.NullFloat64 `json:"free_throw_rate,omitempty" db:"free_throw_rate"`
	Possessions            sql.NullFloat64 `json:"possessions,omitempty" db:"possessions"`
	Pace                   sql.NullFloat64 `json:"pace,omitempty" db:"pace"`
	OffensiveRating        sql.NullFloat64 `json:"offensive_rating,omitempty" db:"offensive_rating"`
	DefensiveRating        sql.NullFloat64 `json:"defensive_rating,omitempty" db:"defensive_rating"`
	NetRating              sql.NullFloat64 `json:"net_rating,omitempty" db:"net_rating"`
	CreatedAt              time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time       `json:"updated_at" db:"updated_at"`
}

// OddsMapping links ESPN games to Alexandria events
type OddsMapping struct {
	ID                  int            `json:"id" db:"id"`
	ESPNGameID          string         `json:"espn_game_id" db:"espn_game_id"`
	AlexandriaEventID   sql.NullString `json:"alexandria_event_id,omitempty" db:"alexandria_event_id"`
	MappingConfidence   float64        `json:"mapping_confidence" db:"mapping_confidence"`
	MappedAt            time.Time      `json:"mapped_at" db:"mapped_at"`
	CreatedAt           time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at" db:"updated_at"`
}

