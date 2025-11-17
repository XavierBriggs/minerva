-- Create team_game_stats table (v2 - improved schema)
-- Based on ATLAS_DATABASE_DESIGN.md

CREATE TABLE team_game_stats (
  stat_id BIGSERIAL PRIMARY KEY,
  game_id INTEGER NOT NULL REFERENCES games(game_id),
  team_id INTEGER NOT NULL REFERENCES teams(team_id),
  is_home BOOLEAN NOT NULL,
  won BOOLEAN,                             -- NULL if game not final
  
  -- Basketball Team Stats
  points INTEGER,
  field_goals_made INTEGER,
  field_goals_attempted INTEGER,
  field_goal_pct NUMERIC(5,4),
  three_pointers_made INTEGER,
  three_pointers_attempted INTEGER,
  three_point_pct NUMERIC(5,4),
  free_throws_made INTEGER,
  free_throws_attempted INTEGER,
  free_throw_pct NUMERIC(5,4),
  rebounds INTEGER,
  offensive_rebounds INTEGER,
  defensive_rebounds INTEGER,
  assists INTEGER,
  steals INTEGER,
  blocks INTEGER,
  turnovers INTEGER,
  personal_fouls INTEGER,
  fast_break_points INTEGER,
  points_in_paint INTEGER,
  second_chance_points INTEGER,
  bench_points INTEGER,
  biggest_lead INTEGER,
  time_leading_seconds INTEGER,
  
  -- Advanced Team Metrics (Dean Oliver's Four Factors + More)
  -- Pace and Efficiency
  pace NUMERIC(6,2),                       -- Possessions per 48 minutes: 0.5 * ((Tm FGA + 0.44 * Tm FTA - Tm ORB + Tm TOV) + (Opp FGA + 0.44 * Opp FTA - Opp ORB + Opp TOV))
  possessions INTEGER,                     -- Estimated possessions in game
  offensive_rating NUMERIC(6,2),           -- ORtg: Points per 100 possessions
  defensive_rating NUMERIC(6,2),           -- DRtg: Points allowed per 100 possessions
  net_rating NUMERIC(6,2),                 -- NetRtg: ORtg - DRtg
  
  -- Shooting Efficiency (Four Factors #1)
  effective_fg_pct NUMERIC(5,4),           -- eFG%: (FGM + 0.5 * 3PM) / FGA
  true_shooting_pct NUMERIC(5,4),          -- TS%: PTS / (2 * (FGA + 0.44 * FTA))
  
  -- Turnover Rate (Four Factors #2)
  turnover_rate NUMERIC(5,4),              -- TOV%: Turnovers per 100 possessions
  
  -- Rebounding (Four Factors #3)
  offensive_rebound_pct NUMERIC(5,4),      -- ORB%: ORB / (ORB + Opp DRB)
  defensive_rebound_pct NUMERIC(5,4),      -- DRB%: DRB / (DRB + Opp ORB)
  total_rebound_pct NUMERIC(5,4),          -- TRB%: Total rebounds / Total available
  
  -- Free Throw Rate (Four Factors #4)
  free_throw_rate NUMERIC(5,4),            -- FTr: FTA / FGA
  
  -- Additional Advanced Metrics
  assist_to_turnover_ratio NUMERIC(5,4),   -- AST/TO: Assists / Turnovers
  steal_percentage NUMERIC(5,4),           -- STL%: Steals per 100 opponent possessions
  block_percentage NUMERIC(5,4),           -- BLK%: Blocks per 100 opponent FGA
  
  -- Extensible
  sport_specific_stats JSONB DEFAULT '{}',
  
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  
  CONSTRAINT team_game_stats_unique UNIQUE(game_id, team_id)
);

CREATE INDEX idx_team_game_stats_game ON team_game_stats(game_id);
CREATE INDEX idx_team_game_stats_team ON team_game_stats(team_id);
CREATE INDEX idx_team_game_stats_team_date ON team_game_stats(team_id, game_id);
CREATE INDEX idx_team_game_stats_sport_specific ON team_game_stats USING GIN(sport_specific_stats);

COMMENT ON TABLE team_game_stats IS 'Per-game team statistics across all sports';
COMMENT ON COLUMN team_game_stats.pace IS 'Estimated possessions per 48 minutes';

