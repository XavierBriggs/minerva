-- Create odds_mappings table (v2 - improved schema)
-- Based on ATLAS_DATABASE_DESIGN.md

CREATE TABLE odds_mappings (
  mapping_id SERIAL PRIMARY KEY,
  sport VARCHAR(50) NOT NULL,
  minerva_game_id INTEGER REFERENCES games(game_id),
  minerva_team_id INTEGER REFERENCES teams(team_id),
  minerva_player_id INTEGER REFERENCES players(player_id),
  alexandria_event_id VARCHAR(100) NOT NULL,
  alexandria_participant_name VARCHAR(200),
  mapping_type VARCHAR(20) NOT NULL,       -- 'game', 'team', 'player'
  confidence NUMERIC(3,2) DEFAULT 1.00,    -- Matching confidence (0.00-1.00)
  match_method VARCHAR(50),                -- 'exact', 'fuzzy', 'manual'
  verified BOOLEAN DEFAULT false,          -- Manual verification flag
  verified_by VARCHAR(100),
  verified_at TIMESTAMP,
  metadata JSONB DEFAULT '{}',             -- Matching details, aliases
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  
  CONSTRAINT odds_mappings_valid_type CHECK (mapping_type IN ('game', 'team', 'player')),
  CONSTRAINT odds_mappings_valid_confidence CHECK (confidence >= 0 AND confidence <= 1),
  CONSTRAINT odds_mappings_has_minerva_id CHECK (
    (mapping_type = 'game' AND minerva_game_id IS NOT NULL) OR
    (mapping_type = 'team' AND minerva_team_id IS NOT NULL) OR
    (mapping_type = 'player' AND minerva_player_id IS NOT NULL)
  )
);

CREATE INDEX idx_odds_mappings_minerva_game ON odds_mappings(minerva_game_id);
CREATE INDEX idx_odds_mappings_minerva_team ON odds_mappings(minerva_team_id);
CREATE INDEX idx_odds_mappings_minerva_player ON odds_mappings(minerva_player_id);
CREATE INDEX idx_odds_mappings_alexandria ON odds_mappings(alexandria_event_id);
CREATE INDEX idx_odds_mappings_type ON odds_mappings(mapping_type, sport);
CREATE INDEX idx_odds_mappings_unverified ON odds_mappings(sport, verified) WHERE verified = false;

COMMENT ON TABLE odds_mappings IS 'Links Minerva sports data to Alexandria odds data';
COMMENT ON COLUMN odds_mappings.confidence IS 'Matching confidence score (1.00 = exact match)';
COMMENT ON COLUMN odds_mappings.match_method IS 'How the mapping was created';


