#  NBA Season Information

## 2025-2026 Season

### Key Dates

- **Preseason**: October 4-18, 2025
- **Regular Season Start**: October 21, 2025
- **All-Star Weekend**: February 13-16, 2026 (Indianapolis)
- **Regular Season End**: April 12, 2026
- **Play-In Tournament**: April 14-17, 2026
- **Playoffs Start**: April 18, 2026
- **NBA Finals**: June 2026

### Season Configuration

The pipeline automatically detects the season based on game dates:

| Game Date | Season ID | Logic |
|-----------|-----------|-------|
| Oct 21, 2025 | `2025-26` | Month ≥ October → start year |
| Jan 15, 2026 | `2025-26` | Month < October → year - 1 |
| Apr 12, 2026 | `2025-26` | Season ends in April |
| Oct 20, 2026 | `2026-27` | Next season starts |

### Data Collection Schedule

For optimal data collection during the 2025-26 season:

#### Regular Season (Oct 2025 - Apr 2026)

**Daily Updates** - Run after all games complete:
```bash
# Cron: 2 AM EST (most games finish by 1 AM)
0 2 * * * python -m src.pipeline.daily_update
```

**Pre-Game Odds** - Fetch before games start:
```bash
# Cron: 10 AM EST (before most game times)
0 10 * * * python -m src.pipeline.daily_update --upcoming-odds
```

**Live Odds Updates** (optional) - During game days:
```bash
# Cron: Every 2 hours on game days
0 */2 * * * python -m src.pipeline.daily_update --upcoming-odds
```

#### Playoffs (Apr-Jun 2026)

- Games may start/end later
- Adjust cron to 3-4 AM EST
- More frequent odds updates recommended

#### Off-Season (Jun-Sep 2026)

- Minimal game activity
- Can pause daily updates
- Resume in October for preseason

### Historical Data

To load complete 2025-26 season data:

```bash
# Load regular season (example: Nov-Dec 2025)
python -m src.pipeline.daily_update --backfill \
    --start-date 2025-11-01 \
    --end-date 2025-12-31

# Load from season start
python -m src.pipeline.daily_update --backfill \
    --start-date 2025-10-21 \
    --end-date $(date -v-1d +%Y-%m-%d)
```

### Team Changes

No franchise relocations or name changes for 2025-26 season.

All 30 teams remain:
- **Eastern Conference**: 15 teams
- **Western Conference**: 15 teams

### Database Considerations

#### Season-Specific Queries

```sql
-- Get all 2025-26 regular season games
SELECT * FROM games 
WHERE season_id = '2025-26' 
AND season_type = 'Regular Season';

-- Player stats for current season
SELECT p.player_name, AVG(pgs.points) as avg_points
FROM player_game_stats pgs
JOIN players p ON pgs.player_id = p.player_id
JOIN games g ON pgs.game_id = g.game_id
WHERE g.season_id = '2025-26'
GROUP BY p.player_id;
```

#### Storage Estimates

For full 2025-26 season:
- **Games**: ~1,230 regular season + ~90 playoff = 1,320 records
- **Player Stats**: ~30,000 records (avg 23 per game)
- **Team Stats**: ~2,640 records (2 per game)
- **Odds**: ~20,000 records (multiple bookmakers)
- **Player Props**: ~100,000+ records (multiple markets)

**Total Database Size**: ~50-100 MB per season

### API Usage Planning

#### The Odds API Free Tier
- 500 requests/month
- Each game odds fetch = 1 request
- Plan accordingly:
  - ~15 games/day × 1 request = 15 requests/day
  - 450 requests for daily updates
  - 50 requests for upcoming odds
  - Use sparingly during playoffs

#### ESPN API
- No official rate limit
- Be respectful: 1-2 requests per second max
- No authentication required

### Troubleshooting Season Detection

If season IDs appear incorrect:

```python
# Test season detection
from datetime import datetime
from Gringotts.ingest.espn_stats import ESPNStatsIngestion

test_dates = [
    datetime(2025, 10, 21),  # Season opener
    datetime(2026, 1, 15),   # Mid-season
    datetime(2026, 4, 12),   # Season closer
]

for date in test_dates:
    season = ESPNStatsIngestion._get_season_id(date)
    print(f"{date.date()} → {season}")

# Expected output:
# 2025-10-21 → 2025-26
# 2026-01-15 → 2025-26
# 2026-04-12 → 2025-26
```

### Advanced Analytics Notes

For 2025-26 season analysis:

1. **League averages may change** - Monitor pace, scoring
2. **Rule changes** - Check for new NBA rules affecting stats
3. **Load management** - More prevalent, affects player data
4. **Betting markets** - New prop types may be available

### Recommended Practices

**Start collecting from opening day** (Oct 21, 2025)
**Run daily updates consistently** for complete dataset
**Backup database weekly** during season
**Monitor API usage** to avoid rate limits
**Validate data quality** spot-check games weekly

**Don't skip days** - historical odds may not be available
**Don't over-query** - respect API rate limits
**Don't assume player teams** - trades happen mid-season

---

**Ready for 2025-26!** 

The pipeline is fully configured and tested for the upcoming season. Just run the initialization script and start collecting data from opening night!

