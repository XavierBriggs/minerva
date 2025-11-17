# Atlas Database Design - Multi-Sport Analytics

**Database**: PostgreSQL 14+  
**Schema**: `atlas`  
**Purpose**: Sports analytics data storage for ML training and real-time insights  
**Design Philosophy**: Multi-sport ready, performance optimized, ML-friendly

---

## 🎯 Design Principles (Based on Industry Best Practices)

### 1. **Normalization (3NF) with Strategic Denormalization**
- Reduce data redundancy through proper normalization
- Denormalize only for critical read-heavy queries
- Use materialized views for complex aggregations

### 2. **Temporal Data Management**
- Track player-team relationships over time
- Maintain historical accuracy for roster changes
- Support point-in-time queries for any season

### 3. **Partitioning Strategy**
- Partition large tables by season/year for performance
- Improves query speed and data management
- Enables efficient archival of old data

### 4. **Multi-Sport Extensibility**
- Sport-agnostic core schema
- JSONB for sport-specific attributes
- Consistent patterns across all sports

### 5. **Performance Optimization**
- Strategic indexing on query patterns
- Composite indexes for multi-column queries
- Partial indexes for filtered queries
- GIN indexes for JSONB fields

### 6. **Data Integrity**
- Foreign key constraints for referential integrity
- Check constraints for data validation
- Unique constraints to prevent duplicates
- NOT NULL constraints for required fields

### 7. **Scalability**
- Designed for horizontal scaling
- Support for read replicas
- Connection pooling ready
- Prepared for sharding by sport

---

## 📊 Core Schema

### Table: `seasons`
**Purpose**: Track sports seasons across all sports  
**Partitioning**: None (small table)  
**Estimated Rows**: ~100 per sport

```sql
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
```

---

### Table: `teams`
**Purpose**: Store team information across all sports  
**Partitioning**: None  
**Estimated Rows**: ~30-50 per sport

```sql
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
```

---

### Table: `players`
**Purpose**: Store player biographical information  
**Partitioning**: None  
**Estimated Rows**: ~5,000+ per sport

```sql
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
```

---

### Table: `player_team_history`
**Purpose**: Track player-team relationships over time (CRITICAL for historical accuracy)  
**Partitioning**: By season_id  
**Estimated Rows**: ~10,000+ per season

```sql
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
```

---

### Table: `games`
**Purpose**: Store game information  
**Partitioning**: By season_id (recommended for large datasets)  
**Estimated Rows**: ~1,500 per season per sport

```sql
CREATE TABLE games (
  game_id SERIAL PRIMARY KEY,
  sport VARCHAR(50) NOT NULL,
  external_id VARCHAR(100),                -- ESPN game ID
  season_id INTEGER NOT NULL REFERENCES seasons(season_id),
  home_team_id INTEGER NOT NULL REFERENCES teams(team_id),
  away_team_id INTEGER NOT NULL REFERENCES teams(team_id),
  game_date TIMESTAMP NOT NULL,
  scheduled_time TIMESTAMP NOT NULL,       -- Original scheduled time
  actual_start_time TIMESTAMP,             -- Actual game start
  venue VARCHAR(200),
  attendance INTEGER,
  game_status VARCHAR(20) NOT NULL,        -- 'scheduled', 'in_progress', 'final', 'postponed', 'cancelled'
  period INTEGER,                          -- Current quarter/inning/period
  time_remaining VARCHAR(20),              -- '5:23', 'End of 3rd', 'Halftime'
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
    (game_status != 'final') OR 
    (home_score IS NOT NULL AND away_score IS NOT NULL)
  )
);

CREATE INDEX idx_games_sport ON games(sport);
CREATE INDEX idx_games_season ON games(season_id);
CREATE INDEX idx_games_date ON games(game_date DESC);
CREATE INDEX idx_games_status ON games(game_status, game_date);
CREATE INDEX idx_games_teams ON games(home_team_id, away_team_id);
CREATE INDEX idx_games_home_team_date ON games(home_team_id, game_date);
CREATE INDEX idx_games_away_team_date ON games(away_team_id, game_date);
CREATE INDEX idx_games_live ON games(sport, game_status) WHERE game_status = 'in_progress';
CREATE INDEX idx_games_upcoming ON games(sport, game_date) WHERE game_status = 'scheduled';
CREATE INDEX idx_games_metadata ON games USING GIN(metadata);

COMMENT ON TABLE games IS 'Game information across all sports';
COMMENT ON COLUMN games.game_data IS 'JSONB: quarter_scores[], largest_lead, lead_changes';
COMMENT ON COLUMN games.metadata IS 'JSONB: playoff_round, series_game_number, rivalry_game';

-- Partitioning example (for production with large datasets)
-- CREATE TABLE games_2024_25 PARTITION OF games FOR VALUES IN ('2024-25');
-- CREATE TABLE games_2023_24 PARTITION OF games FOR VALUES IN ('2023-24');
```

---

### Table: `player_game_stats`
**Purpose**: Store per-game player statistics  
**Partitioning**: By season_id (CRITICAL for performance)  
**Estimated Rows**: ~300,000+ per season (13 players × 2 teams × 1,230 games)

```sql
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
  
  -- Advanced Metrics (calculated)
  field_goal_pct NUMERIC(5,4),
  three_point_pct NUMERIC(5,4),
  free_throw_pct NUMERIC(5,4),
  true_shooting_pct NUMERIC(5,4),
  effective_fg_pct NUMERIC(5,4),
  usage_rate NUMERIC(5,4),
  offensive_rating NUMERIC(6,2),
  defensive_rating NUMERIC(6,2),
  
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

-- Materialized view for season averages (performance optimization)
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
  AVG(pgs.true_shooting_pct) as ts_pct
FROM player_game_stats pgs
JOIN games g ON pgs.game_id = g.game_id
WHERE pgs.active = true
GROUP BY pgs.player_id, g.season_id, g.sport;

CREATE UNIQUE INDEX idx_player_season_averages ON player_season_averages(player_id, season_id);
CREATE INDEX idx_player_season_averages_ppg ON player_season_averages(season_id, ppg DESC);

-- Refresh strategy: nightly or after each game day
-- REFRESH MATERIALIZED VIEW CONCURRENTLY player_season_averages;
```

---

### Table: `team_game_stats`
**Purpose**: Store per-game team statistics  
**Partitioning**: By season_id  
**Estimated Rows**: ~3,000 per season (2 teams × 1,230 games)

```sql
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
  
  -- Advanced Team Metrics
  pace NUMERIC(6,2),                       -- Possessions per 48 minutes
  offensive_rating NUMERIC(6,2),
  defensive_rating NUMERIC(6,2),
  net_rating NUMERIC(6,2),
  effective_fg_pct NUMERIC(5,4),
  true_shooting_pct NUMERIC(5,4),
  
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
```

---

### Table: `odds_mappings`
**Purpose**: Link Minerva games/teams to Alexandria (Mercury) events  
**Partitioning**: None  
**Estimated Rows**: ~5,000 per season

```sql
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
```

---

### Table: `backfill_jobs`
**Purpose**: Track historical data backfill operations  
**Partitioning**: None  
**Estimated Rows**: ~1,000

```sql
CREATE TABLE backfill_jobs (
  job_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  sport VARCHAR(50) NOT NULL,
  job_type VARCHAR(20) NOT NULL,           -- 'season', 'date_range', 'game'
  season_id VARCHAR(20),
  start_date DATE,
  end_date DATE,
  game_ids TEXT[],                         -- For specific game backfills
  status VARCHAR(20) NOT NULL DEFAULT 'queued',
  status_message TEXT,
  progress_current INTEGER DEFAULT 0,
  progress_total INTEGER DEFAULT 0,
  items_processed INTEGER DEFAULT 0,
  items_failed INTEGER DEFAULT 0,
  last_error TEXT,
  retry_count INTEGER DEFAULT 0,
  max_retries INTEGER DEFAULT 3,
  priority INTEGER DEFAULT 5,              -- 1-10 (10 = highest)
  created_at TIMESTAMP DEFAULT NOW(),
  started_at TIMESTAMP,
  completed_at TIMESTAMP,
  updated_at TIMESTAMP DEFAULT NOW(),
  
  CONSTRAINT backfill_jobs_valid_type CHECK (job_type IN ('season', 'date_range', 'game')),
  CONSTRAINT backfill_jobs_valid_status CHECK (status IN ('queued', 'running', 'completed', 'failed', 'cancelled')),
  CONSTRAINT backfill_jobs_valid_dates CHECK (end_date IS NULL OR end_date >= start_date)
);

CREATE INDEX idx_backfill_jobs_status ON backfill_jobs(status, priority DESC) WHERE status IN ('queued', 'running');
CREATE INDEX idx_backfill_jobs_sport ON backfill_jobs(sport, created_at DESC);
CREATE INDEX idx_backfill_jobs_created ON backfill_jobs(created_at DESC);

COMMENT ON TABLE backfill_jobs IS 'Tracks historical data backfill operations';
COMMENT ON COLUMN backfill_jobs.priority IS 'Job priority (1-10, higher = more urgent)';
```

---

### Table: `backfill_job_events`
**Purpose**: Detailed event log for backfill jobs  
**Partitioning**: None  
**Estimated Rows**: ~10,000+

```sql
CREATE TABLE backfill_job_events (
  event_id BIGSERIAL PRIMARY KEY,
  job_id UUID NOT NULL REFERENCES backfill_jobs(job_id) ON DELETE CASCADE,
  event_type VARCHAR(50) NOT NULL,         -- 'started', 'progress', 'error', 'completed'
  message TEXT,
  details JSONB DEFAULT '{}',              -- Error details, progress info
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_backfill_job_events_job ON backfill_job_events(job_id, created_at);
CREATE INDEX idx_backfill_job_events_type ON backfill_job_events(event_type, created_at);

COMMENT ON TABLE backfill_job_events IS 'Detailed event log for backfill operations';
```

---

## 🔧 Database Functions & Triggers

### Auto-update `updated_at` timestamp
```sql
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply to all tables with updated_at
CREATE TRIGGER update_teams_updated_at BEFORE UPDATE ON teams
  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_players_updated_at BEFORE UPDATE ON players
  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_games_updated_at BEFORE UPDATE ON games
  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ... apply to other tables
```

### Calculate shooting percentages automatically
```sql
CREATE OR REPLACE FUNCTION calculate_shooting_percentages()
RETURNS TRIGGER AS $$
BEGIN
  -- Field Goal %
  IF NEW.field_goals_attempted > 0 THEN
    NEW.field_goal_pct = NEW.field_goals_made::NUMERIC / NEW.field_goals_attempted;
  END IF;
  
  -- 3-Point %
  IF NEW.three_pointers_attempted > 0 THEN
    NEW.three_point_pct = NEW.three_pointers_made::NUMERIC / NEW.three_pointers_attempted;
  END IF;
  
  -- Free Throw %
  IF NEW.free_throws_attempted > 0 THEN
    NEW.free_throw_pct = NEW.free_throws_made::NUMERIC / NEW.free_throws_attempted;
  END IF;
  
  -- True Shooting % = PTS / (2 * (FGA + 0.44 * FTA))
  IF NEW.field_goals_attempted > 0 OR NEW.free_throws_attempted > 0 THEN
    NEW.true_shooting_pct = NEW.points::NUMERIC / 
      (2 * (NEW.field_goals_attempted + 0.44 * NEW.free_throws_attempted));
  END IF;
  
  -- Effective FG% = (FGM + 0.5 * 3PM) / FGA
  IF NEW.field_goals_attempted > 0 THEN
    NEW.effective_fg_pct = (NEW.field_goals_made + 0.5 * NEW.three_pointers_made)::NUMERIC / 
      NEW.field_goals_attempted;
  END IF;
  
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER calculate_player_shooting BEFORE INSERT OR UPDATE ON player_game_stats
  FOR EACH ROW EXECUTE FUNCTION calculate_shooting_percentages();
```

---

## 📈 Performance Optimization

### Connection Pooling (Application Level)
```go
// Recommended pgxpool settings
config, _ := pgxpool.ParseConfig(databaseURL)
config.MaxConns = 25                    // Max connections
config.MinConns = 5                     // Min idle connections
config.MaxConnLifetime = time.Hour      // Recycle connections
config.MaxConnIdleTime = 30 * time.Minute
config.HealthCheckPeriod = time.Minute
```

### Query Optimization Examples
```sql
-- GOOD: Uses index on (sport, game_status, game_date)
SELECT * FROM games 
WHERE sport = 'basketball_nba' 
  AND game_status = 'in_progress'
ORDER BY game_date DESC;

-- GOOD: Uses index on (player_id, game_id)
SELECT * FROM player_game_stats 
WHERE player_id = 123 
ORDER BY game_id DESC 
LIMIT 10;

-- BAD: Full table scan (avoid SELECT *)
SELECT * FROM player_game_stats WHERE points > 30;

-- GOOD: Select only needed columns
SELECT player_id, game_id, points, rebounds, assists 
FROM player_game_stats 
WHERE points > 30;
```

### Maintenance Tasks
```sql
-- Run weekly
VACUUM ANALYZE player_game_stats;
VACUUM ANALYZE games;

-- Reindex monthly
REINDEX TABLE player_game_stats;

-- Update statistics
ANALYZE player_game_stats;
```

---

## 🔒 Security & Access Control

### Role-Based Access
```sql
-- Read-only role for analytics/ML
CREATE ROLE minerva_readonly;
GRANT CONNECT ON DATABASE atlas TO minerva_readonly;
GRANT USAGE ON SCHEMA public TO minerva_readonly;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO minerva_readonly;

-- Application role (read/write)
CREATE ROLE minerva_app;
GRANT CONNECT ON DATABASE atlas TO minerva_app;
GRANT USAGE ON SCHEMA public TO minerva_app;
GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA public TO minerva_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO minerva_app;

-- Admin role (full access)
CREATE ROLE minerva_admin WITH SUPERUSER;
```

---

## 📊 Monitoring Queries

### Table sizes
```sql
SELECT 
  schemaname,
  tablename,
  pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

### Index usage
```sql
SELECT 
  schemaname,
  tablename,
  indexname,
  idx_scan as index_scans,
  pg_size_pretty(pg_relation_size(indexrelid)) AS index_size
FROM pg_stat_user_indexes
ORDER BY idx_scan ASC;
```

### Slow queries (enable pg_stat_statements extension)
```sql
SELECT 
  query,
  calls,
  total_time,
  mean_time,
  max_time
FROM pg_stat_statements
ORDER BY mean_time DESC
LIMIT 10;
```

---

## 🚀 Migration Strategy

### Phase 1: Core Tables
1. seasons, teams, players
2. games
3. player_team_history

### Phase 2: Stats Tables
1. player_game_stats
2. team_game_stats

### Phase 3: Integration
1. odds_mappings
2. backfill_jobs

### Phase 4: Optimization
1. Materialized views
2. Partitioning (if needed)
3. Additional indexes based on query patterns

---

**Last Updated**: November 14, 2025  
**Schema Version**: 1.0  
**Next Review**: After Phase 1 implementation

