-- Create players table (v2 - improved schema)
-- Based on ATLAS_DATABASE_DESIGN.md

CREATE TABLE players (
  player_id SERIAL PRIMARY KEY,
  sport VARCHAR(50) NOT NULL,
  external_id VARCHAR(100),                -- ESPN player ID
  first_name VARCHAR(100),
  last_name VARCHAR(100) NOT NULL,
  full_name VARCHAR(200) NOT NULL,
  display_name VARCHAR(200),               -- Preferred display name (e.g., 'LeBron James')
  birth_date DATE,
  birth_city VARCHAR(100),
  birth_country VARCHAR(100),
  nationality VARCHAR(100),
  height VARCHAR(20),                      -- '6-9', '6\'9"'
  height_inches INTEGER,                   -- 81 (for calculations)
  weight INTEGER,                          -- pounds
  position VARCHAR(20),                    -- Current/primary position
  college VARCHAR(100),
  high_school VARCHAR(100),
  draft_year INTEGER,
  draft_round INTEGER,
  draft_pick INTEGER,
  draft_team_id INTEGER REFERENCES teams(team_id),
  headshot_url TEXT,
  jersey_number VARCHAR(10),               -- Current jersey (can change)
  status VARCHAR(20) DEFAULT 'active',     -- 'active', 'retired', 'free_agent', 'injured'
  metadata JSONB DEFAULT '{}',             -- Sport-specific: wingspan, vertical_leap, etc.
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  
  CONSTRAINT players_unique_sport_external UNIQUE(sport, external_id),
  CONSTRAINT players_valid_sport CHECK (sport IN ('basketball_nba', 'football_nfl', 'baseball_mlb', 'hockey_nhl')),
  CONSTRAINT players_valid_height CHECK (height_inches IS NULL OR (height_inches > 48 AND height_inches < 108)),
  CONSTRAINT players_valid_weight CHECK (weight IS NULL OR (weight > 100 AND weight < 500))
);

CREATE INDEX idx_players_sport ON players(sport);
CREATE INDEX idx_players_name ON players(sport, last_name, first_name);
CREATE INDEX idx_players_full_name ON players USING GIN(to_tsvector('english', full_name)); -- Full-text search
CREATE INDEX idx_players_status ON players(sport, status);
CREATE INDEX idx_players_draft ON players(sport, draft_year, draft_round, draft_pick);
CREATE INDEX idx_players_metadata ON players USING GIN(metadata);

COMMENT ON TABLE players IS 'Player biographical data across all sports';
COMMENT ON COLUMN players.metadata IS 'JSONB: wingspan, vertical_leap, combine_stats, awards[]';
COMMENT ON COLUMN players.height_inches IS 'Total height in inches for calculations';


