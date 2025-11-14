# NBA Analytics Pipeline - Quick Start Guide

## Prerequisites

- Python 3.12 or higher
- MySQL 8.0 or higher
- The Odds API key (for betting data)

## Installation

### 1. Clone and Setup

```bash
git clone https://github.com/XavierBriggs/8-ball.git
cd 8-ball
```

### 2. Install Dependencies

```bash
pip install -r requirements.txt
```

### 3. Configure Environment

Create a `.env` file in the project root:

```bash
# Database Configuration
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_password
DB_NAME=nba_analytics

# API Keys
ODDS_API_KEY=your_api_key_here
```

### 4. Create MySQL Database

First, connect to MySQL:

```bash
mysql -u root -p
```

Then inside MySQL, run:

```sql
CREATE DATABASE nba_analytics CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
exit;
```

Or as a one-liner from terminal:

```bash
mysql -u root -p -e "CREATE DATABASE nba_analytics CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
```

### 5. Initialize Database

```bash
python scripts/init_database.py
```

This will:
- Create all database tables
- Seed 30 NBA teams
- Seed current seasons (2024-25, 2025-26)

## Usage

### Daily Updates

Run the daily pipeline to fetch latest games and stats:

```bash
python -m src.pipeline.daily_update
```

Or use the shell script:

```bash
./run_daily_update.sh
```

### Historical Data Backfill

Fetch data from start of 25-26:
 ```bash 
 python -m src.pipeline.daily_update --backfill --start-date 2025-10-21 
 ```
Fetch data for past seasons:

```bash
# List available seasons
python scripts/backfill_season.py --list

# Backfill entire season
python scripts/backfill_season.py --season 2024-25

# Backfill date range
python scripts/backfill_season.py --season 2023-24 --start-date 2023-12-01 --end-date 2023-12-31
```

### Seed Historical Seasons

Add season metadata for all available years:

```bash
python scripts/seed_seasons.py
```

### Migrate Player Seasons

After ingesting game data, create player-season records:

```bash
# Dry run to see what would be created
python scripts/migrate_player_seasons.py

# Actually create records
python scripts/migrate_player_seasons.py --update
```

### Update Player Status

Mark inactive/retired players:

```bash
# Show current statistics
python scripts/update_player_status.py --stats

# Check who would be marked inactive (dry run)
python scripts/update_player_status.py --check

# Mark players as inactive (no games in 2 years)
python scripts/update_player_status.py --update

# Mark known retired players
python scripts/update_player_status.py --known-retired --update
```

## Viewing Data

### MySQL Command Line

```bash
mysql -u root -p nba_analytics
```

```sql
-- View teams
SELECT * FROM teams;

-- View seasons
SELECT * FROM seasons ORDER BY season_id DESC;

-- Recent games
SELECT g.game_date, ht.team_abbreviation AS home, at.team_abbreviation AS away,
       g.home_score, g.away_score, g.season_type
FROM games g
JOIN teams ht ON g.home_team_id = ht.team_id
JOIN teams at ON g.away_team_id = at.team_id
ORDER BY g.game_date DESC
LIMIT 10;

-- Player stats from recent game
SELECT p.player_name, pgs.points, pgs.rebounds, pgs.assists, pgs.minutes
FROM player_game_stats pgs
JOIN players p ON pgs.player_id = p.player_id
JOIN games g ON pgs.game_id = g.game_id
ORDER BY g.game_date DESC, pgs.points DESC
LIMIT 20;
```

### Python Script

```bash
python view_database.py
```

## Database Structure

### Core Tables

- `seasons` - NBA season metadata (dates, champions, status)
- `teams` - 30 NBA teams
- `players` - NBA players (active and retired)
- `player_seasons` - Player participation per season

### Game Data

- `games` - Game schedule and results
- `player_game_stats` - Player box scores and advanced stats
- `team_game_stats` - Team statistics per game

### Betting Data

- `team_odds` - Team betting odds (moneyline, spreads, totals)
- `player_prop_odds` - Player prop bets

## Advanced Statistics

The pipeline automatically calculates:

### Player Stats
- True Shooting Percentage (TS%)
- Effective Field Goal Percentage (eFG%)
- Usage Rate (USG%)
- Assist/Rebound/Turnover Percentages
- Points Per Possession (PPP)

### Team Stats
- Offensive/Defensive Rating
- Net Rating
- Pace
- Four Factors Score
- Possession Estimates

## Automated Scheduling

### Daily Updates with Cron

```bash
crontab -e
```

Add this line to run daily at 6 AM:

```cron
0 6 * * * cd /path/to/8-ball && /usr/local/bin/python3 -m src.pipeline.daily_update >> logs/daily_update.log 2>&1
```

## Common Commands

```bash
# Initialize database
python scripts/init_database.py

# Drop all tables (WARNING: deletes all data)
python scripts/init_database.py --drop

# Seed all historical seasons
python scripts/seed_seasons.py

# Backfill last season
python scripts/backfill_season.py --season 2024-25

# Update player seasons
python scripts/migrate_player_seasons.py --update

# Mark retired players
python scripts/update_player_status.py --update

# View database
python view_database.py

# Run daily update
python -m src.pipeline.daily_update
```

## Troubleshooting

### Database Connection Issues

Verify your `.env` configuration:

```bash
python -c "from Gringotts.store.database import DatabaseManager; db = DatabaseManager(); print('Success!' if db.test_connection() else 'Failed')"
```

### Missing Dependencies

```bash
pip install --upgrade -r requirements.txt
```

### Import Errors

Ensure you're in the project root when running scripts:

```bash
cd /path/to/8-ball
python scripts/init_database.py
```

### MySQL Authentication

If you get "Access denied" errors:

```bash
mysql -u root -p

# In MySQL:
ALTER USER 'root'@'localhost' IDENTIFIED WITH mysql_native_password BY 'your_password';
FLUSH PRIVILEGES;
```

## Project Structure

```
8-ball/
├── src/
│   ├── data/
│   │   ├── store/          # Database models and connection
│   │   ├── ingest/         # API data fetching (ESPN, Odds)
│   │   └── analytics/      # Advanced stats calculations
│   ├── pipeline/           # Daily update orchestration
│   └── utils/              # Logger and helpers
├── scripts/
│   ├── init_database.py    # Initialize database
│   ├── seed_seasons.py     # Seed season metadata
│   ├── backfill_season.py  # Historical data ingestion
│   ├── migrate_player_seasons.py  # Create player_seasons records
│   └── update_player_status.py    # Manage player active status
├── config.py               # Configuration constants
├── requirements.txt        # Python dependencies
├── .env                    # Environment variables (create this)
└── README.md               # Full documentation
```

## Next Steps

1. Run daily updates to collect current season data
2. Backfill historical seasons for analysis
3. Set up automated scheduling with cron
4. Build analytics queries and models
5. Integrate with your betting strategy

## Documentation

- `README.md` - Complete project documentation
- `DATABASE_HIERARCHY.md` - Database structure details
- `SEASON_INFO.md` - NBA season information
- `CURRENT_SEASON.md` - Current season context
- `GETTING_STARTED.md` - Detailed setup guide

## Support

For issues or questions, see the full documentation in README.md or check the database hierarchy guide.

