# Advanced NBA Statistics Implementation Summary

## 📋 Overview

This document summarizes the implementation of advanced NBA statistics in the Minerva Sports Analytics system, including database schema updates, automatic calculation triggers, and integration with the Fortuna ecosystem (Holocron, Alexandria, Mercury).

**Date**: 2025-11-14  
**Version**: 2.0  
**Status**: Database schema complete, ready for migration

---

## ✅ What Was Implemented

### 1. Enhanced Database Schema

#### Player Game Stats (`player_game_stats`)
Added **18 advanced metrics** automatically calculated from box score data:

**Shooting Efficiency**:
- `true_shooting_pct` - TS%: Overall shooting efficiency
- `effective_fg_pct` - eFG%: Adjusted for 3-point value
- `field_goal_pct`, `three_point_pct`, `free_throw_pct` - Basic percentages

**Usage and Involvement**:
- `usage_rate` - USG%: Percentage of team plays used
- `assist_percentage` - AST%: Teammate FGs assisted
- `turnover_percentage` - TOV%: Turnovers per 100 plays

**Rebounding**:
- `offensive_rebound_pct` - ORB%: Offensive rebounds grabbed
- `defensive_rebound_pct` - DRB%: Defensive rebounds grabbed
- `total_rebound_pct` - TRB%: Total rebounds grabbed

**Impact Metrics**:
- `offensive_rating` - ORtg: Points per 100 possessions
- `defensive_rating` - DRtg: Points allowed per 100 possessions
- `net_rating` - NetRtg: Point differential per 100 possessions
- `player_efficiency_rating` - PER: Comprehensive efficiency
- `game_score` - GmSc: John Hollinger's single-game metric
- `box_plus_minus` - BPM: Contribution per 100 possessions

#### Team Game Stats (`team_game_stats`)
Added **14 advanced metrics** including Dean Oliver's Four Factors:

**Pace and Efficiency**:
- `pace` - Possessions per 48 minutes
- `possessions` - Estimated game possessions
- `offensive_rating`, `defensive_rating`, `net_rating`

**Four Factors** (in order of importance):
1. **Shooting** - `effective_fg_pct`, `true_shooting_pct`
2. **Turnovers** - `turnover_rate`
3. **Rebounding** - `offensive_rebound_pct`, `defensive_rebound_pct`, `total_rebound_pct`
4. **Free Throws** - `free_throw_rate`

**Additional Metrics**:
- `assist_to_turnover_ratio` - Ball movement vs. security
- `steal_percentage` - Steals per 100 opponent possessions
- `block_percentage` - Blocks per 100 opponent FGA

### 2. Database Triggers for Automatic Calculation

Created PostgreSQL functions and triggers that automatically calculate advanced stats when raw box score data is inserted:

**`calculate_player_advanced_stats()`**:
- Calculates TS%, eFG%, FG%, 3P%, FT%
- Calculates Game Score (weighted box score formula)
- Calculates Net Rating (if ORtg/DRtg provided)
- Triggers on INSERT/UPDATE to `player_game_stats`

**`calculate_team_advanced_stats()`**:
- Calculates shooting percentages (TS%, eFG%)
- Calculates Free Throw Rate
- Calculates Assist/Turnover Ratio
- Calculates Offensive Rating (if possessions provided)
- Calculates Net Rating
- Triggers on INSERT/UPDATE to `team_game_stats`

### 3. Comprehensive Documentation

Created three detailed documentation files:

#### `ADVANCED_STATS_FORMULAS.md` (794 lines)
- **26 advanced statistics** with formulas and descriptions
- Implementation notes (database vs. application layer)
- Calculation pipeline diagram
- References to Basketball Reference, NBA.com, Dean Oliver
- Integration with Fortuna ML models

#### `SYSTEM_INTEGRATION.md` (432 lines)
- Architecture diagram showing Minerva ↔ Holocron ↔ Alexandria
- Integration points and data flows
- Real-time Redis stream specifications
- ML features endpoint design
- Bet opportunity enrichment examples
- Live betting adjustment algorithms

#### `ADVANCED_STATS_IMPLEMENTATION_SUMMARY.md` (this file)
- Implementation summary
- Migration instructions
- Next steps and roadmap

### 4. Updated Database Migrations

Created **12 new migration files** (010-021):

1. `010_drop_all_tables.sql` - Clean slate for schema redesign
2. `011_create_seasons_v2.sql` - Improved seasons table
3. `012_create_teams_v2.sql` - Improved teams table
4. `013_create_players_v2.sql` - Improved players table with full-text search
5. `014_create_player_team_history.sql` - **NEW**: Temporal tracking of player-team relationships
6. `015_create_games_v2.sql` - Improved games table with partitioning support
7. `016_create_player_game_stats_v2.sql` - **18 new advanced stat columns**
8. `017_create_team_game_stats_v2.sql` - **14 new advanced stat columns**
9. `018_create_odds_mappings_v2.sql` - Improved Alexandria integration
10. `019_create_backfill_jobs_v2.sql` - Improved backfill tracking
11. `020_create_triggers.sql` - **Automatic advanced stats calculation**
12. `021_create_materialized_views.sql` - Performance optimization

---

## 🔧 Technical Details

### Database Trigger Flow

```
1. ESPN API → Raw Box Score Data
   ↓
2. Go Application → Parse and store in player_game_stats
   ↓
3. PostgreSQL Trigger → calculate_player_advanced_stats()
   - Automatically calculates TS%, eFG%, Game Score
   - No application code needed
   ↓
4. Application Layer → Calculate context-dependent metrics
   - Usage Rate (requires team totals)
   - Rebound Percentages (requires opponent data)
   - PER (requires league averages)
   ↓
5. Store calculated metrics back to database
   ↓
6. Materialized Views → Pre-aggregate season averages
```

### Metrics Calculated by Database vs. Application

**Database Triggers** (automatic, no app code):
- ✅ FG%, 3P%, FT%, TS%, eFG%
- ✅ Game Score
- ✅ Net Rating (if ORtg/DRtg provided)
- ✅ Free Throw Rate
- ✅ Assist/Turnover Ratio

**Application Layer** (requires context):
- ⚙️ Usage Rate (needs team totals)
- ⚙️ Assist/Turnover/Rebound Percentages (needs opponent data)
- ⚙️ Offensive/Defensive Rating (needs possessions estimation)
- ⚙️ PER (needs league averages and pace)
- ⚙️ BPM (needs regression model)

### Integration with Fortuna Systems

#### Holocron (Bet History)
- `odds_mappings` table links Minerva players/games to Alexandria events
- Advanced stats enrich bet opportunities with context
- Historical performance tracking for bet analysis
- ML model features for prediction

#### Alexandria (Odds Data)
- Player props linked to Minerva player stats
- Game totals informed by team pace and efficiency
- Live betting adjustments based on in-game advanced stats

#### Mercury (Odds Polling)
- No direct integration (Mercury → Alexandria → Holocron → Minerva)
- Minerva provides sports context for odds analysis

---

## 📊 Example Queries

### Get Player's Advanced Stats for Season
```sql
SELECT 
  p.full_name,
  COUNT(DISTINCT pgs.game_id) as games_played,
  ROUND(AVG(pgs.points), 1) as ppg,
  ROUND(AVG(pgs.rebounds), 1) as rpg,
  ROUND(AVG(pgs.assists), 1) as apg,
  ROUND(AVG(pgs.true_shooting_pct), 3) as ts_pct,
  ROUND(AVG(pgs.effective_fg_pct), 3) as efg_pct,
  ROUND(AVG(pgs.usage_rate), 3) as usg_rate,
  ROUND(AVG(pgs.game_score), 1) as avg_game_score,
  ROUND(AVG(pgs.player_efficiency_rating), 1) as per
FROM players p
JOIN player_game_stats pgs ON p.player_id = pgs.player_id
JOIN games g ON pgs.game_id = g.game_id
WHERE p.full_name = 'LeBron James'
  AND g.season_id = '2024-25'
GROUP BY p.player_id, p.full_name;
```

### Get Team's Four Factors
```sql
SELECT 
  t.team_name,
  ROUND(AVG(tgs.effective_fg_pct), 3) as efg_pct,
  ROUND(AVG(tgs.turnover_rate), 3) as tov_rate,
  ROUND(AVG(tgs.offensive_rebound_pct), 3) as orb_pct,
  ROUND(AVG(tgs.free_throw_rate), 3) as ftr,
  ROUND(AVG(tgs.pace), 1) as pace,
  ROUND(AVG(tgs.offensive_rating), 1) as ortg,
  ROUND(AVG(tgs.defensive_rating), 1) as drtg,
  ROUND(AVG(tgs.net_rating), 1) as net_rtg
FROM teams t
JOIN team_game_stats tgs ON t.team_id = tgs.team_id
JOIN games g ON tgs.game_id = g.game_id
WHERE t.team_name = 'Los Angeles Lakers'
  AND g.season_id = '2024-25'
GROUP BY t.team_id, t.team_name;
```

### Find High-Efficiency Games (TS% > 70%)
```sql
SELECT 
  p.full_name,
  g.game_date,
  t_home.abbreviation || ' @ ' || t_away.abbreviation as matchup,
  pgs.points,
  pgs.field_goals_made || '-' || pgs.field_goals_attempted as fg,
  pgs.three_pointers_made || '-' || pgs.three_pointers_attempted as threes,
  pgs.free_throws_made || '-' || pgs.free_throws_attempted as ft,
  ROUND(pgs.true_shooting_pct, 3) as ts_pct,
  ROUND(pgs.game_score, 1) as game_score
FROM player_game_stats pgs
JOIN players p ON pgs.player_id = p.player_id
JOIN games g ON pgs.game_id = g.game_id
JOIN teams t_home ON g.home_team_id = t_home.team_id
JOIN teams t_away ON g.away_team_id = t_away.team_id
WHERE pgs.true_shooting_pct > 0.700
  AND pgs.points >= 20
  AND g.season_id = '2024-25'
ORDER BY pgs.true_shooting_pct DESC
LIMIT 20;
```

---

## 🚀 Next Steps

### Immediate (Before Applying Migrations)
1. **Backup Current Database**
   ```bash
   docker exec fortuna-atlas pg_dump -U fortuna atlas > atlas_backup_$(date +%Y%m%d).sql
   ```

2. **Review Migration Files**
   - Verify all 12 migration files (010-021)
   - Ensure no syntax errors
   - Test on development database first

3. **Apply Migrations**
   ```bash
   # Option 1: Restart Minerva (migrations auto-apply)
   docker-compose restart minerva
   
   # Option 2: Manual application
   docker exec -it fortuna-atlas psql -U fortuna -d atlas -f /path/to/migrations/010_drop_all_tables.sql
   # ... repeat for 011-021
   ```

### Short-Term (Phase 2 - Current Sprint)
1. **Implement Application-Layer Calculations**
   - Create `internal/analytics/calculator.go`
   - Implement Usage Rate calculation
   - Implement Rebound Percentage calculations
   - Implement PER calculation
   - Update ingestion pipeline to call calculator

2. **Update Repository Layer**
   - Add methods to fetch team totals for context
   - Add methods to fetch opponent stats
   - Add methods to fetch league averages

3. **Test Advanced Stats**
   - Backfill historical games
   - Verify trigger calculations
   - Verify application-layer calculations
   - Compare against Basketball Reference for accuracy

### Medium-Term (Phase 3-4)
1. **ML Features Endpoint**
   - Implement `GET /api/v1/ml/features`
   - Expose structured data for model training
   - Add filtering by date range, player, team

2. **Odds Mapping Automation**
   - Implement fuzzy matching for player names
   - Auto-link Alexandria events to Minerva games
   - Confidence scoring for mappings

3. **Bet Context Enrichment**
   - Integrate with Holocron API
   - Enrich opportunities with advanced stats
   - Display in web UI

4. **Frontend Integration**
   - Player stats dashboard with advanced metrics
   - Team stats dashboard with Four Factors
   - Game analysis page with advanced breakdowns

### Long-Term (Phase 5-6)
1. **Predictive Models**
   - Player prop prediction models
   - Game total prediction models
   - Live betting adjustment models

2. **Real-Time Advanced Stats**
   - Calculate advanced stats during live games
   - Stream updates via Redis
   - Display in live game UI

3. **Multi-Sport Expansion**
   - NFL: Passing efficiency, DVOA, EPA
   - MLB: wOBA, FIP, WAR
   - NHL: Corsi, Fenwick, xG

---

## 📚 References

### Documentation Files
- `ADVANCED_STATS_FORMULAS.md` - Detailed formulas and calculations
- `SYSTEM_INTEGRATION.md` - Integration with Fortuna systems
- `ATLAS_DATABASE_DESIGN.md` - Complete database schema
- `MINERVA_IMPLEMENTATION_PLAN.md` - Full implementation roadmap

### External Resources
- [Basketball Reference - Glossary](https://www.basketball-reference.com/about/glossary.html)
- [NBA.com Stats - Advanced Stats](https://www.nba.com/stats/help/glossary)
- [Dean Oliver - Basketball on Paper](https://www.amazon.com/Basketball-Paper-Rules-Performance-Analysis/dp/1574886886)
- [John Hollinger - ESPN Basketball Analytics](https://www.espn.com/nba/hollinger/statistics)

---

## ✅ Checklist

### Database Schema
- [x] Research advanced NBA statistics
- [x] Design player_game_stats schema with 18 advanced metrics
- [x] Design team_game_stats schema with 14 advanced metrics
- [x] Create player_team_history table for temporal tracking
- [x] Design database triggers for automatic calculation
- [x] Create migration files (010-021)
- [ ] Apply migrations to development database
- [ ] Apply migrations to production database

### Documentation
- [x] Document all 26 advanced statistics with formulas
- [x] Document system integration (Holocron, Alexandria, Mercury)
- [x] Document calculation pipeline
- [x] Create implementation summary (this file)

### Application Layer
- [ ] Implement analytics calculator for context-dependent metrics
- [ ] Update ingestion pipeline to call calculator
- [ ] Add repository methods for team/opponent context
- [ ] Test calculations against Basketball Reference

### Integration
- [ ] Implement odds_mappings automation
- [ ] Integrate with Holocron for bet enrichment
- [ ] Implement ML features endpoint
- [ ] Update web UI to display advanced stats

### Testing
- [ ] Unit tests for trigger functions
- [ ] Integration tests for ingestion pipeline
- [ ] Accuracy tests vs. Basketball Reference
- [ ] Performance tests for large datasets

---

**Status**: ✅ Database schema complete, ready for migration  
**Next Action**: Apply migrations 010-021 to development database  
**Estimated Time to Production**: 2-3 weeks (including testing and validation)



