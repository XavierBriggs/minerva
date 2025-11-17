# NBA Advanced Statistics Formulas

This document details all advanced NBA statistics calculated and stored in the Atlas database. These metrics are automatically calculated via database triggers when raw box score data is inserted.

## 📊 Player Advanced Statistics

### Shooting Efficiency Metrics

#### 1. **Field Goal Percentage (FG%)**
- **Formula**: `FG% = FGM / FGA`
- **Description**: Basic shooting percentage
- **Range**: 0.000 to 1.000 (0% to 100%)
- **Good Value**: >0.500 (50%)

#### 2. **Three-Point Percentage (3P%)**
- **Formula**: `3P% = 3PM / 3PA`
- **Description**: Three-point shooting percentage
- **Range**: 0.000 to 1.000
- **Good Value**: >0.360 (36%)

#### 3. **Free Throw Percentage (FT%)**
- **Formula**: `FT% = FTM / FTA`
- **Description**: Free throw shooting percentage
- **Range**: 0.000 to 1.000
- **Good Value**: >0.800 (80%)

#### 4. **True Shooting Percentage (TS%)**
- **Formula**: `TS% = PTS / (2 * (FGA + 0.44 * FTA))`
- **Description**: Measures shooting efficiency accounting for 2PT, 3PT, and FT
- **Why 0.44?**: Estimates possessions ending in free throws (accounts for and-ones, technicals)
- **Range**: 0.000 to 1.000+
- **League Average**: ~0.560 (56%)
- **Elite**: >0.600 (60%)
- **Source**: [Basketball Reference](https://www.basketball-reference.com/about/glossary.html)

#### 5. **Effective Field Goal Percentage (eFG%)**
- **Formula**: `eFG% = (FGM + 0.5 * 3PM) / FGA`
- **Description**: Adjusts FG% to account for 3-pointers being worth 50% more
- **Range**: 0.000 to 1.500 (theoretical max if all 3PT)
- **League Average**: ~0.530 (53%)
- **Elite**: >0.580 (58%)

### Usage and Involvement Metrics

#### 6. **Usage Rate (USG%)**
- **Formula**: `USG% = 100 * ((FGA + 0.44 * FTA + TOV) * (Tm MP / 5)) / (MP * (Tm FGA + 0.44 * Tm FTA + Tm TOV))`
- **Description**: Percentage of team plays used by a player while on floor
- **Calculation**: Requires team totals (calculated in application layer)
- **Range**: 0.000 to 1.000
- **League Average**: ~0.200 (20%)
- **High Usage**: >0.300 (30%)
- **Note**: Stored as NULL in database, calculated by application when team data available

#### 7. **Assist Percentage (AST%)**
- **Formula**: `AST% = 100 * AST / (((MP / (Tm MP / 5)) * Tm FG) - FG)`
- **Description**: Percentage of teammate field goals assisted while on floor
- **Calculation**: Requires team totals
- **Range**: 0.000 to 1.000
- **Elite Playmaker**: >0.300 (30%)

#### 8. **Turnover Percentage (TOV%)**
- **Formula**: `TOV% = 100 * TOV / (FGA + 0.44 * FTA + TOV)`
- **Description**: Turnovers per 100 plays
- **Range**: 0.000 to 1.000
- **Good (Low)**: <0.100 (10%)

### Rebounding Metrics

#### 9. **Offensive Rebound Percentage (ORB%)**
- **Formula**: `ORB% = 100 * (ORB * (Tm MP / 5)) / (MP * (Tm ORB + Opp DRB))`
- **Description**: Percentage of available offensive rebounds grabbed
- **Calculation**: Requires opponent stats
- **Range**: 0.000 to 1.000
- **Elite**: >0.100 (10%)

#### 10. **Defensive Rebound Percentage (DRB%)**
- **Formula**: `DRB% = 100 * (DRB * (Tm MP / 5)) / (MP * (Tm DRB + Opp ORB))`
- **Description**: Percentage of available defensive rebounds grabbed
- **Range**: 0.000 to 1.000
- **Elite**: >0.250 (25%)

#### 11. **Total Rebound Percentage (TRB%)**
- **Formula**: `TRB% = 100 * (TRB * (Tm MP / 5)) / (MP * (Tm TRB + Opp TRB))`
- **Description**: Percentage of all available rebounds grabbed
- **Range**: 0.000 to 1.000
- **Elite**: >0.150 (15%)

### Impact Metrics

#### 12. **Offensive Rating (ORtg)**
- **Formula**: `ORtg = (Points Produced / Possessions Used) * 100`
- **Simplified**: `ORtg ≈ (PTS / Possessions) * 100`
- **Description**: Points produced per 100 possessions
- **Calculation**: Complex, requires possession estimation
- **Range**: 50 to 150+
- **League Average**: ~110
- **Elite**: >120

#### 13. **Defensive Rating (DRtg)**
- **Formula**: `DRtg = (Points Allowed / Possessions) * 100`
- **Description**: Points allowed per 100 possessions
- **Calculation**: Very complex, requires team defensive data and player on/off splits
- **Range**: 80 to 130+
- **League Average**: ~110
- **Elite (Low)**: <105

#### 14. **Net Rating (NetRtg)**
- **Formula**: `NetRtg = ORtg - DRtg`
- **Description**: Point differential per 100 possessions
- **Range**: -50 to +50
- **Positive**: Player's team outscores opponents when on floor
- **Elite**: >+10

#### 15. **Player Efficiency Rating (PER)**
- **Formula**: Complex multi-step calculation (see [Basketball Reference](https://www.basketball-reference.com/about/per.html))
- **Description**: Comprehensive per-minute efficiency metric
- **Calculation**: 
  1. Calculate unadjusted PER from box score stats
  2. Adjust for pace
  3. Adjust for league average
  4. Scale to league average of 15.0
- **Range**: 0 to 40+
- **League Average**: 15.0
- **All-Star**: >20.0
- **MVP**: >25.0
- **Note**: Full calculation requires league averages and pace data

#### 16. **Game Score (GmSc)**
- **Formula**: `GmSc = PTS + 0.4*FGM - 0.7*FGA - 0.4*(FTA-FTM) + 0.7*ORB + 0.3*DRB + STL + 0.7*AST + 0.7*BLK - 0.4*PF - TOV`
- **Description**: John Hollinger's single-game performance metric
- **Calculation**: Weighted sum of box score stats
- **Range**: -20 to 60+
- **Good Game**: >15
- **Great Game**: >25
- **Historic**: >40
- **Source**: [Basketball Reference](https://www.basketball-reference.com/about/glossary.html)
- **✅ Calculated by Database Trigger**

#### 17. **Box Plus/Minus (BPM)**
- **Formula**: Regression-based metric using box score stats
- **Description**: Estimates point differential per 100 possessions
- **Calculation**: Complex regression requiring historical data
- **Range**: -15 to +15
- **League Average**: 0.0
- **All-Star**: >+4.0
- **MVP**: >+8.0
- **Note**: Requires advanced statistical modeling

---

## 🏀 Team Advanced Statistics

### Dean Oliver's Four Factors

The "Four Factors" are the key determinants of basketball success, identified by Dean Oliver in "Basketball on Paper". Listed in order of importance:

#### 1. **Shooting Efficiency (eFG%)**
- **Weight**: ~40% of winning
- **Formula**: `eFG% = (FGM + 0.5 * 3PM) / FGA`
- **Description**: Most important factor
- **✅ Calculated by Database Trigger**

#### 2. **Turnover Rate (TOV%)**
- **Weight**: ~25% of winning
- **Formula**: `TOV% = TOV / Possessions * 100`
- **Description**: Protecting the ball
- **Good (Low)**: <12%
- **✅ Calculated by Database Trigger**

#### 3. **Rebounding (ORB%, DRB%)**
- **Weight**: ~20% of winning
- **Offensive Formula**: `ORB% = ORB / (ORB + Opp DRB)`
- **Defensive Formula**: `DRB% = DRB / (DRB + Opp ORB)`
- **Description**: Controlling possessions
- **✅ Calculated by Database Trigger**

#### 4. **Free Throw Rate (FTr)**
- **Weight**: ~15% of winning
- **Formula**: `FTr = FTA / FGA`
- **Description**: Getting to the line
- **Good**: >0.250 (25%)
- **✅ Calculated by Database Trigger**

### Pace and Efficiency

#### 18. **Pace**
- **Formula**: `Pace = 48 * ((Tm Poss + Opp Poss) / (2 * (Tm MP / 5)))`
- **Simplified**: `Pace = Possessions per 48 minutes`
- **Description**: Tempo of the game
- **Range**: 90 to 110+
- **League Average**: ~100 (2024-25 season)
- **Fast**: >105
- **Slow**: <95

#### 19. **Possessions**
- **Formula**: `Poss = 0.5 * ((FGA + 0.44 * FTA - ORB + TOV) + (Opp FGA + 0.44 * Opp FTA - Opp ORB + Opp TOV))`
- **Description**: Estimated number of possessions
- **Typical Game**: 95-105 possessions per team

#### 20. **Offensive Rating (ORtg)**
- **Formula**: `ORtg = (Points / Possessions) * 100`
- **Description**: Points scored per 100 possessions
- **Range**: 90 to 130+
- **League Average**: ~115 (2024-25)
- **Elite**: >120
- **✅ Calculated by Database Trigger** (when possessions available)

#### 21. **Defensive Rating (DRtg)**
- **Formula**: `DRtg = (Opp Points / Possessions) * 100`
- **Description**: Points allowed per 100 possessions
- **Range**: 90 to 130+
- **League Average**: ~115 (2024-25)
- **Elite (Low)**: <110

#### 22. **Net Rating (NetRtg)**
- **Formula**: `NetRtg = ORtg - DRtg`
- **Description**: Point differential per 100 possessions
- **Range**: -30 to +30
- **Championship Team**: >+8
- **✅ Calculated by Database Trigger**

### Additional Team Metrics

#### 23. **True Shooting Percentage (TS%)**
- **Formula**: `TS% = PTS / (2 * (FGA + 0.44 * FTA))`
- **Description**: Overall team shooting efficiency
- **✅ Calculated by Database Trigger**

#### 24. **Assist to Turnover Ratio (AST/TO)**
- **Formula**: `AST/TO = AST / TOV`
- **Description**: Ball movement vs. ball security
- **Range**: 0.5 to 2.5
- **Good**: >1.5
- **Elite**: >2.0
- **✅ Calculated by Database Trigger**

#### 25. **Steal Percentage (STL%)**
- **Formula**: `STL% = (STL / Opp Poss) * 100`
- **Description**: Steals per 100 opponent possessions
- **Range**: 5% to 15%
- **Good**: >10%

#### 26. **Block Percentage (BLK%)**
- **Formula**: `BLK% = (BLK / Opp FGA) * 100`
- **Description**: Blocks per 100 opponent field goal attempts
- **Range**: 2% to 10%
- **Good**: >6%

---

## 🔧 Implementation Notes

### Database Triggers

The following metrics are **automatically calculated** by PostgreSQL triggers on INSERT/UPDATE:

**Player Stats** (`calculate_player_advanced_stats()`):
- ✅ FG%, 3P%, FT%
- ✅ TS%, eFG%
- ✅ Game Score
- ✅ Net Rating (if ORtg/DRtg provided)

**Team Stats** (`calculate_team_advanced_stats()`):
- ✅ FG%, 3P%, FT%
- ✅ TS%, eFG%
- ✅ Free Throw Rate
- ✅ Assist/Turnover Ratio
- ✅ Offensive Rating (if possessions provided)
- ✅ Net Rating

### Application Layer Calculations

The following metrics **require team/opponent context** and must be calculated in the Go application layer:

**Player Stats** (require team totals):
- ⚙️ Usage Rate (USG%)
- ⚙️ Assist Percentage (AST%)
- ⚙️ Turnover Percentage (TOV%)
- ⚙️ Rebound Percentages (ORB%, DRB%, TRB%)
- ⚙️ Offensive/Defensive Rating (ORtg, DRtg)
- ⚙️ Player Efficiency Rating (PER)
- ⚙️ Box Plus/Minus (BPM)

**Team Stats** (require opponent data):
- ⚙️ Pace
- ⚙️ Possessions
- ⚙️ Defensive Rating (DRtg)
- ⚙️ Turnover Rate (TOV%)
- ⚙️ Rebound Percentages (ORB%, DRB%, TRB%)
- ⚙️ Steal Percentage (STL%)
- ⚙️ Block Percentage (BLK%)

### Calculation Pipeline

```
1. ESPN API → Raw Box Score Data
   ↓
2. Database INSERT → player_game_stats, team_game_stats
   ↓
3. Database Triggers → Calculate simple metrics (TS%, eFG%, GmSc, etc.)
   ↓
4. Application Layer → Calculate context-dependent metrics (USG%, PER, Pace, etc.)
   ↓
5. Database UPDATE → Store calculated metrics
   ↓
6. Materialized Views → Pre-aggregate season averages
```

---

## 📚 References

1. **Basketball Reference** - [Glossary of Statistical Terms](https://www.basketball-reference.com/about/glossary.html)
2. **NBA.com Stats** - [Advanced Stats Glossary](https://www.nba.com/stats/help/glossary)
3. **Dean Oliver** - "Basketball on Paper" (Four Factors)
4. **John Hollinger** - ESPN Basketball Analytics (PER, Game Score)
5. **Basketball Analytics** - [Advanced Metrics Explained](https://squared2020.com/2017/09/05/introduction-to-olivers-four-factors/)

---

## 🎯 Integration with Fortuna

### Holocron Integration
- Advanced stats feed into **bet opportunity detection**
- Player efficiency metrics help identify **value bets** on player props
- Team Four Factors correlate with **game outcome predictions**

### Alexandria Integration
- Advanced stats stored alongside **odds data** for correlation analysis
- Historical advanced stats enable **closing line value (CLV)** analysis
- Pace and efficiency metrics improve **total (over/under) predictions**

### ML Model Features
All advanced stats are exposed via Minerva's `/api/v1/ml/features` endpoint for:
- Game outcome prediction models
- Player prop prediction models
- Live betting models (using real-time advanced stats)
- Closing line value (CLV) models

---

## 🚀 Future Enhancements

### Phase 2 (Current)
- ✅ Database schema with advanced stat columns
- ✅ Triggers for automatic calculation
- ⏳ Application layer for context-dependent metrics

### Phase 3
- ⏳ Materialized views for season averages
- ⏳ Historical trend analysis (rolling averages)
- ⏳ Player comparison endpoints

### Phase 4
- ⏳ Real-time advanced stats during live games
- ⏳ Advanced stat alerts (e.g., "Player X on pace for 30 PER game")
- ⏳ Integration with live betting models

### Phase 5
- ⏳ Custom advanced metrics (proprietary formulas)
- ⏳ Multi-sport expansion (NFL, MLB, NHL)
- ⏳ Advanced visualization in web client

---

**Last Updated**: 2025-11-14  
**Version**: 2.0  
**Author**: Minerva Sports Analytics System

