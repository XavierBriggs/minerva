# Minerva System Integration Guide

This document describes how Minerva integrates with other Fortuna systems (Holocron, Alexandria, Mercury) and how advanced NBA statistics flow through the architecture.

## 🏗️ System Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                         FORTUNA ECOSYSTEM                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────────┐      ┌──────────────┐      ┌──────────────┐     │
│  │   MERCURY    │      │   MINERVA    │      │  ALEXANDRIA  │     │
│  │              │      │              │      │              │     │
│  │ Odds Polling │◄────►│ Sports Data  │◄────►│  Odds Data   │     │
│  │ (TheOddsAPI) │      │ (ESPN/Google)│      │  (Raw Odds)  │     │
│  └──────┬───────┘      └──────┬───────┘      └──────┬───────┘     │
│         │                     │                     │              │
│         │                     │                     │              │
│         ▼                     ▼                     ▼              │
│  ┌──────────────────────────────────────────────────────────┐     │
│  │                    REDIS STREAMS                          │     │
│  │  • odds.raw.basketball_nba                               │     │
│  │  • games.live.basketball_nba                             │     │
│  │  • games.stats.basketball_nba                            │     │
│  │  • opportunities.detected                                │     │
│  └──────────────────────────────────────────────────────────┘     │
│         │                     │                     │              │
│         ▼                     ▼                     ▼              │
│  ┌──────────────┐      ┌──────────────┐      ┌──────────────┐     │
│  │  NORMALIZER  │      │   HOLOCRON   │      │ WS-BROADCAST │     │
│  │              │      │              │      │              │     │
│  │ Edge Calc    │─────►│ Opportunities│─────►│  Web Client  │     │
│  │ Fair Prices  │      │ Bet History  │      │  (Next.js)   │     │
│  └──────────────┘      └──────────────┘      └──────────────┘     │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

## 🔗 Integration Points

### 1. Minerva ↔ Alexandria (Odds Mapping)

**Purpose**: Link Minerva's sports data (games, teams, players) to Alexandria's odds data (events, outcomes).

**Mechanism**: `odds_mappings` table in Atlas database

```sql
-- Example: Map Minerva game to Alexandria event
INSERT INTO odds_mappings (
  sport,
  minerva_game_id,
  alexandria_event_id,
  alexandria_participant_name,
  mapping_type,
  confidence,
  match_method
) VALUES (
  'basketball_nba',
  12345,
  'abc123xyz',
  'Los Angeles Lakers vs Boston Celtics',
  'game',
  1.00,
  'exact'
);
```

**Use Cases**:
- **Game Mapping**: Link Minerva game to Alexandria event for bet context
- **Team Mapping**: Map team names across systems (e.g., "LAL" → "Los Angeles Lakers")
- **Player Mapping**: Link player props to Minerva player stats

**Advanced Stats Integration**:
- When a bet opportunity is detected on a player prop (e.g., "LeBron James Over 24.5 Points")
- Minerva can provide historical context:
  - Player's season average (PPG)
  - Recent form (last 5 games)
  - Advanced metrics (TS%, Usage Rate, ORtg)
  - Matchup history vs. opponent
  - Team pace and offensive rating

### 2. Minerva → Redis Streams (Real-Time Updates)

**Purpose**: Broadcast live game updates and stats to other services and web clients.

**Streams Published**:

#### `games.live.basketball_nba`
```json
{
  "game_id": 12345,
  "home_team": "Lakers",
  "away_team": "Celtics",
  "home_score": 98,
  "away_score": 95,
  "period": 4,
  "clock": "2:34",
  "status": "in_progress"
}
```

#### `games.stats.basketball_nba`
```json
{
  "game_id": 12345,
  "player_id": 2544,
  "player_name": "LeBron James",
  "points": 28,
  "rebounds": 8,
  "assists": 10,
  "true_shooting_pct": 0.612,
  "game_score": 32.4,
  "usage_rate": 0.285
}
```

**Consumers**:
- **WS-Broadcaster**: Pushes updates to web clients
- **Edge Detector**: Adjusts live betting models based on in-game performance
- **Alert Service**: Triggers notifications for significant stat milestones

### 3. Minerva → Holocron (Bet Context & Analysis)

**Purpose**: Provide sports context for bet opportunities and historical bet analysis.

**Integration Flow**:

```
1. Alexandria: Bet opportunity detected (e.g., "LeBron Over 24.5 Pts @ +110")
   ↓
2. Holocron: Store opportunity in `opportunities` table
   ↓
3. Minerva: Query player stats via odds_mappings
   ↓
4. Enrich Opportunity:
   - Season Average: 26.3 PPG
   - Last 5 Games: 28.2 PPG
   - vs. Celtics (Season): 29.5 PPG
   - True Shooting %: 61.2% (Elite)
   - Usage Rate: 28.5% (High)
   - Team Pace: 102.3 (Fast)
   ↓
5. Display in Web UI with full context
```

**Bet Performance Analysis**:
- After game completion, Minerva provides actual stats
- Compare bet outcome to player's advanced metrics
- Track which stat profiles correlate with winning bets
- Feed data into ML models for future predictions

### 4. Minerva API → ML Features Endpoint

**Purpose**: Expose structured data for machine learning model training.

**Endpoint**: `GET /api/v1/ml/features`

**Response Structure**:
```json
{
  "game_features": {
    "game_id": 12345,
    "home_team_id": 1,
    "away_team_id": 2,
    "home_pace": 102.3,
    "away_pace": 98.7,
    "home_offensive_rating": 118.5,
    "away_defensive_rating": 112.3,
    "home_net_rating": 6.2,
    "away_net_rating": -1.8,
    "home_four_factors": {
      "efg_pct": 0.545,
      "tov_rate": 0.118,
      "orb_pct": 0.285,
      "ftr": 0.245
    }
  },
  "player_features": [
    {
      "player_id": 2544,
      "player_name": "LeBron James",
      "season_stats": {
        "ppg": 26.3,
        "rpg": 7.8,
        "apg": 9.2,
        "ts_pct": 0.612,
        "usage_rate": 0.285,
        "per": 24.8
      },
      "last_5_games": {
        "ppg": 28.2,
        "ts_pct": 0.625,
        "game_score_avg": 28.5
      },
      "vs_opponent": {
        "games": 3,
        "ppg": 29.5,
        "ts_pct": 0.638
      }
    }
  ]
}
```

**ML Model Use Cases**:
1. **Game Outcome Prediction**
   - Features: Team pace, ORtg, DRtg, Four Factors, recent form
   - Target: Win probability, point spread

2. **Player Prop Prediction**
   - Features: Player season stats, recent form, matchup history, team pace
   - Target: Over/under probability for points, rebounds, assists

3. **Live Betting Models**
   - Features: Real-time advanced stats, game flow, momentum indicators
   - Target: Adjusted win probability, live line value

4. **Closing Line Value (CLV) Models**
   - Features: Historical advanced stats, line movement, sharp book action
   - Target: Predicted closing line, CLV estimation

## 📊 Advanced Stats Integration Examples

### Example 1: Player Prop Bet Analysis

**Scenario**: Bet opportunity detected for "LeBron James Over 24.5 Points"

**Minerva Enrichment**:
```sql
-- Get player's advanced stats context
SELECT 
  p.full_name,
  -- Season averages
  AVG(pgs.points) as season_ppg,
  AVG(pgs.true_shooting_pct) as season_ts_pct,
  AVG(pgs.usage_rate) as season_usg_rate,
  AVG(pgs.game_score) as season_game_score,
  -- Recent form (last 5 games)
  AVG(pgs.points) FILTER (WHERE g.game_date >= NOW() - INTERVAL '10 days') as recent_ppg,
  -- vs. Opponent
  AVG(pgs.points) FILTER (WHERE g.away_team_id = 2 OR g.home_team_id = 2) as vs_opponent_ppg
FROM players p
JOIN player_game_stats pgs ON p.player_id = pgs.player_id
JOIN games g ON pgs.game_id = g.game_id
WHERE p.player_id = 2544
  AND g.season_id = '2024-25'
GROUP BY p.player_id, p.full_name;
```

**Decision Support**:
- ✅ **Take Bet** if: Recent PPG > 28, TS% > 60%, vs. Opponent PPG > 27
- ⚠️ **Caution** if: Usage Rate declining, Team pace slowing
- ❌ **Pass** if: Recent PPG < 22, TS% < 50%, High turnover rate

### Example 2: Game Total (Over/Under) Analysis

**Scenario**: Bet opportunity for "Lakers vs Celtics Over 223.5 Total Points"

**Minerva Enrichment**:
```sql
-- Get team pace and efficiency metrics
SELECT 
  t.team_name,
  AVG(tgs.pace) as avg_pace,
  AVG(tgs.offensive_rating) as avg_ortg,
  AVG(tgs.defensive_rating) as avg_drtg,
  AVG(tgs.points) as avg_points_scored,
  AVG(tgs.possessions) as avg_possessions
FROM teams t
JOIN team_game_stats tgs ON t.team_id = tgs.team_id
JOIN games g ON tgs.game_id = g.game_id
WHERE t.team_id IN (1, 2)  -- Lakers, Celtics
  AND g.season_id = '2024-25'
GROUP BY t.team_id, t.team_name;
```

**Calculation**:
```
Estimated Total = (Lakers Pace + Celtics Pace) / 2 * 
                  ((Lakers ORtg + Celtics ORtg) / 200)

Example:
  Lakers Pace: 102.3, ORtg: 118.5
  Celtics Pace: 98.7, ORtg: 115.2
  
  Avg Pace: (102.3 + 98.7) / 2 = 100.5
  Avg ORtg: (118.5 + 115.2) / 2 = 116.85
  
  Estimated Total: 100.5 * (116.85 / 100) = 117.4 points per team
  Game Total: 117.4 * 2 = 234.8 points
  
  Line: 223.5 → OVER looks good (11.3 point edge)
```

### Example 3: Live Betting Adjustment

**Scenario**: Lakers trailing by 8 at halftime, live line adjusted to Lakers +5.5

**Minerva Real-Time Analysis**:
```sql
-- Get halftime stats and compare to season averages
SELECT 
  -- Current game stats (1st half)
  SUM(pgs.points) as h1_points,
  AVG(pgs.true_shooting_pct) as h1_ts_pct,
  SUM(pgs.turnovers) as h1_turnovers,
  -- Season 2nd half performance
  AVG(pgs.points) FILTER (WHERE g.period >= 3) as season_h2_ppg,
  AVG(tgs.net_rating) FILTER (WHERE g.period >= 3) as season_h2_net_rating
FROM player_game_stats pgs
JOIN games g ON pgs.game_id = g.game_id
JOIN team_game_stats tgs ON g.game_id = tgs.game_id AND pgs.team_id = tgs.team_id
WHERE pgs.team_id = 1  -- Lakers
  AND g.season_id = '2024-25'
GROUP BY pgs.team_id;
```

**Decision**:
- If Lakers' 2nd half Net Rating > +8: **Take Lakers +5.5**
- If current TS% < season average: **Pass** (shooting cold)
- If high turnovers (>10): **Pass** (sloppy play)

## 🔄 Data Flow for Bet Opportunity

```
1. Mercury polls TheOddsAPI
   ↓
2. New odds detected: "LeBron Over 24.5 Pts @ +110"
   ↓
3. Alexandria stores raw odds in `odds_raw` table
   ↓
4. Normalizer calculates fair price and edge
   ↓
5. Edge Detector identifies +2.5% edge
   ↓
6. Holocron stores opportunity in `opportunities` table
   ↓
7. API Gateway queries Minerva via `odds_mappings`:
   - Get LeBron's player_id from Alexandria participant name
   - Fetch season stats, recent form, matchup history
   - Calculate advanced metrics (TS%, Usage%, PER)
   ↓
8. WS-Broadcaster pushes enriched opportunity to web client:
   {
     "opportunity_id": 12345,
     "type": "edge",
     "edge_pct": 2.5,
     "market": "player_points",
     "player": "LeBron James",
     "line": "Over 24.5",
     "price": +110,
     "context": {
       "season_ppg": 26.3,
       "last_5_ppg": 28.2,
       "vs_opponent_ppg": 29.5,
       "true_shooting_pct": 0.612,
       "usage_rate": 0.285,
       "recommendation": "STRONG BET"
     }
   }
   ↓
9. User takes bet in web UI
   ↓
10. Holocron stores bet in `bets` table with Minerva context
   ↓
11. After game, Minerva provides actual stats
   ↓
12. Holocron calculates bet performance and CLV
   ↓
13. ML model trained on historical bet + stats data
```

## 🎯 Key Integration Benefits

### 1. **Contextual Bet Opportunities**
- Every bet opportunity enriched with relevant stats
- Historical performance data at your fingertips
- Advanced metrics provide edge beyond basic stats

### 2. **Bet Performance Tracking**
- Link every bet to actual game outcomes
- Analyze which player profiles yield winning bets
- Track ROI by stat category (high TS%, high usage, etc.)

### 3. **ML Model Training**
- Structured feature sets for model training
- Historical data for backtesting strategies
- Real-time features for live betting models

### 4. **Real-Time Adjustments**
- Live game stats update betting models
- Momentum indicators from advanced metrics
- In-game prop adjustments based on performance

## 🚀 Future Enhancements

### Phase 1 (Current)
- ✅ Database schema with advanced stats
- ✅ Triggers for automatic calculation
- ✅ `odds_mappings` table for Alexandria integration
- ⏳ Application layer for context-dependent metrics

### Phase 2
- ⏳ Automatic odds mapping (fuzzy matching)
- ⏳ ML features endpoint implementation
- ⏳ Bet context enrichment in Holocron
- ⏳ Real-time stat streaming to WS-Broadcaster

### Phase 3
- ⏳ Predictive models for player props
- ⏳ Game total prediction models
- ⏳ Live betting adjustment algorithms
- ⏳ CLV estimation models

### Phase 4
- ⏳ Multi-sport expansion (NFL, MLB, NHL)
- ⏳ Custom advanced metrics (proprietary formulas)
- ⏳ Automated bet recommendations
- ⏳ Portfolio optimization (Kelly Criterion)

---

**Last Updated**: 2025-11-14  
**Version**: 1.0  
**Author**: Minerva Sports Analytics System


