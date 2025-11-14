# Database Hierarchical Structure

## Overview

The database  follows a **hierarchical structure** with `Season` at the top, making it easier to:
- Track player participation across seasons
- Query season-specific data efficiently
- Handle retired players correctly
- Store season metadata (champions, dates, etc.)

##   Hierarchy

```
Level 1: SEASONS (Top Entity)
└── seasons
    ├── season_id (PK) = "2025-26"
    ├── start_date, end_date
    ├── champion_team_id
    └── is_complete

Level 2: SEASON PARTICIPATION
├── teams (30 permanent teams)
│
└── player_seasons (NEW!)
    ├── player_id + season_id (unique)
    ├── team_id (which team they played for)
    ├── was_active (were they active this season?)
    ├── games_played
    └── season averages (ppg, rpg, apg)

Level 3: GAMES
└── games
    ├── game_id
    ├── season_id (FK → seasons) ← Now a real FK!
    └── team IDs

Level 4: STATS & ODDS
├── player_game_stats
├── team_game_stats
└── odds
```

## Tables

### `seasons` Table

Stores metadata for each NBA season:

```sql
CREATE TABLE seasons (
    season_id VARCHAR(10) PRIMARY KEY,  -- "2025-26"
    season_name VARCHAR(50),
    start_date DATE,
    end_date DATE,
    playoff_start_date DATE,
    champion_team_id INT,
    is_complete BOOLEAN,
    total_games INT
);
```

**Example Data:**
```sql
SELECT * FROM seasons ORDER BY season_id DESC LIMIT 3;

season_id | start_date  | end_date    | is_complete | champion_team_id
----------|-------------|-------------|-------------|------------------
2025-26   | 2025-10-21  | 2026-04-12  | FALSE       | NULL
2024-25   | 2024-10-22  | 2025-04-13  | FALSE       | NULL  
2023-24   | 2023-10-24  | 2024-04-14  | TRUE        | 2 (BOS)
```

### `player_seasons` Table

Tracks which players were active in which seasons:

```sql
CREATE TABLE player_seasons (
    id INT PRIMARY KEY AUTO_INCREMENT,
    player_id INT,
    season_id VARCHAR(10),
    team_id INT,
    was_active BOOLEAN,
    games_played INT,
    season_ppg FLOAT,
    season_rpg FLOAT,
    season_apg FLOAT,
    UNIQUE(player_id, season_id)
);
```

**Example Data:**
```sql
SELECT * FROM player_seasons WHERE player_id = 123;

player_id | season_id | team_id | was_active | games_played | season_ppg
----------|-----------|---------|------------|--------------|------------
123       | 2025-26   | 14      | TRUE       | 15           | 25.3
123       | 2024-25   | 14      | TRUE       | 71           | 28.7
123       | 2023-24   | 6       | TRUE       | 69           | 27.1
```
### Season Queries

```sql
-- Get current season
SELECT * FROM seasons WHERE is_complete = FALSE ORDER BY season_id DESC LIMIT 1;

-- Get all completed seasons with champions
SELECT s.season_id, s.start_date, s.end_date, t.team_abbreviation as champion
FROM seasons s
LEFT JOIN teams t ON s.champion_team_id = t.team_id
WHERE s.is_complete = TRUE
ORDER BY s.season_id DESC;

-- Games in a specific season
SELECT COUNT(*) as total_games FROM games WHERE season_id = '2024-25';
```

### Player Season Queries

```sql
-- Was LeBron active in 2020-21?
SELECT ps.*, p.player_name, t.team_abbreviation
FROM player_seasons ps
JOIN players p ON ps.player_id = p.player_id
JOIN teams t ON ps.team_id = t.team_id
WHERE p.player_name LIKE '%LeBron%'
  AND ps.season_id = '2020-21';

-- Players who played in multiple seasons
SELECT p.player_name, COUNT(*) as seasons_played, 
       MIN(ps.season_id) as first_season,
       MAX(ps.season_id) as last_season
FROM player_seasons ps
JOIN players p ON ps.player_id = p.player_id
WHERE ps.was_active = TRUE
GROUP BY p.player_id
HAVING COUNT(*) >= 3
ORDER BY seasons_played DESC;

-- Season leaders (from player_seasons aggregates)
SELECT p.player_name, t.team_abbreviation, ps.season_ppg
FROM player_seasons ps
JOIN players p ON ps.player_id = p.player_id
JOIN teams t ON ps.team_id = t.team_id
WHERE ps.season_id = '2024-25'
  AND ps.games_played >= 20
ORDER BY ps.season_ppg DESC
LIMIT 10;
```

### Retired Player Queries

```sql
-- Players who were active in 2019-20 but not in 2024-25
SELECT p.player_name, 
       ps_old.season_id as last_active_season,
       ps_old.team_id as last_team
FROM players p
JOIN player_seasons ps_old ON p.player_id = ps_old.player_id
LEFT JOIN player_seasons ps_new ON p.player_id = ps_new.player_id 
    AND ps_new.season_id = '2024-25'
WHERE ps_old.season_id = '2019-20'
  AND ps_new.player_id IS NULL
  AND ps_old.was_active = TRUE;
```

### Cross-Season Analysis

```sql
-- Player performance over time
SELECT ps.season_id, 
       ps.games_played,
       ps.season_ppg,
       ps.season_rpg,
       ps.season_apg
FROM player_seasons ps
JOIN players p ON ps.player_id = p.player_id
WHERE p.player_name = 'Stephen Curry'
ORDER BY ps.season_id;

-- Team's active roster per season
SELECT ps.season_id, COUNT(*) as roster_size
FROM player_seasons ps
WHERE ps.team_id = (SELECT team_id FROM teams WHERE team_abbreviation = 'LAL')
  AND ps.was_active = TRUE
GROUP BY ps.season_id
ORDER BY ps.season_id DESC;
```

##  Backward Compatibility

The `games.season_id` field remains as a **foreign key** to `seasons.season_id`, but:
-  Old queries still work (it's still a string field)
-  New queries can use the relationship: `game.season.is_complete`
-  No breaking changes to existing ingestion code

##  Benefits

**Before** (season as string):
```python
# Had to manually check dates
games_2024 = session.query(Game).filter_by(season_id="2024-25").all()
# No way to know if LeBron was active in 2020
```

**After** (season as entity):
```python
# Rich metadata
season = session.query(Season).filter_by(season_id="2024-25").first()
is_ongoing = not season.is_complete
champion = season.champion.team_name if season.champion else None

# Player history
lebron_2020 = session.query(PlayerSeason).filter_by(
    player_id=lebron.player_id,
    season_id="2020-21"
).first()
was_active = lebron_2020.was_active if lebron_2020 else False
```

##  Maintenance

### Update Season Status

```python
# Mark 2024-25 season as complete and set champion
from Gringotts.store.database import DatabaseManager
from Gringotts.store.models import Season, Team

db = DatabaseManager()
with db.get_session() as session:
    season = session.query(Season).filter_by(season_id="2024-25").first()
    lakers = session.query(Team).filter_by(team_abbreviation="LAL").first()
    
    season.is_complete = True
    season.champion_team_id = lakers.team_id
    session.commit()
```

### Refresh Player Seasons

```bash
# After ingesting new games, update player_seasons aggregates
python scripts/migrate_player_seasons.py --update
```

##  Next Steps

1. **Run migration** if you have existing data
2. **Use player_seasons** for accurate active/retired status
3. **Query by season** for historical analysis
4. **Update player status script** to use was_active per season

---

**The hierarchy is now proper and scalable!** 
