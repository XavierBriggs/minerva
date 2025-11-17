-- Create materialized views for performance optimization
-- Based on ATLAS_DATABASE_DESIGN.md

-- Materialized view for player season averages (performance optimization)
CREATE MATERIALIZED VIEW player_season_averages AS
SELECT 
  pgs.player_id,
  g.season_id,
  g.sport,
  COUNT(*) as games_played,
  AVG(pgs.minutes_played) as avg_minutes,
  AVG(pgs.points) as ppg,
  AVG(pgs.rebounds) as rpg,
  AVG(pgs.assists) as apg,
  AVG(pgs.steals) as spg,
  AVG(pgs.blocks) as bpg,
  AVG(pgs.field_goal_pct) as fg_pct,
  AVG(pgs.three_point_pct) as three_pt_pct,
  AVG(pgs.free_throw_pct) as ft_pct,
  AVG(pgs.true_shooting_pct) as ts_pct,
  SUM(pgs.points) as total_points,
  SUM(pgs.rebounds) as total_rebounds,
  SUM(pgs.assists) as total_assists
FROM player_game_stats pgs
JOIN games g ON pgs.game_id = g.game_id
WHERE pgs.active = true
GROUP BY pgs.player_id, g.season_id, g.sport;

CREATE UNIQUE INDEX idx_player_season_averages ON player_season_averages(player_id, season_id);
CREATE INDEX idx_player_season_averages_ppg ON player_season_averages(season_id, ppg DESC);
CREATE INDEX idx_player_season_averages_games ON player_season_averages(season_id, games_played DESC);

COMMENT ON MATERIALIZED VIEW player_season_averages IS 'Pre-calculated player season averages for fast queries. Refresh nightly or after each game day.';

-- Note: To refresh the materialized view (should be done nightly or after game days):
-- REFRESH MATERIALIZED VIEW CONCURRENTLY player_season_averages;


