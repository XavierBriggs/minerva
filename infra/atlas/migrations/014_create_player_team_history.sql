-- Create player_team_history table (NEW - temporal tracking)
-- Based on ATLAS_DATABASE_DESIGN.md
-- CRITICAL for tracking player-team relationships over time (trades, signings)

CREATE TABLE player_team_history (
  history_id SERIAL PRIMARY KEY,
  player_id INTEGER NOT NULL REFERENCES players(player_id),
  team_id INTEGER NOT NULL REFERENCES teams(team_id),
  season_id INTEGER NOT NULL REFERENCES seasons(season_id),
  start_date DATE NOT NULL,
  end_date DATE,                           -- NULL if currently with team
  jersey_number VARCHAR(10),
  position VARCHAR(20),
  contract_type VARCHAR(50),               -- 'rookie', 'veteran', 'two-way', etc.
  is_starter BOOLEAN DEFAULT false,
  games_played INTEGER DEFAULT 0,
  metadata JSONB DEFAULT '{}',             -- Trade details, signing info
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  
  CONSTRAINT player_team_history_valid_dates CHECK (end_date IS NULL OR end_date >= start_date),
  CONSTRAINT player_team_history_unique UNIQUE(player_id, team_id, start_date)
);

CREATE INDEX idx_player_team_history_player ON player_team_history(player_id, season_id);
CREATE INDEX idx_player_team_history_team ON player_team_history(team_id, season_id);
CREATE INDEX idx_player_team_history_season ON player_team_history(season_id);
CREATE INDEX idx_player_team_history_dates ON player_team_history(start_date, end_date);
CREATE INDEX idx_player_team_history_current ON player_team_history(player_id) WHERE end_date IS NULL;

COMMENT ON TABLE player_team_history IS 'Temporal tracking of player-team relationships';
COMMENT ON COLUMN player_team_history.end_date IS 'NULL indicates player currently on team';
COMMENT ON COLUMN player_team_history.metadata IS 'JSONB: trade_details, signing_bonus, contract_years';

-- Function to get player's team at a specific date
CREATE OR REPLACE FUNCTION get_player_team_at_date(p_player_id INTEGER, p_date DATE)
RETURNS INTEGER AS $$
  SELECT team_id
  FROM player_team_history
  WHERE player_id = p_player_id
    AND start_date <= p_date
    AND (end_date IS NULL OR end_date >= p_date)
  LIMIT 1;
$$ LANGUAGE SQL STABLE;

COMMENT ON FUNCTION get_player_team_at_date IS 'Get the team a player was on at a specific date';


