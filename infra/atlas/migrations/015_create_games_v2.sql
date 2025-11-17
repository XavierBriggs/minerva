-- Create games table (v2 - improved schema)
-- Based on ATLAS_DATABASE_DESIGN.md

CREATE TABLE games (
  game_id SERIAL PRIMARY KEY,
  sport VARCHAR(50) NOT NULL,
  external_id VARCHAR(100),                -- ESPN game ID
  season_id INTEGER NOT NULL REFERENCES seasons(season_id),
  home_team_id INTEGER NOT NULL REFERENCES teams(team_id),
  away_team_id INTEGER NOT NULL REFERENCES teams(team_id),
  game_date TIMESTAMP NOT NULL,
  game_time TIMESTAMP,                     -- Nullable game time (can be NULL for TBD games)
  venue VARCHAR(200),
  attendance INTEGER,
  status VARCHAR(20) NOT NULL,             -- 'scheduled', 'in_progress', 'final', 'postponed', 'cancelled'
  period INTEGER,                          -- Current quarter/inning/period
  clock VARCHAR(20),                       -- '5:23', 'End of 3rd', 'Halftime'
  home_score INTEGER,
  away_score INTEGER,
  overtime_periods INTEGER DEFAULT 0,
  broadcast_info JSONB DEFAULT '{}',       -- {tv: ['ESPN', 'TNT'], radio: ['KLAC']}
  weather JSONB DEFAULT '{}',              -- For outdoor sports: {temp: 72, conditions: 'Clear'}
  officials JSONB DEFAULT '[]',            -- [{name: 'John Doe', role: 'referee'}]
  game_data JSONB DEFAULT '{}',            -- Sport-specific: quarter_scores, play_by_play_available
  metadata JSONB DEFAULT '{}',             -- Playoff round, rivalry game, etc.
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  
  CONSTRAINT games_unique_sport_external UNIQUE(sport, external_id),
  CONSTRAINT games_valid_teams CHECK (home_team_id != away_team_id),
  CONSTRAINT games_valid_sport CHECK (sport IN ('basketball_nba', 'football_nfl', 'baseball_mlb', 'hockey_nhl')),
  CONSTRAINT games_valid_scores CHECK (
    (status != 'final') OR 
    (home_score IS NOT NULL AND away_score IS NOT NULL)
  )
);

CREATE INDEX idx_games_sport ON games(sport);
CREATE INDEX idx_games_season ON games(season_id);
CREATE INDEX idx_games_date ON games(game_date DESC);
CREATE INDEX idx_games_status ON games(status, game_date);
CREATE INDEX idx_games_teams ON games(home_team_id, away_team_id);
CREATE INDEX idx_games_home_team_date ON games(home_team_id, game_date);
CREATE INDEX idx_games_away_team_date ON games(away_team_id, game_date);
CREATE INDEX idx_games_live ON games(sport, status) WHERE status = 'in_progress';
CREATE INDEX idx_games_upcoming ON games(sport, game_date) WHERE status = 'scheduled';
CREATE INDEX idx_games_metadata ON games USING GIN(metadata);

COMMENT ON TABLE games IS 'Game information across all sports';
COMMENT ON COLUMN games.game_data IS 'JSONB: quarter_scores[], largest_lead, lead_changes';
COMMENT ON COLUMN games.metadata IS 'JSONB: playoff_round, series_game_number, rivalry_game';

