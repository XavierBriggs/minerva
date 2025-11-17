-- Create player_game_stats table (v2 - improved schema)
-- Based on ATLAS_DATABASE_DESIGN.md

CREATE TABLE player_game_stats (
  stat_id BIGSERIAL PRIMARY KEY,
  game_id INTEGER NOT NULL REFERENCES games(game_id),
  player_id INTEGER NOT NULL REFERENCES players(player_id),
  team_id INTEGER NOT NULL REFERENCES teams(team_id),
  
  -- Universal Stats (applicable across sports)
  minutes_played NUMERIC(5,2),             -- 36.5
  seconds_played INTEGER,                  -- For precise calculations
  starter BOOLEAN DEFAULT false,
  active BOOLEAN DEFAULT true,             -- Did player suit up?
  dnp_reason VARCHAR(100),                 -- 'Injury', 'Coach Decision', etc.
  
  -- Basketball-Specific Stats
  points INTEGER,
  rebounds INTEGER,
  offensive_rebounds INTEGER,
  defensive_rebounds INTEGER,
  assists INTEGER,
  steals INTEGER,
  blocks INTEGER,
  turnovers INTEGER,
  personal_fouls INTEGER,
  field_goals_made INTEGER,
  field_goals_attempted INTEGER,
  three_pointers_made INTEGER,
  three_pointers_attempted INTEGER,
  free_throws_made INTEGER,
  free_throws_attempted INTEGER,
  plus_minus INTEGER,
  
  -- Advanced Metrics (calculated from box score data)
  -- Shooting Efficiency
  field_goal_pct NUMERIC(5,4),                -- FG%: FGM / FGA
  three_point_pct NUMERIC(5,4),              -- 3P%: 3PM / 3PA
  free_throw_pct NUMERIC(5,4),               -- FT%: FTM / FTA
  true_shooting_pct NUMERIC(5,4),            -- TS%: PTS / (2 * (FGA + 0.44 * FTA))
  effective_fg_pct NUMERIC(5,4),             -- eFG%: (FGM + 0.5 * 3PM) / FGA
  
  -- Usage and Involvement
  usage_rate NUMERIC(5,4),                   -- USG%: % of team plays used while on floor
  assist_percentage NUMERIC(5,4),            -- AST%: % of teammate FGs assisted
  turnover_percentage NUMERIC(5,4),          -- TOV%: Turnovers per 100 plays
  
  -- Rebounding
  offensive_rebound_pct NUMERIC(5,4),        -- ORB%: % of available offensive rebounds
  defensive_rebound_pct NUMERIC(5,4),        -- DRB%: % of available defensive rebounds
  total_rebound_pct NUMERIC(5,4),            -- TRB%: % of available rebounds
  
  -- Impact Metrics
  offensive_rating NUMERIC(6,2),             -- ORtg: Points produced per 100 possessions
  defensive_rating NUMERIC(6,2),             -- DRtg: Points allowed per 100 possessions
  net_rating NUMERIC(6,2),                   -- NetRtg: ORtg - DRtg
  player_efficiency_rating NUMERIC(6,2),     -- PER: Overall efficiency metric
  
  -- Game Score (John Hollinger's metric)
  game_score NUMERIC(6,2),                   -- GmSc: Single-game performance metric
  
  -- Box Plus/Minus components (requires team context)
  box_plus_minus NUMERIC(6,2),               -- BPM: Contribution per 100 possessions
  
  -- Extensible for other sports
  sport_specific_stats JSONB DEFAULT '{}', -- Football: passing_yards, rushing_yards, etc.
  
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  
  CONSTRAINT player_game_stats_unique UNIQUE(game_id, player_id),
  CONSTRAINT player_game_stats_valid_minutes CHECK (minutes_played IS NULL OR (minutes_played >= 0 AND minutes_played <= 60)),
  CONSTRAINT player_game_stats_valid_shooting CHECK (
    (field_goals_made IS NULL OR field_goals_attempted IS NULL) OR 
    (field_goals_made <= field_goals_attempted)
  )
);

CREATE INDEX idx_player_game_stats_game ON player_game_stats(game_id);
CREATE INDEX idx_player_game_stats_player ON player_game_stats(player_id);
CREATE INDEX idx_player_game_stats_team ON player_game_stats(team_id);
CREATE INDEX idx_player_game_stats_player_date ON player_game_stats(player_id, game_id); -- For player game logs
CREATE INDEX idx_player_game_stats_points ON player_game_stats(player_id, points DESC) WHERE points IS NOT NULL; -- For career highs
CREATE INDEX idx_player_game_stats_sport_specific ON player_game_stats USING GIN(sport_specific_stats);

COMMENT ON TABLE player_game_stats IS 'Per-game player statistics across all sports';
COMMENT ON COLUMN player_game_stats.sport_specific_stats IS 'JSONB: Football: {passing_yards, rushing_yards, touchdowns}';
COMMENT ON COLUMN player_game_stats.dnp_reason IS 'Reason if player did not play (DNP)';

