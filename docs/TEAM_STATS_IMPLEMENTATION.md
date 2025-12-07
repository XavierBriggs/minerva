# Team Stats Ingestion Implementation

## Summary
Added complete team statistics ingestion for both live games and backfill operations.

## Changes Made

### 1. Parser (`minerva-go/internal/ingest/espn/parser.go`)
- ✅ Added `ParsedTeamStats` struct to hold team stats with metadata
- ✅ Added `ParseTeamStats()` function to extract team-level statistics from ESPN game summary
- ✅ Added `parseTeamStatsDetailed()` to parse individual team stat lines
- ✅ Added `calculateTeamTotalsFromPlayers()` as fallback to sum player stats if team totals not available
- ✅ Added `parseStatValue()` helper to extract numeric values from stat strings

**Stats Parsed:**
- Points (PTS)
- Field Goals (FG: made-attempted)
- 3-Pointers (3PT: made-attempted)
- Free Throws (FT: made-attempted)
- Rebounds (REB, OREB, DREB)
- Assists (AST)
- Steals (STL)
- Blocks (BLK)
- Turnovers (TO)
- Personal Fouls (PF)

### 2. Ingester (`minerva-go/internal/ingest/espn/ingester.go`)
- ✅ Modified `ingestStatsFromSummary()` to call team stats ingestion
- ✅ Added `ingestTeamStatsFromSummary()` function to:
  - Parse team stats from ESPN summary
  - Lookup team IDs
  - Determine home/away status
  - Upsert team stats to database

**Integration Points:**
- Called automatically during backfill operations
- Called automatically during live game ingestion
- Non-blocking - failures logged but don't stop player stats ingestion

### 3. Repository (`minerva-go/internal/store/repository/stats.go`)
- ✅ Added `UpsertTeamStats()` method
- ✅ Uses `ON CONFLICT (game_id, team_id)` for idempotent upserts
- ✅ Returns `stat_id` (matches Atlas schema)
- ✅ Updates `updated_at` timestamp on conflicts

### 4. Frontend Endpoints
- ✅ Added `GET /api/v1/minerva/teams` - Get all teams
- ✅ Added `GET /api/v1/minerva/teams/{id}` - Get specific team
- ✅ Added player search functionality
- ✅ Enhanced Stats & History tab with team/player browsing

## Data Flow

```
ESPN API (Game Summary)
    ↓
ParseTeamStats() - Extract team totals from boxscore
    ↓
ingestTeamStatsFromSummary() - Resolve team IDs, set home/away
    ↓
UpsertTeamStats() - Insert/Update in database
    ↓
team_game_stats table
```

## Schema Compliance

✅ **Matches Atlas Schema:**
- Primary key: `stat_id` (BIGSERIAL)
- Unique constraint: `(game_id, team_id)`
- Foreign keys: `game_id → games`, `team_id → teams`
- `is_home` BOOLEAN to distinguish home/away
- All stat columns match schema definition

## Advanced Stats

The database triggers automatically calculate advanced metrics:
- True Shooting % (TS%)
- Effective FG % (eFG%)
- Turnover Rate
- Offensive/Defensive Rebound %
- Free Throw Rate
- Assist/Turnover Ratio
- Steal/Block Percentages
- Pace, Possessions, Offensive/Defensive/Net Rating

## Testing Checklist

### Backend
- [ ] Rebuild Minerva service
- [ ] Run backfill for a recent date
- [ ] Verify team stats appear in `team_game_stats` table
- [ ] Check logs for team stats ingestion messages
- [ ] Verify both home and away teams have stats

### Database
```sql
-- Check team stats were ingested
SELECT COUNT(*) FROM team_game_stats;

-- View sample team stats
SELECT 
  tgs.stat_id,
  g.game_date,
  t.abbreviation as team,
  tgs.is_home,
  tgs.points,
  tgs.field_goals_made || '-' || tgs.field_goals_attempted as fg,
  tgs.three_pointers_made || '-' || tgs.three_pointers_attempted as three_pt,
  tgs.rebounds,
  tgs.assists
FROM team_game_stats tgs
JOIN games g ON tgs.game_id = g.game_id
JOIN teams t ON tgs.team_id = t.team_id
ORDER BY g.game_date DESC
LIMIT 10;

-- Verify advanced stats are calculated
SELECT 
  t.abbreviation,
  tgs.effective_fg_pct,
  tgs.true_shooting_pct,
  tgs.offensive_rating,
  tgs.defensive_rating,
  tgs.net_rating
FROM team_game_stats tgs
JOIN teams t ON tgs.team_id = t.team_id
WHERE tgs.effective_fg_pct IS NOT NULL
LIMIT 5;
```

### Frontend
- [ ] Teams endpoint returns all teams
- [ ] Player search shows dropdown results
- [ ] Browse Teams button works
- [ ] Team detail pages load
- [ ] Box scores display correctly

## Future Enhancements

1. **API Endpoints for Team Stats**
   - `GET /api/v1/minerva/teams/{id}/stats?season={season}` - Season averages
   - `GET /api/v1/minerva/games/{id}/team-stats` - Team stats for specific game

2. **UI Components**
   - Team stats comparison view
   - Team performance trends
   - Four Factors visualization

3. **Analytics**
   - Team efficiency rankings
   - Head-to-head comparisons
   - Playoff performance metrics

## Notes

- Team stats ingestion is **non-blocking** - if it fails, player stats still succeed
- ESPN sometimes provides explicit team totals, sometimes we calculate from player stats
- Advanced metrics are calculated automatically by database triggers (see `020_create_triggers.sql`)
- The `is_home` field is determined by comparing `team_id` with `game.home_team_id`








