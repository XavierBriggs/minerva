-- Create triggers and functions
-- Based on ATLAS_DATABASE_DESIGN.md

-- Auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply to all tables with updated_at
CREATE TRIGGER update_seasons_updated_at BEFORE UPDATE ON seasons
  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_teams_updated_at BEFORE UPDATE ON teams
  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_players_updated_at BEFORE UPDATE ON players
  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_player_team_history_updated_at BEFORE UPDATE ON player_team_history
  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_games_updated_at BEFORE UPDATE ON games
  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_player_game_stats_updated_at BEFORE UPDATE ON player_game_stats
  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_team_game_stats_updated_at BEFORE UPDATE ON team_game_stats
  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_odds_mappings_updated_at BEFORE UPDATE ON odds_mappings
  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_backfill_jobs_updated_at BEFORE UPDATE ON backfill_jobs
  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Auto-calculate advanced player stats
CREATE OR REPLACE FUNCTION calculate_player_advanced_stats()
RETURNS TRIGGER AS $$
BEGIN
  -- Basic Shooting Percentages
  IF NEW.field_goals_attempted IS NOT NULL AND NEW.field_goals_attempted > 0 THEN
    NEW.field_goal_pct = NEW.field_goals_made::NUMERIC / NEW.field_goals_attempted;
  END IF;
  
  IF NEW.three_pointers_attempted IS NOT NULL AND NEW.three_pointers_attempted > 0 THEN
    NEW.three_point_pct = NEW.three_pointers_made::NUMERIC / NEW.three_pointers_attempted;
  END IF;
  
  IF NEW.free_throws_attempted IS NOT NULL AND NEW.free_throws_attempted > 0 THEN
    NEW.free_throw_pct = NEW.free_throws_made::NUMERIC / NEW.free_throws_attempted;
  END IF;
  
  -- True Shooting % = PTS / (2 * (FGA + 0.44 * FTA))
  IF NEW.points IS NOT NULL AND NEW.field_goals_attempted IS NOT NULL AND NEW.free_throws_attempted IS NOT NULL THEN
    IF (NEW.field_goals_attempted + 0.44 * NEW.free_throws_attempted) > 0 THEN
      NEW.true_shooting_pct = NEW.points::NUMERIC / 
        (2 * (NEW.field_goals_attempted + 0.44 * NEW.free_throws_attempted));
    END IF;
  END IF;
  
  -- Effective FG% = (FGM + 0.5 * 3PM) / FGA
  IF NEW.field_goals_attempted IS NOT NULL AND NEW.field_goals_attempted > 0 THEN
    NEW.effective_fg_pct = (NEW.field_goals_made + 0.5 * COALESCE(NEW.three_pointers_made, 0))::NUMERIC / 
      NEW.field_goals_attempted;
  END IF;
  
  -- Game Score = PTS + 0.4*FGM - 0.7*FGA - 0.4*(FTA-FTM) + 0.7*ORB + 0.3*DRB + STL + 0.7*AST + 0.7*BLK - 0.4*PF - TOV
  IF NEW.points IS NOT NULL THEN
    NEW.game_score = NEW.points 
      + 0.4 * COALESCE(NEW.field_goals_made, 0)
      - 0.7 * COALESCE(NEW.field_goals_attempted, 0)
      - 0.4 * (COALESCE(NEW.free_throws_attempted, 0) - COALESCE(NEW.free_throws_made, 0))
      + 0.7 * COALESCE(NEW.offensive_rebounds, 0)
      + 0.3 * COALESCE(NEW.defensive_rebounds, 0)
      + COALESCE(NEW.steals, 0)
      + 0.7 * COALESCE(NEW.assists, 0)
      + 0.7 * COALESCE(NEW.blocks, 0)
      - 0.4 * COALESCE(NEW.personal_fouls, 0)
      - COALESCE(NEW.turnovers, 0);
  END IF;
  
  -- Net Rating (if both ORtg and DRtg are available)
  IF NEW.offensive_rating IS NOT NULL AND NEW.defensive_rating IS NOT NULL THEN
    NEW.net_rating = NEW.offensive_rating - NEW.defensive_rating;
  END IF;
  
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER calculate_player_stats BEFORE INSERT OR UPDATE ON player_game_stats
  FOR EACH ROW EXECUTE FUNCTION calculate_player_advanced_stats();

-- Auto-calculate advanced team stats
CREATE OR REPLACE FUNCTION calculate_team_advanced_stats()
RETURNS TRIGGER AS $$
BEGIN
  -- Basic Shooting Percentages
  IF NEW.field_goals_attempted IS NOT NULL AND NEW.field_goals_attempted > 0 THEN
    NEW.field_goal_pct = NEW.field_goals_made::NUMERIC / NEW.field_goals_attempted;
  END IF;
  
  IF NEW.three_pointers_attempted IS NOT NULL AND NEW.three_pointers_attempted > 0 THEN
    NEW.three_point_pct = NEW.three_pointers_made::NUMERIC / NEW.three_pointers_attempted;
  END IF;
  
  IF NEW.free_throws_attempted IS NOT NULL AND NEW.free_throws_attempted > 0 THEN
    NEW.free_throw_pct = NEW.free_throws_made::NUMERIC / NEW.free_throws_attempted;
  END IF;
  
  -- True Shooting % = PTS / (2 * (FGA + 0.44 * FTA))
  IF NEW.points IS NOT NULL AND NEW.field_goals_attempted IS NOT NULL AND NEW.free_throws_attempted IS NOT NULL THEN
    IF (NEW.field_goals_attempted + 0.44 * NEW.free_throws_attempted) > 0 THEN
      NEW.true_shooting_pct = NEW.points::NUMERIC / 
        (2 * (NEW.field_goals_attempted + 0.44 * NEW.free_throws_attempted));
    END IF;
  END IF;
  
  -- Effective FG% = (FGM + 0.5 * 3PM) / FGA
  IF NEW.field_goals_attempted IS NOT NULL AND NEW.field_goals_attempted > 0 THEN
    NEW.effective_fg_pct = (NEW.field_goals_made + 0.5 * COALESCE(NEW.three_pointers_made, 0))::NUMERIC / 
      NEW.field_goals_attempted;
  END IF;
  
  -- Free Throw Rate = FTA / FGA
  IF NEW.field_goals_attempted IS NOT NULL AND NEW.field_goals_attempted > 0 THEN
    NEW.free_throw_rate = NEW.free_throws_attempted::NUMERIC / NEW.field_goals_attempted;
  END IF;
  
  -- Assist to Turnover Ratio
  IF NEW.turnovers IS NOT NULL AND NEW.turnovers > 0 AND NEW.assists IS NOT NULL THEN
    NEW.assist_to_turnover_ratio = NEW.assists::NUMERIC / NEW.turnovers;
  END IF;
  
  -- Offensive/Defensive Rating (if possessions available)
  IF NEW.possessions IS NOT NULL AND NEW.possessions > 0 THEN
    IF NEW.points IS NOT NULL THEN
      NEW.offensive_rating = (NEW.points::NUMERIC / NEW.possessions) * 100;
    END IF;
  END IF;
  
  -- Net Rating
  IF NEW.offensive_rating IS NOT NULL AND NEW.defensive_rating IS NOT NULL THEN
    NEW.net_rating = NEW.offensive_rating - NEW.defensive_rating;
  END IF;
  
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER calculate_team_stats BEFORE INSERT OR UPDATE ON team_game_stats
  FOR EACH ROW EXECUTE FUNCTION calculate_team_advanced_stats();

COMMENT ON FUNCTION update_updated_at_column IS 'Automatically updates the updated_at timestamp on row modification';
COMMENT ON FUNCTION calculate_player_advanced_stats IS 'Automatically calculates advanced player statistics from box score data';
COMMENT ON FUNCTION calculate_team_advanced_stats IS 'Automatically calculates advanced team statistics including Four Factors';

