# 🏦 Gringotts: NBA Data Vault

**The foundation of truth.**

Gringotts is the historical data layer for the 8-ball NBA betting platform. While TheWolf tracks market intelligence and 8-ball makes predictions, Gringotts is the **single source of truth** for what actually happened on the court.

## Philosophy

```
Gringotts = Data Vault (what happened)
8-ball = Prediction Engine (what will happen)
TheWolf = Market Intelligence (where is the edge)
```

**The Critical Foundation:** You can't predict the future without understanding the past. Gringotts stores every game, every stat, every player performance—ingested directly from ESPN's API with zero manual intervention.

## Key Features

### 1. Automated ESPN Data Ingestion
- Daily game and stats updates from ESPN API
- Player box score data (points, rebounds, assists, shooting %, etc.)
- Team statistics and advanced metrics
- Historical backfill support (2000+ games)
- **Fixed:** Team mapping now uses abbreviations (reliable across seasons)
- **Fixed:** Date conversion to US Eastern time (no more timezone bugs)

### 2. Hierarchical Data Model
- **Seasons** → Track NBA seasons with metadata (champions, dates, status)
- **Teams** → 30 permanent NBA franchises
- **Players** → Player profiles with career tracking
- **PlayerSeasons** → Season-by-season player participation and stats
- **Games** → Every NBA game with scores and context
- **PlayerGameStats** → Individual player performance per game
- **Odds** → Historical betting lines (integrated with TheWolf)

### 3. Advanced Statistics
- True Shooting Percentage (TS%)
- Effective Field Goal Percentage (eFG%)
- Usage Rate, Assist %, Rebound %
- Plus/Minus tracking
- Points per possession
- Custom composite stats (PRA, PR, PA, RA)

### 4. Service Layer Architecture
- **PlayerService** - Player data and career stats
- **GameService** - Game data and schedules
- **TeamService** - Team data and rosters
- **StatsService** - Statistical analysis and aggregations
- **OddsService** - Historical betting line data

### 5. Data Quality Guarantees
- ✅ **No duplicate games** (unique constraint on ESPN game_id)
- ✅ **No duplicate player stats** (unique constraint on game_id + player_id)
- ✅ **Automatic updates** (re-ingesting updates existing records)
- ✅ **Timezone-aware** (games stored in US Eastern time)
- ✅ **Reliable team mapping** (uses ESPN abbreviations, not fragile IDs)

## Architecture

```
Gringotts/
├── store/                     # Database layer (MySQL)
│   ├── models.py              # SQLAlchemy ORM models
│   └── database.py            # Database manager & connection pool
│
├── ingest/                    # Data collection from ESPN
│   ├── espn_stats.py          # ESPN API ingestion (FIXED!)
│   ├── player_mapping.py      # ESPN → DB player ID mapping
│   └── odds_api.py            # Odds API integration (legacy)
│
├── analytics/                 # Statistical analysis
│   └── advanced_stats.py      # Advanced metrics calculation
│
├── DATABASE_HIERARCHY.md      # Database structure guide
├── SEASON_INFO.md             # Season metadata reference
└── QUICKSTART.md              # Setup and usage guide
```

**Note:** Service and repository layers are in `src/` (shared across modules):
- `src/services/` - Business logic layer
- `src/repositories/` - Data access layer

## Installation

### 1. Database Setup

Gringotts uses MySQL for relational data storage:

```bash
# Create database
mysql -u root -p -e "CREATE DATABASE nba_analytics CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

# Set environment variables in .env
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_password
DB_NAME=nba_analytics

# Initialize database tables
python scripts/init_database.py
```

### 2. Verify Installation

```python
from Gringotts.store.database import DatabaseManager

db = DatabaseManager()
if db.test_connection():
    print("✅ Gringotts ready!")
```

### 3. Ingest Data

```bash
# Fetch yesterday's games
python scripts/ingest_gringotts_daily.py

# Backfill historical data
python scripts/ingest_gringotts_daily.py --backfill 2025-10-22 2025-11-01
```

## Usage

### Basic: Query Recent Games

```python
from Gringotts.store.database import DatabaseManager
from Gringotts.store.models import Game, Team
from sqlalchemy.orm import joinedload

db = DatabaseManager()

with db.get_session() as session:
    # Get recent games
    games = (
        session.query(Game)
        .options(joinedload(Game.home_team), joinedload(Game.away_team))
        .order_by(Game.game_date.desc())
        .limit(10)
        .all()
    )

    for game in games:
        print(f"{game.game_date}: {game.away_team.team_abbreviation} @ {game.home_team.team_abbreviation} ({game.away_score}-{game.home_score})")
```

### Intermediate: Player Performance History

```python
from src.services import PlayerService

player_service = PlayerService()

# Get player's recent stats
lebron_stats = player_service.get_player_recent_stats(
    player_name="LeBron James",
    last_n_games=10
)

for stat in lebron_stats:
    print(f"{stat.game_date}: {stat.points}pts {stat.rebounds}reb {stat.assists}ast")

# Season averages
season_avg = player_service.get_season_averages(
    player_name="LeBron James",
    season_id="2025-26"
)
print(f"PPG: {season_avg['ppg']:.1f}, RPG: {season_avg['rpg']:.1f}, APG: {season_avg['apg']:.1f}")
```

### Advanced: Team Statistics

```python
from src.services import TeamService, GameService

team_service = TeamService()
game_service = GameService()

# Get team's upcoming opponent
lakers = team_service.get_team_by_abbreviation("LAL")
next_game = game_service.get_next_game_for_team(lakers.team_id)

# Analyze opponent's defense
opponent_stats = team_service.get_defensive_stats(
    team_id=next_game.away_team_id,
    last_n_games=10
)

print(f"Opponent allows {opponent_stats['opp_ppg']:.1f} PPG")
print(f"Opponent defensive rating: {opponent_stats['def_rating']:.1f}")
```

### Expert: Custom Queries

```python
from Gringotts.store.database import DatabaseManager
from Gringotts.store.models import PlayerGameStats, Player, Game
from sqlalchemy import func, and_

db = DatabaseManager()

with db.get_session() as session:
    # Players with 30+ point games this season
    high_scorers = (
        session.query(
            Player.player_name,
            func.count(PlayerGameStats.id).label('games_30plus')
        )
        .join(PlayerGameStats)
        .join(Game)
        .filter(
            and_(
                PlayerGameStats.points >= 30,
                Game.season_id == '2025-26'
            )
        )
        .group_by(Player.player_id)
        .order_by(func.count(PlayerGameStats.id).desc())
        .limit(10)
        .all()
    )

    for player, count in high_scorers:
        print(f"{player}: {count} games with 30+ points")
```

## Integration with 8-ball & TheWolf

Gringotts provides the historical foundation for predictions and market analysis:

```python
from Gringotts.store.database import DatabaseManager
from src.services import PlayerService, GameService
from src.modeling.pipelines import PlayerPropsPredictor
from TheWolf.services import MarketService

# Your betting pipeline
class BettingPipeline:
    def __init__(self):
        # Gringotts: Historical truth
        self.player_service = PlayerService()
        self.game_service = GameService()

        # 8-ball: Predictions
        self.predictor = PlayerPropsPredictor()

        # TheWolf: Market intelligence
        self.market_service = MarketService()

    def analyze_player_prop(self, player_name, prop_type, game_id):
        # STEP 1: Get historical context from Gringotts
        recent_stats = self.player_service.get_player_recent_stats(
            player_name=player_name,
            last_n_games=10
        )

        season_avg = self.player_service.get_season_averages(
            player_name=player_name,
            season_id="2025-26"
        )

        # STEP 2: Make prediction with 8-ball
        prediction = self.predictor.predict({
            'player_name': player_name,
            'prop_type': prop_type,
            'recent_stats': recent_stats,
            'season_avg': season_avg
        })

        # STEP 3: Compare to market with TheWolf
        market_line = self.market_service.get_best_odds(
            market_id=f"{player_name}_{prop_type}_{game_id}"
        )

        return {
            'prediction': prediction,
            'market_line': market_line,
            'historical_avg': season_avg[prop_type],
            'recent_trend': recent_stats[-5:].mean()
        }
```

## Data Quality & Recent Fixes

### 🐛 Bug Fixes (Nov 2025)

**1. Team Mapping Bug** Fixed
- **Problem:** ESPN team IDs were hardcoded incorrectly (ID "11" mapped to HOU instead of IND)
- **Impact:** ALL games had wrong teams stored
- **Fix:** Refactored to use team **abbreviations** from ESPN API instead of unreliable ESPN IDs
- **File:** `Gringotts/ingest/espn_stats.py:443-462`

**2. Timezone Bug** Fixed
- **Problem:** Game dates stored in UTC instead of US local time (off by 1 day)
- **Impact:** Games appeared on wrong dates (Oct 30 games showed as Oct 31)
- **Fix:** Convert ESPN UTC timestamps to US Eastern time before storing
- **File:** `Gringotts/ingest/espn_stats.py:467-496`

### Data Integrity

**Protected Against Duplicates:**
- ✅ `games.game_id` - UNIQUE constraint
- ✅ `player_game_stats(game_id, player_id)` - UNIQUE constraint
- Re-running ingestion **updates** existing records, doesn't duplicate

**Safe Operations:**
```bash
# Safe to re-run daily ingestion (updates existing games)
python scripts/ingest_gringotts_daily.py

# Safe to backfill (won't create duplicates)
python scripts/ingest_gringotts_daily.py --backfill 2025-10-22
```

## Database Schema

### Core Tables

**seasons** - NBA season metadata
```sql
CREATE TABLE seasons (
    season_id VARCHAR(10) PRIMARY KEY,     -- "2025-26"
    start_date DATE,
    end_date DATE,
    champion_team_id INT,
    is_complete BOOLEAN
);
```

**teams** - 30 NBA franchises
```sql
CREATE TABLE teams (
    team_id INT PRIMARY KEY,
    team_abbreviation VARCHAR(10) UNIQUE,  -- "LAL", "BOS", etc.
    team_name VARCHAR(100),                -- "Los Angeles Lakers"
    conference VARCHAR(10),                -- "West" or "East"
    division VARCHAR(50)
);
```

**players** - Player profiles
```sql
CREATE TABLE players (
    player_id INT PRIMARY KEY,
    player_name VARCHAR(150),
    espn_player_id VARCHAR(20) UNIQUE,     -- ESPN's ID
    team_id INT,                           -- Current team
    position VARCHAR(10),
    is_active BOOLEAN
);
```

**player_seasons** - Season-by-season tracking
```sql
CREATE TABLE player_seasons (
    player_id INT,
    season_id VARCHAR(10),
    team_id INT,                           -- Team for that season
    was_active BOOLEAN,
    games_played INT,
    season_ppg FLOAT,
    UNIQUE(player_id, season_id)
);
```

**games** - Every NBA game
```sql
CREATE TABLE games (
    id INT PRIMARY KEY AUTO_INCREMENT,
    game_id VARCHAR(50) UNIQUE,            -- ESPN game ID
    season_id VARCHAR(10),
    game_date DATE,
    home_team_id INT,
    away_team_id INT,
    home_score INT,
    away_score INT,
    game_status VARCHAR(20)
);
```

**player_game_stats** - Player performance per game
```sql
CREATE TABLE player_game_stats (
    id INT PRIMARY KEY AUTO_INCREMENT,
    game_id VARCHAR(50),
    player_id INT,
    team_id INT,
    points INT,
    rebounds INT,
    assists INT,
    -- ... 20+ more stat fields
    UNIQUE(game_id, player_id)
);
```

See [DATABASE_HIERARCHY.md](DATABASE_HIERARCHY.md) for full schema documentation.

## Configuration

Gringotts uses environment variables (`.env` file):

```bash
# Database
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_password
DB_NAME=nba_analytics

# Logging
LOG_LEVEL=INFO
LOG_FILE=logs/gringotts.log
```

## Automation

### Daily Ingestion (Cron)

```bash
# Add to crontab (runs at 6 AM daily)
0 6 * * * cd /path/to/8-ball && /path/to/venv/bin/python scripts/ingest_gringotts_daily.py >> logs/cron.log 2>&1
```

See `crontab.example` for full automation setup.

### Pipeline Integration

```bash
# Master daily pipeline (recommended)
python scripts/master_daily_pipeline.py

# This runs:
# 1. Gringotts ingestion (games & stats)
# 2. TheWolf odds collection
# 3. 8-ball predictions
# 4. Betting recommendations
```

## Key Queries

### Season Leaders

```sql
SELECT p.player_name, ps.season_ppg, ps.season_rpg, ps.season_apg
FROM player_seasons ps
JOIN players p ON ps.player_id = p.player_id
WHERE ps.season_id = '2025-26' AND ps.games_played >= 20
ORDER BY ps.season_ppg DESC
LIMIT 10;
```

### Team Schedule

```sql
SELECT g.game_date,
       CASE WHEN g.home_team_id = 14 THEN away.team_abbreviation
            ELSE home.team_abbreviation END as opponent,
       CASE WHEN g.home_team_id = 14 THEN 'vs' ELSE '@' END as location
FROM games g
JOIN teams home ON g.home_team_id = home.team_id
JOIN teams away ON g.away_team_id = away.team_id
WHERE (g.home_team_id = 14 OR g.away_team_id = 14)
  AND g.season_id = '2025-26'
ORDER BY g.game_date;
```

### Player Performance Trends

```sql
SELECT g.game_date, pgs.points, pgs.rebounds, pgs.assists
FROM player_game_stats pgs
JOIN games g ON pgs.game_id = g.game_id
JOIN players p ON pgs.player_id = p.player_id
WHERE p.player_name = 'LeBron James'
  AND g.season_id = '2025-26'
ORDER BY g.game_date DESC
LIMIT 10;
```

## Troubleshooting

### "No MySQL credentials found"
Set environment variables in `.env`:
```bash
DB_HOST=localhost
DB_USER=root
DB_PASSWORD=your_password
DB_NAME=nba_analytics
```

### "ModuleNotFoundError: No module named 'scipy'"
Install missing dependencies:
```bash
pip install -r requirements.txt
```

### "No games found for date"
Check:
1. Is it an NBA off-day? (check ESPN.com)
2. Are you querying the correct date? (dates are in US Eastern time)
3. Run with `--backfill` to fetch specific date range

### "Team not found in database: XYZ"
Run team initialization:
```bash
python scripts/init_database.py --seed-teams
```

## Performance Considerations

### Indexing
- ✅ All foreign keys indexed
- ✅ Composite indexes on (game_date, team_id)
- ✅ Unique indexes prevent duplicates

### Query Optimization
```python
# GOOD: Use eager loading
games = session.query(Game).options(
    joinedload(Game.home_team),
    joinedload(Game.away_team)
).all()

# BAD: N+1 queries
games = session.query(Game).all()
for game in games:
    print(game.home_team.team_abbreviation)  # Each access = new query!
```

### Connection Pooling
- Pool size: 10 connections
- Max overflow: 20 connections
- Pre-ping enabled (auto-reconnect)

## Next Steps

1. **Ingest historical data**: `python scripts/ingest_gringotts_daily.py --backfill 2024-10-01`
2. **Set up automation**: Add to crontab for daily updates
3. **Explore data**: Use services or SQL to analyze trends
4. **Integrate with 8-ball**: Use historical data for predictions
5. **Track performance**: Monitor CLV with TheWolf

## Documentation

- **[DATABASE_HIERARCHY.md](DATABASE_HIERARCHY.md)** - Full schema reference
- **[SEASON_INFO.md](SEASON_INFO.md)** - Season metadata guide
- **[QUICKSTART.md](QUICKSTART.md)** - Setup and usage guide

## Philosophy

> "Garbage in, garbage out. Your predictions are only as good as your data."

Gringotts exists because **data quality is everything**. You can have the most sophisticated ML model in the world, but if your training data is wrong (wrong teams, wrong dates, missing games), your predictions will be worthless.

Our commitment:
1. **Accuracy** - Direct from ESPN API, zero manual entry
2. **Completeness** - Every game, every stat, back to 2000
3. **Reliability** - Timezone-aware, duplicate-protected, automatically updated
4. **Auditability** - Full history preserved, never deleted

If your data is clean, your models have a chance. Gringotts is that foundation.

---

**Store wisely. 🏦**
