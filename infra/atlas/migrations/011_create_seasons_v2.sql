-- Create seasons table (v2 - improved schema)
-- Based on ATLAS_DATABASE_DESIGN.md

CREATE TABLE seasons (
  season_id SERIAL PRIMARY KEY,
  sport VARCHAR(50) NOT NULL,
  season_year VARCHAR(20) NOT NULL,        -- '2024-25', '2024', '2024-25'
  season_type VARCHAR(20) DEFAULT 'regular', -- 'preseason', 'regular', 'playoffs'
  start_date DATE NOT NULL,
  end_date DATE NOT NULL,
  is_active BOOLEAN DEFAULT false,
  total_games INTEGER,                     -- Expected number of games
  metadata JSONB DEFAULT '{}',             -- Sport-specific: playoff format, divisions, etc.
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  
  CONSTRAINT seasons_unique_sport_year UNIQUE(sport, season_year, season_type),
  CONSTRAINT seasons_valid_dates CHECK (end_date >= start_date),
  CONSTRAINT seasons_valid_sport CHECK (sport IN ('basketball_nba', 'football_nfl', 'baseball_mlb', 'hockey_nhl'))
);

CREATE INDEX idx_seasons_sport_active ON seasons(sport, is_active) WHERE is_active = true;
CREATE INDEX idx_seasons_dates ON seasons(start_date, end_date);
CREATE INDEX idx_seasons_metadata ON seasons USING GIN(metadata);

COMMENT ON TABLE seasons IS 'Sports seasons across all leagues';
COMMENT ON COLUMN seasons.season_year IS 'Season identifier: 2024-25 for NBA, 2024 for NFL';
COMMENT ON COLUMN seasons.metadata IS 'JSONB: playoff_format, conference_structure, rule_changes';


