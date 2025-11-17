-- Create teams table (v2 - improved schema)
-- Based on ATLAS_DATABASE_DESIGN.md

CREATE TABLE teams (
  team_id SERIAL PRIMARY KEY,
  sport VARCHAR(50) NOT NULL,
  external_id VARCHAR(100),                -- ESPN team ID
  abbreviation VARCHAR(10) NOT NULL,       -- 'LAL', 'BOS', 'KC'
  full_name VARCHAR(100) NOT NULL,         -- 'Los Angeles Lakers'
  short_name VARCHAR(50),                  -- 'Lakers'
  city VARCHAR(100),                       -- 'Los Angeles'
  state VARCHAR(50),                       -- 'California'
  conference VARCHAR(50),                  -- 'Western', 'AFC', 'National'
  division VARCHAR(50),                    -- 'Pacific', 'West', 'Central'
  venue_name VARCHAR(200),                 -- 'Crypto.com Arena'
  venue_capacity INTEGER,
  founded_year INTEGER,
  logo_url TEXT,
  colors JSONB DEFAULT '{}',               -- {primary: '#552583', secondary: '#FDB927'}
  social_media JSONB DEFAULT '{}',         -- {twitter: '@Lakers', instagram: '@lakers'}
  metadata JSONB DEFAULT '{}',             -- Sport-specific: championships, retired_numbers
  is_active BOOLEAN DEFAULT true,          -- For relocated/defunct teams
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  
  CONSTRAINT teams_unique_sport_external UNIQUE(sport, external_id),
  CONSTRAINT teams_unique_sport_abbr UNIQUE(sport, abbreviation),
  CONSTRAINT teams_valid_sport CHECK (sport IN ('basketball_nba', 'football_nfl', 'baseball_mlb', 'hockey_nhl'))
);

CREATE INDEX idx_teams_sport ON teams(sport) WHERE is_active = true;
CREATE INDEX idx_teams_abbreviation ON teams(sport, abbreviation);
CREATE INDEX idx_teams_name ON teams(sport, full_name);
CREATE INDEX idx_teams_conference_division ON teams(sport, conference, division);
CREATE INDEX idx_teams_metadata ON teams USING GIN(metadata);

COMMENT ON TABLE teams IS 'Teams across all sports leagues';
COMMENT ON COLUMN teams.metadata IS 'JSONB: championships[], retired_numbers[], historical_names[]';
COMMENT ON COLUMN teams.is_active IS 'False for relocated or defunct teams (e.g., Seattle SuperSonics)';


