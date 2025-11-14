# 🏦 Gringotts Database Schema

**Database Name:** `nba_analytics`  
**Type:** MySQL  
**Purpose:** Single source of truth for NBA historical data, game statistics, and player performance

---

## 📊 Database Overview

Gringotts serves as the historical data vault for the 8-ball NBA betting platform. It stores comprehensive NBA data ingested directly from ESPN's API, including games, players, teams, statistics, and betting odds.

### Architecture Philosophy
```
Season (Top Level)
    ↓
Player Seasons + Teams (Season Participation)
    ↓
Games (Individual Matches)
    ↓
Stats & Odds (Game-Level Data)
```

### Connection Details
- **Host:** Configured via `DB_HOST` environment variable (default: `localhost`)
- **Port:** Configured via `DB_PORT` (default: `3306`)
- **Database:** `nba_analytics`
- **User:** Configured via `DB_USER` (default: `root`)
- **Password:** Configured via `DB_PASSWORD`
- **Engine:** SQLAlchemy with connection pooling (pool_size=10, max_overflow=20)

---

## 📋 Table Schemas

### 1. `seasons` - NBA Season Metadata
**Purpose:** Top-level hierarchy tracking NBA seasons with championship and date information

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `season_id` | VARCHAR(10) | PRIMARY KEY | Season identifier (e.g., "2024-25") |
| `season_name` | VARCHAR(50) | | Human-readable name (e.g., "2024-25 NBA Season") |
| `start_date` | DATE | NOT NULL | Season start date |
| `end_date` | DATE | NOT NULL | Season end date (regular season) |
| `playoff_start_date` | DATE | | Playoff start date |
| `finals_start_date` | DATE | | Finals start date |
| `champion_team_id` | INTEGER | FK → teams.team_id | Championship winner (NULL if incomplete) |
| `is_complete` | BOOLEAN | DEFAULT FALSE | Whether season has concluded |
| `total_games` | INTEGER | | Expected number of regular season games |
| `created_at` | DATETIME | | Record creation timestamp |
| `updated_at` | DATETIME | | Last update timestamp |

**Relationships:**
- `champion` → `teams` (many-to-one)
- `games` ← `games` (one-to-many)
- `player_seasons` ← `player_seasons` (one-to-many)

**Example Data:**
```sql
season_id | season_name        | start_date  | end_date    | is_complete | champion_team_id
----------|-------------------|-------------|-------------|-------------|------------------
2024-25   | 2024-25 NBA Season| 2024-10-22  | 2025-04-13  | FALSE       | NULL
2023-24   | 2023-24 NBA Season| 2023-10-24  | 2024-04-14  | TRUE        | 2 (BOS)
```

---

### 2. `teams` - NBA Franchises
**Purpose:** Permanent 30 NBA team records with conference and division information

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `team_id` | INTEGER | PRIMARY KEY AUTO_INCREMENT | Internal unique team ID |
| `team_abbreviation` | VARCHAR(10) | UNIQUE, NOT NULL, INDEXED | Team abbreviation (e.g., "LAL", "BOS") |
| `team_name` | VARCHAR(100) | NOT NULL | Team name (e.g., "Lakers") |
| `team_city` | VARCHAR(100) | | Team city (e.g., "Los Angeles") |
| `team_full_name` | VARCHAR(150) | | Full team name (e.g., "Los Angeles Lakers") |
| `conference` | VARCHAR(10) | | "East" or "West" |
| `division` | VARCHAR(50) | | Division name (e.g., "Pacific", "Atlantic") |
| `espn_team_id` | VARCHAR(20) | UNIQUE, INDEXED | ESPN's team identifier |
| `created_at` | DATETIME | | Record creation timestamp |
| `updated_at` | DATETIME | | Last update timestamp |

**Relationships:**
- `home_games` ← `games` (one-to-many via home_team_id)
- `away_games` ← `games` (one-to-many via away_team_id)
- `players` ← `players` (one-to-many, current roster)

**Indexes:**
- `team_abbreviation` (UNIQUE)
- `espn_team_id` (UNIQUE)

**Example Data:**
```sql
team_id | team_abbreviation | team_full_name           | conference | division
--------|-------------------|--------------------------|------------|----------
1       | LAL               | Los Angeles Lakers       | West       | Pacific
2       | BOS               | Boston Celtics           | East       | Atlantic
14      | GSW               | Golden State Warriors    | West       | Pacific
```

---

### 3. `players` - NBA Player Profiles
**Purpose:** Player biographical and career information with external ID mappings

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `player_id` | INTEGER | PRIMARY KEY AUTO_INCREMENT | Internal unique player ID |
| `player_name` | VARCHAR(150) | NOT NULL, INDEXED | Full player name (e.g., "LeBron James") |
| `espn_player_id` | VARCHAR(20) | UNIQUE, INDEXED | ESPN's player identifier |
| `nba_api_id` | VARCHAR(20) | UNIQUE, INDEXED | NBA API player identifier |
| `team_id` | INTEGER | FK → teams.team_id, NULL | Current team (NULL for free agents) |
| `position` | VARCHAR(10) | | Position (e.g., "PG", "SF", "C") |
| `jersey_number` | VARCHAR(5) | | Current jersey number |
| `height` | VARCHAR(10) | | Height (e.g., "6-8") |
| `weight` | INTEGER | | Weight in pounds |
| `birth_date` | DATE | | Date of birth |
| `college` | VARCHAR(100) | | College attended |
| `is_active` | BOOLEAN | DEFAULT TRUE | Currently active in NBA |
| `created_at` | DATETIME | | Record creation timestamp |
| `updated_at` | DATETIME | | Last update timestamp |

**Relationships:**
- `team` → `teams` (many-to-one)
- `game_stats` ← `player_game_stats` (one-to-many)
- `player_seasons` ← `player_seasons` (one-to-many)

**Indexes:**
- `player_name` (indexed for name searches)
- `espn_player_id` (UNIQUE)
- `nba_api_id` (UNIQUE)

**Note:** `is_active` reflects current status; historical activity tracked in `player_seasons`

---

### 4. `player_seasons` - Season Participation Tracking
**Purpose:** Links players to seasons with team affiliation and aggregated statistics

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | INTEGER | PRIMARY KEY AUTO_INCREMENT | Unique record ID |
| `player_id` | INTEGER | FK → players.player_id, NOT NULL, INDEXED | Player reference |
| `season_id` | VARCHAR(10) | FK → seasons.season_id, NOT NULL, INDEXED | Season reference |
| `team_id` | INTEGER | FK → teams.team_id, NULL | Team for this season (NULL for free agents) |
| `position` | VARCHAR(10) | | Position played this season |
| `jersey_number` | VARCHAR(5) | | Jersey number this season |
| `was_active` | BOOLEAN | DEFAULT TRUE | Was player active this season? |
| `games_played` | INTEGER | DEFAULT 0 | Total games played in season |
| `season_ppg` | FLOAT | | Average points per game |
| `season_rpg` | FLOAT | | Average rebounds per game |
| `season_apg` | FLOAT | | Average assists per game |
| `created_at` | DATETIME | | Record creation timestamp |
| `updated_at` | DATETIME | | Last update timestamp |

**Relationships:**
- `player` → `players` (many-to-one)
- `season` → `seasons` (many-to-one)
- `team` → `teams` (many-to-one)

**Constraints:**
- UNIQUE (`player_id`, `season_id`) - One record per player per season

**Indexes:**
- Composite: (`season_id`, `player_id`)
- Individual: `player_id`, `season_id`

**Use Cases:**
- Track player movement between teams across seasons
- Identify retired vs active players per season
- Season-by-season career statistics
- Historical roster reconstruction

---

### 5. `games` - NBA Game Records
**Purpose:** Every NBA game with teams, scores, and game context

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | INTEGER | PRIMARY KEY AUTO_INCREMENT | Internal unique game ID |
| `game_id` | VARCHAR(50) | UNIQUE, NOT NULL, INDEXED | ESPN game identifier |
| `season_id` | VARCHAR(10) | FK → seasons.season_id, NOT NULL, INDEXED | Season reference |
| `season_type` | VARCHAR(20) | DEFAULT "Regular Season" | "Regular Season", "Playoffs", etc. |
| `game_date` | DATE | NOT NULL, INDEXED | Game date (US Eastern time) |
| `game_time` | DATETIME | | Full game datetime (US Eastern) |
| `home_team_id` | INTEGER | FK → teams.team_id, NOT NULL | Home team reference |
| `away_team_id` | INTEGER | FK → teams.team_id, NOT NULL | Away team reference |
| `home_score` | INTEGER | | Home team final score |
| `away_score` | INTEGER | | Away team final score |
| `game_status` | VARCHAR(20) | | "Scheduled", "In Progress", "Final" |
| `venue` | VARCHAR(150) | | Venue name |
| `attendance` | INTEGER | | Attendance count |
| `created_at` | DATETIME | | Record creation timestamp |
| `updated_at` | DATETIME | | Last update timestamp |

**Relationships:**
- `season` → `seasons` (many-to-one)
- `home_team` → `teams` (many-to-one)
- `away_team` → `teams` (many-to-one)
- `player_stats` ← `player_game_stats` (one-to-many)
- `team_odds` ← `team_odds` (one-to-many)
- `player_props` ← `player_prop_odds` (one-to-many)

**Indexes:**
- `game_id` (UNIQUE)
- `season_id`
- `game_date`
- Composite: (`game_date`, `home_team_id`, `away_team_id`)

**Data Quality Guarantees:**
- No duplicate games (UNIQUE constraint on `game_id`)
- All games stored in US Eastern timezone (fixed timezone bug)
- Team mapping uses reliable abbreviations (fixed team ID bug)

---

### 6. `player_game_stats` - Player Box Score Statistics
**Purpose:** Individual player performance in each game with comprehensive statistics

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | INTEGER | PRIMARY KEY AUTO_INCREMENT | Unique stat record ID |
| `game_id` | VARCHAR(50) | FK → games.game_id, NOT NULL, INDEXED | Game reference |
| `player_id` | INTEGER | FK → players.player_id, NOT NULL, INDEXED | Player reference |
| `team_id` | INTEGER | FK → teams.team_id, NOT NULL | Team played for this game |
| **Basic Stats** | | | |
| `points` | INTEGER | DEFAULT 0 | Total points scored |
| `rebounds` | INTEGER | DEFAULT 0 | Total rebounds |
| `assists` | INTEGER | DEFAULT 0 | Total assists |
| `steals` | INTEGER | DEFAULT 0 | Total steals |
| `blocks` | INTEGER | DEFAULT 0 | Total blocks |
| `turnovers` | INTEGER | DEFAULT 0 | Total turnovers |
| **Shooting Stats** | | | |
| `field_goals_made` | INTEGER | DEFAULT 0 | Field goals made |
| `field_goals_attempted` | INTEGER | DEFAULT 0 | Field goals attempted |
| `three_pointers_made` | INTEGER | DEFAULT 0 | 3-pointers made |
| `three_pointers_attempted` | INTEGER | DEFAULT 0 | 3-pointers attempted |
| `free_throws_made` | INTEGER | DEFAULT 0 | Free throws made |
| `free_throws_attempted` | INTEGER | DEFAULT 0 | Free throws attempted |
| **Additional Stats** | | | |
| `offensive_rebounds` | INTEGER | DEFAULT 0 | Offensive rebounds |
| `defensive_rebounds` | INTEGER | DEFAULT 0 | Defensive rebounds |
| `personal_fouls` | INTEGER | DEFAULT 0 | Personal fouls |
| `minutes_played` | FLOAT | DEFAULT 0.0 | Minutes played |
| `plus_minus` | INTEGER | | Plus/minus rating |
| **Composite Stats** | | | |
| `points_rebounds_assists` | INTEGER | DEFAULT 0 | PRA (points + rebounds + assists) |
| **Advanced Statistics** | | | |
| `true_shooting_pct` | FLOAT | | TS% = PTS / (2 × (FGA + 0.44 × FTA)) |
| `effective_fg_pct` | FLOAT | | eFG% = (FGM + 0.5 × 3PM) / FGA |
| `usage_pct` | FLOAT | | USG% - Percentage of team plays used |
| `assist_pct` | FLOAT | | AST% - Percentage of teammate FGs assisted |
| `rebound_pct` | FLOAT | | REB% - Percentage of rebounds grabbed |
| `offensive_rebound_pct` | FLOAT | | ORB% - Offensive rebound percentage |
| `defensive_rebound_pct` | FLOAT | | DRB% - Defensive rebound percentage |
| `turnover_pct` | FLOAT | | TOV% - Turnover percentage |
| `steal_pct` | FLOAT | | STL% - Steal percentage |
| `block_pct` | FLOAT | | BLK% - Block percentage |
| `free_throw_rate` | FLOAT | | FTr = FTA / FGA |
| `points_per_possession` | FLOAT | | PPP - Points per possession |
| **Metadata** | | | |
| `starter` | BOOLEAN | DEFAULT FALSE | Was player a starter? |
| `created_at` | DATETIME | | Record creation timestamp |
| `updated_at` | DATETIME | | Last update timestamp |

**Relationships:**
- `game` → `games` (many-to-one)
- `player` → `players` (many-to-one)
- `team` → `teams` (many-to-one)

**Constraints:**
- UNIQUE (`game_id`, `player_id`) - One stat line per player per game

**Indexes:**
- Composite: (`player_id`, `game_id`)
- Individual: `game_id`, `player_id`

**Advanced Metrics Calculations:**
- All percentages calculated using standard NBA formulas
- Advanced stats optional (NULL if insufficient data)
- Calculated by `Gringotts/analytics/advanced_stats.py`

---

### 7. `team_game_stats` - Team Box Score Statistics
**Purpose:** Team-level performance and advanced metrics for each game

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | INTEGER | PRIMARY KEY AUTO_INCREMENT | Unique stat record ID |
| `game_id` | VARCHAR(50) | FK → games.game_id, NOT NULL, INDEXED | Game reference |
| `team_id` | INTEGER | FK → teams.team_id, NOT NULL, INDEXED | Team reference |
| `is_home` | BOOLEAN | NOT NULL | Home (TRUE) or Away (FALSE) |
| **Basic Stats** | | | |
| `points` | INTEGER | DEFAULT 0 | Total points scored |
| `field_goals_made` | INTEGER | DEFAULT 0 | Field goals made |
| `field_goals_attempted` | INTEGER | DEFAULT 0 | Field goals attempted |
| `three_pointers_made` | INTEGER | DEFAULT 0 | 3-pointers made |
| `three_pointers_attempted` | INTEGER | DEFAULT 0 | 3-pointers attempted |
| `free_throws_made` | INTEGER | DEFAULT 0 | Free throws made |
| `free_throws_attempted` | INTEGER | DEFAULT 0 | Free throws attempted |
| `offensive_rebounds` | INTEGER | DEFAULT 0 | Offensive rebounds |
| `defensive_rebounds` | INTEGER | DEFAULT 0 | Defensive rebounds |
| `rebounds` | INTEGER | DEFAULT 0 | Total rebounds |
| `assists` | INTEGER | DEFAULT 0 | Total assists |
| `steals` | INTEGER | DEFAULT 0 | Total steals |
| `blocks` | INTEGER | DEFAULT 0 | Total blocks |
| `turnovers` | INTEGER | DEFAULT 0 | Total turnovers |
| `personal_fouls` | INTEGER | DEFAULT 0 | Total fouls |
| **Advanced Statistics** | | | |
| `true_shooting_pct` | FLOAT | | TS% |
| `effective_fg_pct` | FLOAT | | eFG% |
| `turnover_pct` | FLOAT | | TOV% |
| `offensive_rebound_pct` | FLOAT | | ORB% |
| `defensive_rebound_pct` | FLOAT | | DRB% |
| `free_throw_rate` | FLOAT | | FTr |
| `possessions` | FLOAT | | Estimated team possessions |
| `pace` | FLOAT | | Possessions per 48 minutes |
| `offensive_rating` | FLOAT | | ORtg - Points per 100 possessions |
| `defensive_rating` | FLOAT | | DRtg - Points allowed per 100 possessions |
| `net_rating` | FLOAT | | Net Rating = ORtg - DRtg |
| `points_per_possession` | FLOAT | | PPP |
| `four_factors_score` | FLOAT | | Dean Oliver's Four Factors composite |
| `created_at` | DATETIME | | Record creation timestamp |
| `updated_at` | DATETIME | | Last update timestamp |

**Relationships:**
- Relationship to `games` and `teams` tables

**Constraints:**
- UNIQUE (`game_id`, `team_id`) - One stat line per team per game

**Indexes:**
- Composite: (`team_id`, `game_id`)
- Individual: `game_id`, `team_id`

---

### 8. `team_odds` - Team Betting Lines
**Purpose:** Historical team-level betting odds (moneyline, spread, totals)

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | INTEGER | PRIMARY KEY AUTO_INCREMENT | Unique odds record ID |
| `game_id` | VARCHAR(50) | FK → games.game_id, NOT NULL, INDEXED | Game reference |
| `team_id` | INTEGER | FK → teams.team_id, NOT NULL | Team reference |
| `bookmaker` | VARCHAR(100) | NOT NULL, INDEXED | Sportsbook name (e.g., "DraftKings") |
| **Moneyline** | | | |
| `moneyline` | INTEGER | | American odds format (e.g., -150, +120) |
| **Spread** | | | |
| `spread_points` | FLOAT | | Point spread (e.g., -5.5) |
| `spread_odds` | INTEGER | | American odds for spread (e.g., -110) |
| **Totals (Over/Under)** | | | |
| `total_points` | FLOAT | | Total line (e.g., 215.5) |
| `over_odds` | INTEGER | | Odds for over |
| `under_odds` | INTEGER | | Odds for under |
| **Metadata** | | | |
| `last_update` | DATETIME | | Last odds update timestamp |
| `is_active` | BOOLEAN | DEFAULT TRUE | Still accepting bets? |
| `created_at` | DATETIME | | Record creation timestamp |

**Relationships:**
- `game` → `games` (many-to-one)

**Indexes:**
- Composite: (`game_id`, `bookmaker`)
- Individual: `game_id`, `bookmaker`

**Note:** Historical odds for CLV tracking; live odds in TheWolf database

---

### 9. `player_prop_odds` - Player Prop Betting Lines
**Purpose:** Historical player prop odds (points, rebounds, assists, etc.)

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | INTEGER | PRIMARY KEY AUTO_INCREMENT | Unique odds record ID |
| `game_id` | VARCHAR(50) | FK → games.game_id, NOT NULL, INDEXED | Game reference |
| `player_id` | INTEGER | FK → players.player_id, NOT NULL, INDEXED | Player reference |
| `bookmaker` | VARCHAR(100) | NOT NULL | Sportsbook name |
| **Prop Details** | | | |
| `prop_type` | VARCHAR(50) | NOT NULL, INDEXED | "points", "rebounds", "assists", "threes", etc. |
| `line` | FLOAT | NOT NULL | Prop line (e.g., 25.5 points) |
| `over_odds` | INTEGER | | American odds for over |
| `under_odds` | INTEGER | | American odds for under |
| **Metadata** | | | |
| `last_update` | DATETIME | | Last odds update timestamp |
| `is_active` | BOOLEAN | DEFAULT TRUE | Still accepting bets? |
| `created_at` | DATETIME | | Record creation timestamp |

**Relationships:**
- `game` → `games` (many-to-one)

**Indexes:**
- Composite: (`player_id`, `prop_type`)
- Composite: (`game_id`, `player_id`, `bookmaker`)
- Individual: `game_id`, `player_id`, `prop_type`

**Supported Prop Types:**
- `points` - Total points scored
- `rebounds` - Total rebounds
- `assists` - Total assists
- `threes` - 3-pointers made
- `points_rebounds_assists` - PRA combo
- `steals`, `blocks`, `turnovers` - Defensive stats

---

### 10. `odds_api_mappings` - External ID Mappings
**Purpose:** Maps ESPN game IDs to Odds API event IDs for integration

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | INTEGER | PRIMARY KEY AUTO_INCREMENT | Unique mapping ID |
| `espn_game_id` | VARCHAR(50) | NOT NULL, INDEXED | ESPN game identifier |
| `odds_api_event_id` | VARCHAR(100) | NOT NULL, INDEXED | Odds API event identifier |
| `espn_team_id` | VARCHAR(20) | INDEXED | ESPN team ID |
| `odds_api_team_name` | VARCHAR(100) | | Odds API team name |
| `created_at` | DATETIME | | Mapping creation timestamp |

**Purpose:** Bridges Gringotts (ESPN data) with TheWolf (Odds API data)

**Indexes:**
- `espn_game_id`
- `odds_api_event_id`
- `espn_team_id`

---

## 🔗 Entity Relationships

### Primary Relationships

```
seasons (1) ──────< (N) games
           ──────< (N) player_seasons

teams (1) ──────< (N) players (current_team)
      (1) ──────< (N) games (home_team)
      (1) ──────< (N) games (away_team)
      (1) ──────< (N) player_seasons

players (1) ──────< (N) player_game_stats
        (1) ──────< (N) player_seasons
        (1) ──────< (N) player_prop_odds

games (1) ──────< (N) player_game_stats
      (1) ──────< (N) team_game_stats
      (1) ──────< (N) team_odds
      (1) ──────< (N) player_prop_odds

player_seasons (N) ──────> (1) players
               (N) ──────> (1) seasons
               (N) ──────> (1) teams
```

### Foreign Key Summary

| Child Table | Foreign Key | Parent Table | Constraint Name |
|-------------|-------------|--------------|-----------------|
| `seasons` | `champion_team_id` | `teams.team_id` | FK_seasons_champion |
| `players` | `team_id` | `teams.team_id` | FK_players_team |
| `player_seasons` | `player_id` | `players.player_id` | FK_player_seasons_player |
| `player_seasons` | `season_id` | `seasons.season_id` | FK_player_seasons_season |
| `player_seasons` | `team_id` | `teams.team_id` | FK_player_seasons_team |
| `games` | `season_id` | `seasons.season_id` | FK_games_season |
| `games` | `home_team_id` | `teams.team_id` | FK_games_home_team |
| `games` | `away_team_id` | `teams.team_id` | FK_games_away_team |
| `player_game_stats` | `game_id` | `games.game_id` | FK_player_stats_game |
| `player_game_stats` | `player_id` | `players.player_id` | FK_player_stats_player |
| `player_game_stats` | `team_id` | `teams.team_id` | FK_player_stats_team |
| `team_game_stats` | `game_id` | `games.game_id` | FK_team_stats_game |
| `team_game_stats` | `team_id` | `teams.team_id` | FK_team_stats_team |
| `team_odds` | `game_id` | `games.game_id` | FK_team_odds_game |
| `player_prop_odds` | `game_id` | `games.game_id` | FK_player_props_game |
| `player_prop_odds` | `player_id` | `players.player_id` | FK_player_props_player |

---

## 📊 Common Query Patterns

### Season Queries

```sql
-- Get current active season
SELECT * FROM seasons 
WHERE is_complete = FALSE 
ORDER BY season_id DESC 
LIMIT 1;

-- Get season champions
SELECT s.season_id, s.season_name, t.team_full_name as champion
FROM seasons s
LEFT JOIN teams t ON s.champion_team_id = t.team_id
WHERE s.is_complete = TRUE
ORDER BY s.season_id DESC;
```

### Player Queries

```sql
-- Player season averages
SELECT p.player_name, ps.season_id, t.team_abbreviation,
       ps.games_played, ps.season_ppg, ps.season_rpg, ps.season_apg
FROM player_seasons ps
JOIN players p ON ps.player_id = p.player_id
JOIN teams t ON ps.team_id = t.team_id
WHERE p.player_name = 'LeBron James'
ORDER BY ps.season_id DESC;

-- Player recent games (last 10)
SELECT g.game_date, pgs.points, pgs.rebounds, pgs.assists, pgs.minutes_played
FROM player_game_stats pgs
JOIN games g ON pgs.game_id = g.game_id
JOIN players p ON pgs.player_id = p.player_id
WHERE p.player_name = 'Stephen Curry'
ORDER BY g.game_date DESC
LIMIT 10;

-- Season leaders (min 20 games)
SELECT p.player_name, t.team_abbreviation, 
       ps.season_ppg, ps.season_rpg, ps.season_apg
FROM player_seasons ps
JOIN players p ON ps.player_id = p.player_id
JOIN teams t ON ps.team_id = t.team_id
WHERE ps.season_id = '2024-25' 
  AND ps.games_played >= 20
ORDER BY ps.season_ppg DESC
LIMIT 10;
```

### Game Queries

```sql
-- Today's games with scores
SELECT g.game_date, g.game_time,
       ht.team_abbreviation as home, g.home_score,
       at.team_abbreviation as away, g.away_score,
       g.game_status
FROM games g
JOIN teams ht ON g.home_team_id = ht.team_id
JOIN teams at ON g.away_team_id = at.team_id
WHERE g.game_date = CURDATE()
ORDER BY g.game_time;

-- Team schedule
SELECT g.game_date, g.game_time,
       CASE 
         WHEN g.home_team_id = 1 THEN CONCAT('vs ', at.team_abbreviation)
         ELSE CONCAT('@ ', ht.team_abbreviation)
       END as opponent,
       CONCAT(g.away_score, '-', g.home_score) as score
FROM games g
JOIN teams ht ON g.home_team_id = ht.team_id
JOIN teams at ON g.away_team_id = at.team_id
WHERE (g.home_team_id = 1 OR g.away_team_id = 1)
  AND g.season_id = '2024-25'
ORDER BY g.game_date DESC;
```

### Advanced Statistical Queries

```sql
-- Players with 30+ point games this season
SELECT p.player_name, COUNT(*) as games_30plus, AVG(pgs.points) as avg_pts
FROM player_game_stats pgs
JOIN players p ON pgs.player_id = p.player_id
JOIN games g ON pgs.game_id = g.game_id
WHERE pgs.points >= 30 
  AND g.season_id = '2024-25'
GROUP BY p.player_id
ORDER BY games_30plus DESC
LIMIT 10;

-- Team offensive/defensive ratings
SELECT t.team_abbreviation,
       AVG(tgs.offensive_rating) as avg_ortg,
       AVG(tgs.defensive_rating) as avg_drtg,
       AVG(tgs.net_rating) as avg_net_rtg
FROM team_game_stats tgs
JOIN teams t ON tgs.team_id = t.team_id
JOIN games g ON tgs.game_id = g.game_id
WHERE g.season_id = '2024-25'
  AND g.game_status = 'Final'
GROUP BY t.team_id
ORDER BY avg_net_rtg DESC;

-- Player shooting efficiency (min 10 games)
SELECT p.player_name, COUNT(*) as games,
       AVG(pgs.points) as ppg,
       AVG(pgs.true_shooting_pct) as ts_pct,
       AVG(pgs.effective_fg_pct) as efg_pct
FROM player_game_stats pgs
JOIN players p ON pgs.player_id = p.player_id
JOIN games g ON pgs.game_id = g.game_id
WHERE g.season_id = '2024-25'
  AND pgs.minutes_played > 20
GROUP BY p.player_id
HAVING COUNT(*) >= 10
ORDER BY ts_pct DESC
LIMIT 10;
```

---

## 🔧 Data Ingestion

### ESPN API Ingestion
**File:** `Gringotts/ingest/espn_stats.py`

**Daily Updates:**
- Fetches previous day's completed games
- Ingests box scores for all players
- Calculates advanced statistics
- Updates player_seasons aggregates

**Backfill Support:**
```bash
python scripts/ingest_gringotts_daily.py --backfill 2024-10-01 2024-11-01
```

### Key Fixes (Nov 2024)
1. **Team Mapping:** Now uses team abbreviations (reliable) instead of ESPN team IDs (unreliable)
2. **Timezone:** All dates converted to US Eastern time (no more date offset bugs)
3. **Idempotency:** Re-ingesting updates existing records, doesn't duplicate

---

## 🔐 Data Integrity

### Unique Constraints
| Table | Unique Constraint | Purpose |
|-------|-------------------|---------|
| `teams` | `team_abbreviation` | Prevent duplicate teams |
| `teams` | `espn_team_id` | Prevent duplicate ESPN mappings |
| `players` | `espn_player_id` | Prevent duplicate players |
| `player_seasons` | (`player_id`, `season_id`) | One record per player per season |
| `games` | `game_id` | Prevent duplicate games |
| `player_game_stats` | (`game_id`, `player_id`) | One stat line per player per game |
| `team_game_stats` | (`game_id`, `team_id`) | One stat line per team per game |

### Referential Integrity
- All foreign keys enforced
- Cascading deletes disabled (preserve historical data)
- NULL allowed for `team_id` in `players` (free agents)

### Data Quality Checks
- No games with NULL teams
- All completed games have scores
- Player stats sum to team stats (validated)
- All dates in US Eastern timezone

---

## 🚀 Performance Optimization

### Index Strategy
- All foreign keys indexed
- Frequent query columns indexed (game_date, player_name, team_abbreviation)
- Composite indexes for multi-column lookups
- Unique indexes double as constraints

### Connection Pooling
```python
from Gringotts.store.database import DatabaseManager

db = DatabaseManager()
# Pool size: 10
# Max overflow: 20
# Pre-ping: True (auto-reconnect)
# Pool recycle: 3600s (1 hour)
```

### Query Optimization
- Use eager loading (`joinedload`) for relationships
- Avoid N+1 queries
- Filter by indexed columns first
- Use LIMIT for pagination

**Example:**
```python
from sqlalchemy.orm import joinedload

# GOOD: Single query with eager loading
games = session.query(Game).options(
    joinedload(Game.home_team),
    joinedload(Game.away_team)
).all()

# BAD: N+1 queries
games = session.query(Game).all()
for game in games:
    print(game.home_team.team_name)  # New query each iteration!
```

---

## 🔌 Integration with TheWolf

**Gringotts provides historical context for TheWolf's market intelligence:**

| Gringotts Data | TheWolf Usage |
|----------------|---------------|
| `player_game_stats` | Predict player prop performance |
| `team_game_stats` | Calculate team matchup factors |
| `games` | Map Odds API games to ESPN games |
| `players` | Match player names to IDs |
| `player_seasons` | Determine active/inactive status |

**Integration Points:**
- `TheWolf/integration/game_mapper.py` - Maps game IDs
- `TheWolf/integration/player_matcher.py` - Matches player names
- Shared via Python service layer (not database joins)

---

## 📚 Service Layer

**Location:** `src/services/` and `src/repositories/`

### Available Services
- **PlayerService** - Player data and career stats
- **GameService** - Game data and schedules  
- **TeamService** - Team data and rosters
- **StatsService** - Statistical analysis and aggregations

### Repository Pattern
All database access goes through repositories:
- `PlayerRepository`
- `GameRepository`
- `TeamRepository`
- `OddsRepository`

**Benefits:**
- Abstraction from raw SQL
- Consistent error handling
- Testable business logic
- Type-safe DTOs

---

## 🛠️ Maintenance

### Regular Tasks
1. **Daily Ingestion** - Run at 6 AM via cron
2. **Update Player Seasons** - After each day's ingestion
3. **Verify Data Quality** - Check for missing games
4. **Monitor Database Size** - Archive old seasons if needed

### Database Initialization
```bash
# Create database
mysql -u root -p -e "CREATE DATABASE nba_analytics CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

# Initialize tables
python scripts/init_database.py

# Seed teams (30 NBA franchises)
python scripts/init_database.py --seed-teams

# Seed seasons
python scripts/seed_seasons.py
```

### Backup Strategy
```bash
# Daily backup (automated)
mysqldump -u root -p nba_analytics > backup_$(date +%Y%m%d).sql

# Restore from backup
mysql -u root -p nba_analytics < backup_20241101.sql
```

---

## 📖 Documentation References

- **[Gringotts README](README.md)** - Overview and usage guide
- **[DATABASE_HIERARCHY](DATABASE_HIERARCHY.md)** - Hierarchical structure explanation
- **[QUICKSTART](QUICKSTART.md)** - Setup and getting started
- **[SEASON_INFO](SEASON_INFO.md)** - Season metadata reference

---

## 🎯 Schema Philosophy

> **"Your predictions are only as good as your data."**

Gringotts is designed for:
1. **Accuracy** - Direct from ESPN API, zero manual entry
2. **Completeness** - Every game, every stat, back to 2000
3. **Reliability** - Timezone-aware, duplicate-protected
4. **Auditability** - Full history preserved, never deleted
5. **Performance** - Indexed for fast queries

**Key Principles:**
- Hierarchical structure (Season → Games → Stats)
- Immutable history (updates, not deletions)
- Foreign key integrity (no orphaned records)
- Defensive defaults (DEFAULT 0 for stats)
- Timezone consistency (all US Eastern)

---

**Database Version:** MySQL 8.0+  
**SQLAlchemy Version:** 2.0+  
**Character Set:** utf8mb4 (full Unicode support)  
**Collation:** utf8mb4_unicode_ci  

**Last Updated:** November 2024  
**Schema Status:** ✅ Production Ready

