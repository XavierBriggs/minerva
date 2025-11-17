# Minerva Sports Analytics - Complete Implementation Plan

**Created**: November 14, 2025  
**Status**: Active Development  
**Focus**: NBA (Multi-sport ready architecture)

---

## 🎯 Executive Summary

Minerva is Fortuna's **sports analytics and data ingestion system**, designed to collect, process, and serve player statistics, team data, and game information for training ML models and providing real-time insights. The system is built with **multi-sport expansion** in mind while initially focusing on NBA.

### Core Objectives
1. **Data Ingestion**: Collect live and historical sports data (ESPN primary, Google Sports fallback)
2. **Real-time Updates**: Stream live game updates via WebSocket
3. **Historical Backfill**: Load past seasons for ML training
4. **ML-Ready Features**: Expose endpoints for model training
5. **User-Friendly UI**: Seamlessly integrated into Fortuna web client
6. **Multi-Sport Ready**: Modular design for easy sport addition

---

## 🏗️ System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     MINERVA SYSTEM                           │
│                                                              │
│  ┌──────────────┐      ┌──────────────┐                    │
│  │   ESPN API   │      │ Google Sports│                    │
│  │  (Primary)   │      │  (Fallback)  │                    │
│  └──────┬───────┘      └──────┬───────┘                    │
│         │                     │                             │
│         └─────────┬───────────┘                             │
│                   ▼                                         │
│         ┌─────────────────────┐                            │
│         │  Data Reconciliation│                            │
│         │      Engine         │                            │
│         └──────────┬──────────┘                            │
│                    ▼                                        │
│         ┌─────────────────────┐                            │
│         │   Atlas Database    │                            │
│         │  (PostgreSQL)       │                            │
│         │                     │                            │
│         │  • Games            │                            │
│         │  • Teams            │                            │
│         │  • Players          │                            │
│         │  • Stats            │                            │
│         │  • Seasons          │                            │
│         └──────────┬──────────┘                            │
│                    │                                        │
│         ┌──────────┴──────────┐                            │
│         ▼                     ▼                             │
│  ┌─────────────┐      ┌──────────────┐                    │
│  │ Redis Cache │      │ Redis Streams│                    │
│  │  (State)    │      │ (Real-time)  │                    │
│  └─────────────┘      └──────┬───────┘                    │
│                               │                             │
└───────────────────────────────┼─────────────────────────────┘
                                │
                ┌───────────────┴───────────────┐
                ▼                               ▼
        ┌──────────────┐              ┌─────────────────┐
        │  REST API    │              │  WS Broadcaster │
        │  (Port 8080) │              │  (Port 8081)    │
        └──────┬───────┘              └────────┬────────┘
               │                               │
               └───────────┬───────────────────┘
                           ▼
                ┌──────────────────────┐
                │   API Gateway        │
                │  (Port 8081)         │
                └──────────┬───────────┘
                           ▼
                ┌──────────────────────┐
                │  Fortuna Web Client  │
                │  (Next.js/React)     │
                └──────────────────────┘
```

---

## 📊 Database Schema (Multi-Sport Design)

> **📖 Full Database Design**: See [ATLAS_DATABASE_DESIGN.md](./ATLAS_DATABASE_DESIGN.md) for comprehensive schema documentation with best practices.

### Design Principles (Research-Based)
1. **Normalization (3NF) with Strategic Denormalization**: Reduce redundancy while optimizing read-heavy queries
2. **Temporal Data Management**: Track player-team relationships over time with `player_team_history` table
3. **Partitioning Strategy**: Partition large tables (games, stats) by season for performance
4. **Multi-Sport Extensibility**: Sport-agnostic core with JSONB for sport-specific attributes
5. **Performance Optimization**: Strategic indexing, composite indexes, GIN indexes for JSONB
6. **Data Integrity**: Foreign keys, check constraints, unique constraints, NOT NULL enforcement
7. **Scalability**: Designed for horizontal scaling, read replicas, connection pooling, sport-based sharding

### Key Improvements from Research

#### 1. **Player-Team History Table** (Critical Addition)
- **Problem**: Players change teams mid-season (trades, signings)
- **Solution**: `player_team_history` table with temporal tracking
- **Benefits**: 
  - Point-in-time queries (e.g., "Who was on Lakers roster on 2024-03-15?")
  - Accurate historical analysis
  - Trade tracking and contract management

#### 2. **Table Partitioning** (Performance)
- **Tables to Partition**: `games`, `player_game_stats`, `team_game_stats`
- **Partition Key**: `season_id` or `game_date`
- **Benefits**:
  - 10-100x faster queries on recent data
  - Easier data archival
  - Improved maintenance operations

#### 3. **Materialized Views** (Query Optimization)
- **`player_season_averages`**: Pre-calculated season stats (PPG, RPG, APG)
- **Refresh Strategy**: Nightly or after each game day
- **Benefits**: Sub-millisecond queries for common aggregations

#### 4. **Full-Text Search** (User Experience)
- **GIN Index** on `players.full_name` using `to_tsvector`
- **Benefits**: Fast, fuzzy player name search
- **Example**: "lebr" matches "LeBron James"

#### 5. **Calculated Fields with Triggers** (Data Consistency)
- Auto-calculate shooting percentages on insert/update
- Ensures consistency across all stats records
- Reduces application logic complexity

#### 6. **Advanced Indexing Strategies**
- **Composite Indexes**: Multi-column queries (e.g., `sport + status + date`)
- **Partial Indexes**: Filtered indexes (e.g., only active players, live games)
- **GIN Indexes**: JSONB fields for sport-specific data

### Core Tables (Summary)

#### 1. `seasons`
```sql
CREATE TABLE seasons (
  season_id SERIAL PRIMARY KEY,
  sport VARCHAR(50) NOT NULL,           -- 'basketball_nba', 'football_nfl', etc.
  season_year VARCHAR(20) NOT NULL,     -- '2024-25', '2024', etc.
  start_date DATE,
  end_date DATE,
  is_active BOOLEAN DEFAULT false,
  metadata JSONB,                       -- Sport-specific season info
  UNIQUE(sport, season_year)
);
```

#### 2. `teams`
```sql
CREATE TABLE teams (
  team_id SERIAL PRIMARY KEY,
  sport VARCHAR(50) NOT NULL,
  external_id VARCHAR(100),             -- ESPN team ID
  abbreviation VARCHAR(10) NOT NULL,
  full_name VARCHAR(100) NOT NULL,
  short_name VARCHAR(50),
  city VARCHAR(100),
  conference VARCHAR(50),               -- 'Eastern', 'AFC', etc.
  division VARCHAR(50),
  logo_url TEXT,
  colors JSONB,                         -- {primary: '#...', secondary: '#...'}
  metadata JSONB,                       -- Sport-specific team data
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  UNIQUE(sport, external_id)
);
CREATE INDEX idx_teams_sport ON teams(sport);
CREATE INDEX idx_teams_abbreviation ON teams(sport, abbreviation);
```

#### 3. `players`
```sql
CREATE TABLE players (
  player_id SERIAL PRIMARY KEY,
  sport VARCHAR(50) NOT NULL,
  external_id VARCHAR(100),             -- ESPN player ID
  first_name VARCHAR(100),
  last_name VARCHAR(100),
  full_name VARCHAR(200),
  jersey_number VARCHAR(10),
  position VARCHAR(20),                 -- 'PG', 'QB', 'P', etc.
  height VARCHAR(20),
  weight INTEGER,
  birth_date DATE,
  college VARCHAR(100),
  draft_year INTEGER,
  draft_round INTEGER,
  draft_pick INTEGER,
  headshot_url TEXT,
  metadata JSONB,                       -- Sport-specific player data
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  UNIQUE(sport, external_id)
);
CREATE INDEX idx_players_sport ON players(sport);
CREATE INDEX idx_players_name ON players(sport, last_name, first_name);
```

#### 4. `player_seasons`
```sql
CREATE TABLE player_seasons (
  player_season_id SERIAL PRIMARY KEY,
  player_id INTEGER REFERENCES players(player_id),
  team_id INTEGER REFERENCES teams(team_id),
  season_id INTEGER REFERENCES seasons(season_id),
  jersey_number VARCHAR(10),
  position VARCHAR(20),
  is_active BOOLEAN DEFAULT true,
  stats JSONB,                          -- Season aggregate stats
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  UNIQUE(player_id, season_id)
);
CREATE INDEX idx_player_seasons_player ON player_seasons(player_id);
CREATE INDEX idx_player_seasons_team ON player_seasons(team_id);
CREATE INDEX idx_player_seasons_season ON player_seasons(season_id);
```

#### 5. `games`
```sql
CREATE TABLE games (
  game_id SERIAL PRIMARY KEY,
  sport VARCHAR(50) NOT NULL,
  external_id VARCHAR(100),             -- ESPN game ID
  season_id INTEGER REFERENCES seasons(season_id),
  home_team_id INTEGER REFERENCES teams(team_id),
  away_team_id INTEGER REFERENCES teams(team_id),
  game_date TIMESTAMP NOT NULL,
  venue VARCHAR(200),
  game_status VARCHAR(20),              -- 'scheduled', 'in_progress', 'final', 'postponed'
  period INTEGER,                       -- Quarter, Inning, Period, etc.
  time_remaining VARCHAR(20),
  home_score INTEGER,
  away_score INTEGER,
  attendance INTEGER,
  broadcast_info JSONB,                 -- TV, streaming info
  game_data JSONB,                      -- Sport-specific game data
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  UNIQUE(sport, external_id)
);
CREATE INDEX idx_games_sport ON games(sport);
CREATE INDEX idx_games_date ON games(game_date);
CREATE INDEX idx_games_status ON games(game_status);
CREATE INDEX idx_games_teams ON games(home_team_id, away_team_id);
CREATE INDEX idx_games_season ON games(season_id);
```

#### 6. `player_game_stats`
```sql
CREATE TABLE player_game_stats (
  stat_id SERIAL PRIMARY KEY,
  game_id INTEGER REFERENCES games(game_id),
  player_id INTEGER REFERENCES players(player_id),
  team_id INTEGER REFERENCES teams(team_id),
  
  -- Universal Stats (common across sports)
  minutes_played NUMERIC(5,2),
  starter BOOLEAN DEFAULT false,
  
  -- Basketball-Specific (NULL for other sports)
  points INTEGER,
  rebounds INTEGER,
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
  offensive_rebounds INTEGER,
  defensive_rebounds INTEGER,
  plus_minus INTEGER,
  
  -- Advanced Metrics
  true_shooting_pct NUMERIC(5,4),
  effective_fg_pct NUMERIC(5,4),
  
  -- Extensible for other sports
  sport_specific_stats JSONB,           -- Football: passing_yards, rushing_yards, etc.
  
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  UNIQUE(game_id, player_id)
);
CREATE INDEX idx_player_game_stats_game ON player_game_stats(game_id);
CREATE INDEX idx_player_game_stats_player ON player_game_stats(player_id);
CREATE INDEX idx_player_game_stats_team ON player_game_stats(team_id);
```

#### 7. `team_game_stats`
```sql
CREATE TABLE team_game_stats (
  stat_id SERIAL PRIMARY KEY,
  game_id INTEGER REFERENCES games(game_id),
  team_id INTEGER REFERENCES teams(team_id),
  is_home BOOLEAN,
  
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
  
  -- Extensible
  sport_specific_stats JSONB,
  
  created_at TIMESTAMP DEFAULT NOW(),
  UNIQUE(game_id, team_id)
);
CREATE INDEX idx_team_game_stats_game ON team_game_stats(game_id);
CREATE INDEX idx_team_game_stats_team ON team_game_stats(team_id);
```

#### 8. `odds_mappings`
```sql
CREATE TABLE odds_mappings (
  mapping_id SERIAL PRIMARY KEY,
  sport VARCHAR(50) NOT NULL,
  minerva_game_id INTEGER REFERENCES games(game_id),
  minerva_team_id INTEGER REFERENCES teams(team_id),
  alexandria_event_id VARCHAR(100),     -- Links to Mercury/Alexandria
  alexandria_participant_name VARCHAR(200),
  mapping_type VARCHAR(20),             -- 'game', 'team', 'player'
  confidence NUMERIC(3,2),              -- Matching confidence score
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);
CREATE INDEX idx_odds_mappings_minerva_game ON odds_mappings(minerva_game_id);
CREATE INDEX idx_odds_mappings_alexandria ON odds_mappings(alexandria_event_id);
```

#### 9. `backfill_jobs`
```sql
CREATE TABLE backfill_jobs (
  job_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  sport VARCHAR(50) NOT NULL,
  job_type VARCHAR(20) NOT NULL,       -- 'season', 'date_range', 'game'
  season_id VARCHAR(20),
  start_date DATE,
  end_date DATE,
  status VARCHAR(20) NOT NULL,         -- 'queued', 'running', 'completed', 'failed'
  status_message TEXT,
  progress_current INTEGER DEFAULT 0,
  progress_total INTEGER DEFAULT 0,
  last_error TEXT,
  created_at TIMESTAMP DEFAULT NOW(),
  started_at TIMESTAMP,
  completed_at TIMESTAMP,
  updated_at TIMESTAMP DEFAULT NOW()
);
CREATE INDEX idx_backfill_jobs_status ON backfill_jobs(status);
CREATE INDEX idx_backfill_jobs_created ON backfill_jobs(created_at DESC);
```

---

## 🔧 Backend Implementation

### 1. Multi-Sport Abstraction Layer

#### Sport Interface
```go
// internal/sports/sport.go
package sports

import (
	"context"
	"time"
)

// Sport defines the interface all sports must implement
type Sport interface {
	// Metadata
	Key() string                    // "basketball_nba", "football_nfl"
	DisplayName() string            // "NBA Basketball"
	
	// Data Ingestion
	FetchLiveGames(ctx context.Context) ([]Game, error)
	FetchGamesByDate(ctx context.Context, date time.Time) ([]Game, error)
	FetchGameDetails(ctx context.Context, gameID string) (*GameDetails, error)
	FetchPlayerStats(ctx context.Context, gameID string) ([]PlayerStats, error)
	
	// Backfill
	FetchSeasonGames(ctx context.Context, seasonID string) ([]Game, error)
	
	// Parsing & Normalization
	ParseGame(data interface{}) (*Game, error)
	ParsePlayerStats(data interface{}) ([]PlayerStats, error)
	NormalizeTeamName(name string) string
	NormalizePlayerName(name string) string
	
	// Validation
	ValidateGame(game *Game) error
	ValidatePlayerStats(stats *PlayerStats) error
}

// Common data structures
type Game struct {
	ExternalID    string
	Sport         string
	HomeTeam      Team
	AwayTeam      Team
	GameDate      time.Time
	Status        string
	HomeScore     *int
	AwayScore     *int
	Period        *int
	TimeRemaining *string
	Venue         *string
	Metadata      map[string]interface{}
}

type PlayerStats struct {
	ExternalID    string
	PlayerName    string
	TeamAbbr      string
	Position      string
	MinutesPlayed *float64
	Starter       bool
	// Basketball fields
	Points        *int
	Rebounds      *int
	Assists       *int
	// ... other stats
	// Extensible
	SportSpecific map[string]interface{}
}
```

#### Basketball NBA Implementation
```go
// internal/sports/basketball_nba/basketball_nba.go
package basketball_nba

import (
	"context"
	"github.com/fortuna/minerva/internal/sports"
	"github.com/fortuna/minerva/internal/ingest/espn"
)

type BasketballNBA struct {
	espnClient *espn.Client
}

func New(espnClient *espn.Client) *BasketballNBA {
	return &BasketballNBA{
		espnClient: espnClient,
	}
}

func (b *BasketballNBA) Key() string {
	return "basketball_nba"
}

func (b *BasketballNBA) DisplayName() string {
	return "NBA Basketball"
}

func (b *BasketballNBA) FetchLiveGames(ctx context.Context) ([]sports.Game, error) {
	// ESPN API call for NBA scoreboard
	data, err := b.espnClient.FetchScoreboard(ctx, espn.BasketballNBA, time.Time{})
	if err != nil {
		return nil, err
	}
	
	// Parse ESPN response into common Game struct
	games := b.parseESPNScoreboard(data)
	return games, nil
}

// ... implement other interface methods
```

### 2. Sport Registry
```go
// internal/sports/registry.go
package sports

import (
	"fmt"
	"sync"
)

var (
	registry = make(map[string]Sport)
	mu       sync.RWMutex
)

// Register adds a sport to the registry
func Register(sport Sport) {
	mu.Lock()
	defer mu.Unlock()
	registry[sport.Key()] = sport
}

// Get retrieves a sport by key
func Get(key string) (Sport, error) {
	mu.RLock()
	defer mu.RUnlock()
	
	sport, ok := registry[key]
	if !ok {
		return nil, fmt.Errorf("sport not found: %s", key)
	}
	return sport, nil
}

// List returns all registered sports
func List() []Sport {
	mu.RLock()
	defer mu.RUnlock()
	
	sports := make([]Sport, 0, len(registry))
	for _, sport := range registry {
		sports = append(sports, sport)
	}
	return sports
}
```

### 3. Service Layer Refactoring

#### Sport-Aware Services
```go
// internal/service/games.go
package service

type GamesService struct {
	db           *store.Database
	sportRegistry map[string]sports.Sport
}

func (s *GamesService) GetLiveGames(ctx context.Context, sport string) ([]store.Game, error) {
	// Get sport implementation
	sportImpl, err := sports.Get(sport)
	if err != nil {
		return nil, err
	}
	
	// Fetch from sport-specific source
	games, err := sportImpl.FetchLiveGames(ctx)
	if err != nil {
		return nil, err
	}
	
	// Store in database
	for _, game := range games {
		if err := s.db.UpsertGame(&game); err != nil {
			log.Printf("Failed to store game: %v", err)
		}
	}
	
	// Return from database (source of truth)
	return s.db.GetGamesBySport(ctx, sport, "in_progress")
}
```

### 4. API Endpoints (Multi-Sport)

```go
// internal/api/rest/games_handlers.go

// GET /api/v1/sports
func (s *Server) HandleListSports(w http.ResponseWriter, r *http.Request) {
	sports := sports.List()
	respondJSON(w, http.StatusOK, sports)
}

// GET /api/v1/{sport}/games/live
func (s *Server) HandleGetLiveGames(w http.ResponseWriter, r *http.Request) {
	sport := chi.URLParam(r, "sport")  // e.g., "basketball_nba"
	
	games, err := s.gamesService.GetLiveGames(r.Context(), sport)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, games)
}

// GET /api/v1/{sport}/games/history?date=2025-11-13
func (s *Server) HandleGetGamesByDate(w http.ResponseWriter, r *http.Request) {
	sport := chi.URLParam(r, "sport")
	dateStr := r.URL.Query().Get("date")
	
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid date format")
		return
	}
	
	games, err := s.gamesService.GetGamesByDate(r.Context(), sport, date)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, games)
}

// GET /api/v1/{sport}/players/search?q=lebron
func (s *Server) HandleSearchPlayers(w http.ResponseWriter, r *http.Request) {
	sport := chi.URLParam(r, "sport")
	query := r.URL.Query().Get("q")
	
	players, err := s.playersService.SearchPlayers(r.Context(), sport, query)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, players)
}

// GET /api/v1/{sport}/teams
func (s *Server) HandleGetTeams(w http.ResponseWriter, r *http.Request) {
	sport := chi.URLParam(r, "sport")
	
	teams, err := s.teamsService.GetTeams(r.Context(), sport)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, teams)
}

// GET /api/v1/{sport}/players/{playerId}/stats?season=2024-25
func (s *Server) HandleGetPlayerStats(w http.ResponseWriter, r *http.Request) {
	sport := chi.URLParam(r, "sport")
	playerID := chi.URLParam(r, "playerId")
	seasonID := r.URL.Query().Get("season")
	
	stats, err := s.playersService.GetPlayerSeasonStats(r.Context(), sport, playerID, seasonID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, stats)
}

// POST /api/v1/{sport}/backfill
func (s *Server) HandleTriggerBackfill(w http.ResponseWriter, r *http.Request) {
	sport := chi.URLParam(r, "sport")
	
	var req BackfillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	
	req.Sport = sport
	job, err := s.backfillService.Enqueue(r.Context(), &req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusAccepted, map[string]interface{}{"job": job})
}
```

---

## 🎨 Frontend Implementation

### 1. Sport-Aware Components

#### Sport Context
```typescript
// contexts/SportContext.tsx
import { createContext, useContext, useState, ReactNode } from 'react';

type Sport = 'basketball_nba' | 'football_nfl' | 'baseball_mlb';

interface SportContextType {
  currentSport: Sport;
  setSport: (sport: Sport) => void;
  availableSports: Sport[];
}

const SportContext = createContext<SportContextType | undefined>(undefined);

export function SportProvider({ children }: { children: ReactNode }) {
  const [currentSport, setCurrentSport] = useState<Sport>('basketball_nba');
  
  const availableSports: Sport[] = ['basketball_nba']; // Expand as we add sports
  
  return (
    <SportContext.Provider value={{ 
      currentSport, 
      setSport: setCurrentSport,
      availableSports 
    }}>
      {children}
    </SportContext.Provider>
  );
}

export function useSport() {
  const context = useContext(SportContext);
  if (!context) throw new Error('useSport must be used within SportProvider');
  return context;
}
```

#### Sport Selector Component
```typescript
// components/minerva/SportSelector.tsx
import { useSport } from '@/contexts/SportContext';
import { Basketball, Football, Baseball } from 'lucide-react';

const sportIcons = {
  basketball_nba: Basketball,
  football_nfl: Football,
  baseball_mlb: Baseball,
};

const sportLabels = {
  basketball_nba: 'NBA',
  football_nfl: 'NFL',
  baseball_mlb: 'MLB',
};

export function SportSelector() {
  const { currentSport, setSport, availableSports } = useSport();
  
  return (
    <div className="flex gap-2">
      {availableSports.map(sport => {
        const Icon = sportIcons[sport];
        const isActive = sport === currentSport;
        
        return (
          <button
            key={sport}
            onClick={() => setSport(sport)}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg transition-colors ${
              isActive 
                ? 'bg-primary text-primary-foreground' 
                : 'bg-card hover:bg-muted'
            }`}
          >
            <Icon className="h-5 w-5" />
            <span className="font-semibold">{sportLabels[sport]}</span>
          </button>
        );
      })}
    </div>
  );
}
```

### 2. Enhanced Player Stats Display

#### Player Card Component
```typescript
// components/minerva/players/PlayerCard.tsx
import { Player, PlayerSeasonStats } from '@/lib/minerva-api';
import { TrendingUp, TrendingDown } from 'lucide-react';

interface PlayerCardProps {
  player: Player;
  stats?: PlayerSeasonStats;
  onClick?: () => void;
}

export function PlayerCard({ player, stats, onClick }: PlayerCardProps) {
  return (
    <div 
      onClick={onClick}
      className="bg-card border border-border rounded-lg p-4 hover:border-primary transition-colors cursor-pointer"
    >
      <div className="flex items-start gap-4">
        {/* Player Headshot */}
        <div className="relative">
          <img 
            src={player.headshot_url || '/default-player.png'} 
            alt={player.full_name}
            className="w-20 h-20 rounded-lg object-cover"
          />
          <div className="absolute -bottom-2 -right-2 bg-primary text-primary-foreground text-xs font-bold px-2 py-1 rounded">
            #{player.jersey_number}
          </div>
        </div>
        
        {/* Player Info */}
        <div className="flex-1">
          <h3 className="text-lg font-bold">{player.full_name}</h3>
          <p className="text-sm text-muted-foreground">
            {player.position} • {player.team_abbreviation}
          </p>
          <div className="flex gap-2 mt-1 text-xs text-muted-foreground">
            <span>{player.height}</span>
            <span>•</span>
            <span>{player.weight} lbs</span>
          </div>
        </div>
      </div>
      
      {/* Season Stats */}
      {stats && (
        <div className="mt-4 pt-4 border-t border-border">
          <div className="grid grid-cols-3 gap-4">
            <div>
              <div className="text-2xl font-bold">{stats.points_per_game.toFixed(1)}</div>
              <div className="text-xs text-muted-foreground">PPG</div>
            </div>
            <div>
              <div className="text-2xl font-bold">{stats.rebounds_per_game.toFixed(1)}</div>
              <div className="text-xs text-muted-foreground">RPG</div>
            </div>
            <div>
              <div className="text-2xl font-bold">{stats.assists_per_game.toFixed(1)}</div>
              <div className="text-xs text-muted-foreground">APG</div>
            </div>
          </div>
          
          {/* Shooting Percentages */}
          <div className="mt-3 flex gap-4 text-sm">
            <div>
              <span className="text-muted-foreground">FG:</span>
              <span className="ml-1 font-semibold">{(stats.field_goal_pct * 100).toFixed(1)}%</span>
            </div>
            <div>
              <span className="text-muted-foreground">3P:</span>
              <span className="ml-1 font-semibold">{(stats.three_point_pct * 100).toFixed(1)}%</span>
            </div>
            <div>
              <span className="text-muted-foreground">FT:</span>
              <span className="ml-1 font-semibold">{(stats.free_throw_pct * 100).toFixed(1)}%</span>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
```

#### Player Search Page
```typescript
// app/minerva/players/page.tsx
'use client';

import { useState, useEffect } from 'react';
import { useSport } from '@/contexts/SportContext';
import { minervaAPI, Player, PlayerSeasonStats } from '@/lib/minerva-api';
import { PlayerCard } from '@/components/minerva/players/PlayerCard';
import { Search, Filter } from 'lucide-react';

export default function PlayersPage() {
  const { currentSport } = useSport();
  const [searchQuery, setSearchQuery] = useState('');
  const [players, setPlayers] = useState<Player[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedPosition, setSelectedPosition] = useState<string>('all');
  
  useEffect(() => {
    if (searchQuery.length >= 2) {
      searchPlayers();
    }
  }, [searchQuery, currentSport]);
  
  const searchPlayers = async () => {
    try {
      setLoading(true);
      const results = await minervaAPI.searchPlayers(currentSport, searchQuery);
      setPlayers(results);
    } catch (err) {
      console.error('Failed to search players:', err);
    } finally {
      setLoading(false);
    }
  };
  
  const filteredPlayers = selectedPosition === 'all' 
    ? players 
    : players.filter(p => p.position === selectedPosition);
  
  return (
    <div className="max-w-7xl mx-auto p-8">
      <h1 className="text-4xl font-bold mb-8">Player Search</h1>
      
      {/* Search Bar */}
      <div className="flex gap-4 mb-8">
        <div className="flex-1 relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-5 w-5 text-muted-foreground" />
          <input
            type="text"
            placeholder="Search players by name..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full pl-10 pr-4 py-3 bg-card border border-border rounded-lg"
          />
        </div>
        
        {/* Position Filter */}
        <select
          value={selectedPosition}
          onChange={(e) => setSelectedPosition(e.target.value)}
          className="px-4 py-3 bg-card border border-border rounded-lg"
        >
          <option value="all">All Positions</option>
          <option value="PG">Point Guard</option>
          <option value="SG">Shooting Guard</option>
          <option value="SF">Small Forward</option>
          <option value="PF">Power Forward</option>
          <option value="C">Center</option>
        </select>
      </div>
      
      {/* Results */}
      {loading ? (
        <div className="text-center py-12">Loading...</div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {filteredPlayers.map(player => (
            <PlayerCard key={player.player_id} player={player} />
          ))}
        </div>
      )}
      
      {!loading && filteredPlayers.length === 0 && searchQuery.length >= 2 && (
        <div className="text-center py-12 text-muted-foreground">
          No players found matching "{searchQuery}"
        </div>
      )}
    </div>
  );
}
```

### 3. Enhanced Box Score Modal

```typescript
// components/minerva/live-games/EnhancedBoxScoreModal.tsx
import { useState, useEffect } from 'react';
import { minervaAPI, PlayerGameStats } from '@/lib/minerva-api';
import { X, TrendingUp, Award } from 'lucide-react';

interface EnhancedBoxScoreModalProps {
  gameId: string | null;
  isOpen: boolean;
  onClose: () => void;
}

export function EnhancedBoxScoreModal({ gameId, isOpen, onClose }: EnhancedBoxScoreModalProps) {
  const [stats, setStats] = useState<PlayerGameStats[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedTeam, setSelectedTeam] = useState<'home' | 'away'>('home');
  
  useEffect(() => {
    if (isOpen && gameId) {
      fetchStats();
    }
  }, [isOpen, gameId]);
  
  const fetchStats = async () => {
    if (!gameId) return;
    
    try {
      setLoading(true);
      const data = await minervaAPI.getGameStats(gameId);
      setStats(data);
    } catch (err) {
      console.error('Failed to fetch stats:', err);
    } finally {
      setLoading(false);
    }
  };
  
  if (!isOpen) return null;
  
  const homeStats = stats.filter(s => s.is_home);
  const awayStats = stats.filter(s => !s.is_home);
  const displayStats = selectedTeam === 'home' ? homeStats : awayStats;
  
  // Find game leaders
  const topScorer = [...stats].sort((a, b) => (b.points || 0) - (a.points || 0))[0];
  const topRebounder = [...stats].sort((a, b) => (b.rebounds || 0) - (a.rebounds || 0))[0];
  const topAssister = [...stats].sort((a, b) => (b.assists || 0) - (a.assists || 0))[0];
  
  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-background border border-border rounded-lg max-w-6xl w-full max-h-[90vh] overflow-hidden flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-border">
          <h2 className="text-2xl font-bold">Box Score</h2>
          <button onClick={onClose} className="p-2 hover:bg-muted rounded-lg">
            <X className="h-5 w-5" />
          </button>
        </div>
        
        {/* Game Leaders */}
        <div className="p-6 bg-muted/30 border-b border-border">
          <h3 className="text-sm font-semibold text-muted-foreground mb-3">GAME LEADERS</h3>
          <div className="grid grid-cols-3 gap-4">
            <div className="flex items-center gap-3">
              <Award className="h-5 w-5 text-yellow-500" />
              <div>
                <div className="text-xs text-muted-foreground">Points</div>
                <div className="font-semibold">{topScorer?.player_name}</div>
                <div className="text-sm text-primary">{topScorer?.points} PTS</div>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <Award className="h-5 w-5 text-blue-500" />
              <div>
                <div className="text-xs text-muted-foreground">Rebounds</div>
                <div className="font-semibold">{topRebounder?.player_name}</div>
                <div className="text-sm text-primary">{topRebounder?.rebounds} REB</div>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <Award className="h-5 w-5 text-green-500" />
              <div>
                <div className="text-xs text-muted-foreground">Assists</div>
                <div className="font-semibold">{topAssister?.player_name}</div>
                <div className="text-sm text-primary">{topAssister?.assists} AST</div>
              </div>
            </div>
          </div>
        </div>
        
        {/* Team Tabs */}
        <div className="flex border-b border-border">
          <button
            onClick={() => setSelectedTeam('home')}
            className={`flex-1 px-6 py-3 font-semibold transition-colors ${
              selectedTeam === 'home' 
                ? 'bg-primary text-primary-foreground' 
                : 'hover:bg-muted'
            }`}
          >
            Home Team ({homeStats.length})
          </button>
          <button
            onClick={() => setSelectedTeam('away')}
            className={`flex-1 px-6 py-3 font-semibold transition-colors ${
              selectedTeam === 'away' 
                ? 'bg-primary text-primary-foreground' 
                : 'hover:bg-muted'
            }`}
          >
            Away Team ({awayStats.length})
          </button>
        </div>
        
        {/* Stats Table */}
        <div className="flex-1 overflow-auto p-6">
          {loading ? (
            <div className="text-center py-12">Loading stats...</div>
          ) : (
            <table className="w-full">
              <thead className="sticky top-0 bg-background border-b border-border">
                <tr className="text-left text-sm text-muted-foreground">
                  <th className="pb-3 font-semibold">PLAYER</th>
                  <th className="pb-3 font-semibold text-center">MIN</th>
                  <th className="pb-3 font-semibold text-center">PTS</th>
                  <th className="pb-3 font-semibold text-center">REB</th>
                  <th className="pb-3 font-semibold text-center">AST</th>
                  <th className="pb-3 font-semibold text-center">FG</th>
                  <th className="pb-3 font-semibold text-center">3PT</th>
                  <th className="pb-3 font-semibold text-center">FT</th>
                  <th className="pb-3 font-semibold text-center">+/-</th>
                </tr>
              </thead>
              <tbody>
                {displayStats.map(stat => (
                  <tr key={stat.stat_id} className="border-b border-border hover:bg-muted/50">
                    <td className="py-3">
                      <div className="flex items-center gap-2">
                        {stat.starter && <span className="text-xs text-primary">★</span>}
                        <div>
                          <div className="font-semibold">{stat.player_name}</div>
                          <div className="text-xs text-muted-foreground">{stat.position}</div>
                        </div>
                      </div>
                    </td>
                    <td className="py-3 text-center">{stat.minutes_played?.toFixed(0) || '-'}</td>
                    <td className="py-3 text-center font-semibold">{stat.points || 0}</td>
                    <td className="py-3 text-center">{stat.rebounds || 0}</td>
                    <td className="py-3 text-center">{stat.assists || 0}</td>
                    <td className="py-3 text-center text-sm">
                      {stat.field_goals_made || 0}/{stat.field_goals_attempted || 0}
                    </td>
                    <td className="py-3 text-center text-sm">
                      {stat.three_pointers_made || 0}/{stat.three_pointers_attempted || 0}
                    </td>
                    <td className="py-3 text-center text-sm">
                      {stat.free_throws_made || 0}/{stat.free_throws_attempted || 0}
                    </td>
                    <td className={`py-3 text-center font-semibold ${
                      (stat.plus_minus || 0) > 0 ? 'text-green-500' : 
                      (stat.plus_minus || 0) < 0 ? 'text-red-500' : ''
                    }`}>
                      {stat.plus_minus > 0 ? '+' : ''}{stat.plus_minus || 0}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>
    </div>
  );
}
```

---

## 🚀 ML Features Endpoint

### Backend Implementation
```go
// internal/api/rest/ml_handlers.go

// GET /api/v1/{sport}/ml/features?season=2024-25&player_id=123
func (s *Server) HandleGetMLFeatures(w http.ResponseWriter, r *http.Request) {
	sport := chi.URLParam(r, "sport")
	seasonID := r.URL.Query().Get("season")
	playerID := r.URL.Query().Get("player_id")
	
	features, err := s.mlService.GetPlayerFeatures(r.Context(), sport, seasonID, playerID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	respondJSON(w, http.StatusOK, features)
}

// internal/service/ml_service.go
type MLService struct {
	db *store.Database
}

type PlayerMLFeatures struct {
	PlayerID          int                    `json:"player_id"`
	SeasonID          string                 `json:"season_id"`
	
	// Aggregate Stats
	GamesPlayed       int                    `json:"games_played"`
	PointsPerGame     float64                `json:"points_per_game"`
	ReboundsPerGame   float64                `json:"rebounds_per_game"`
	AssistsPerGame    float64                `json:"assists_per_game"`
	
	// Shooting Efficiency
	FieldGoalPct      float64                `json:"field_goal_pct"`
	ThreePointPct     float64                `json:"three_point_pct"`
	FreeThrowPct      float64                `json:"free_throw_pct"`
	TrueShootingPct   float64                `json:"true_shooting_pct"`
	EffectiveFGPct    float64                `json:"effective_fg_pct"`
	
	// Advanced Metrics
	UsageRate         float64                `json:"usage_rate"`
	PER               float64                `json:"player_efficiency_rating"`
	
	// Trends (last 5 games)
	RecentPointsTrend []float64              `json:"recent_points_trend"`
	RecentMinutesTrend []float64             `json:"recent_minutes_trend"`
	
	// Matchup Data
	VsPosition        map[string]float64     `json:"vs_position"` // PPG vs each position
	HomeAwayS splits  map[string]float64     `json:"home_away_splits"`
	
	// Raw Game Log (for custom feature engineering)
	GameLog           []PlayerGameStats      `json:"game_log"`
}

func (s *MLService) GetPlayerFeatures(ctx context.Context, sport, seasonID, playerID string) (*PlayerMLFeatures, error) {
	// Fetch all game stats for player in season
	stats, err := s.db.GetPlayerGameStats(ctx, sport, playerID, seasonID)
	if err != nil {
		return nil, err
	}
	
	// Calculate aggregates and trends
	features := &PlayerMLFeatures{
		PlayerID: playerID,
		SeasonID: seasonID,
		GameLog:  stats,
	}
	
	// Calculate per-game averages
	features.GamesPlayed = len(stats)
	totalPoints := 0
	for _, stat := range stats {
		totalPoints += stat.Points
	}
	features.PointsPerGame = float64(totalPoints) / float64(len(stats))
	
	// ... calculate other features
	
	return features, nil
}
```

---

## 📚 Research Summary

Based on industry best practices research (PostgreSQL sports analytics, NBA data warehousing, temporal data management):

### Key Findings

1. **Temporal Tracking is Critical**
   - NBA teams make ~100+ roster moves per season
   - Players can be on multiple teams in one season
   - Historical accuracy requires `start_date` and `end_date` tracking

2. **Partitioning Provides Massive Performance Gains**
   - Sports data grows linearly with time
   - Queries typically focus on recent seasons
   - Partitioning by season reduces query time by 10-100x

3. **Materialized Views for Common Aggregations**
   - Season averages queried thousands of times per day
   - Calculating on-the-fly is wasteful
   - Refresh strategy: nightly or after game days

4. **JSONB vs Normalized Tables**
   - **Use JSONB for**: Sport-specific stats that vary widely (e.g., football positions)
   - **Use Normalized Tables for**: Common stats queried frequently (points, rebounds)
   - **Hybrid Approach**: Core stats in columns, extended stats in JSONB

5. **Index Strategy Matters**
   - Composite indexes for multi-column filters
   - Partial indexes for common WHERE clauses
   - GIN indexes for JSONB and full-text search
   - Monitor index usage and remove unused indexes

6. **Data Validation at Database Level**
   - Check constraints prevent invalid data (e.g., negative points)
   - Foreign keys ensure referential integrity
   - Triggers for calculated fields ensure consistency

### References
- [PostgreSQL Performance Optimization](https://wiki.postgresql.org/wiki/Performance_Optimization)
- [NBA Azure AI Infrastructure](https://www.microsoft.com/en/customers/story/nba-azure-kubernetes-service)
- [Sports Database Design Best Practices](https://stackoverflow.com/questions/15212773/sql-database-sports-league-statistics-rosters)
- [Time-Series Data Management](https://www.timescale.com/blog/time-series-data/)

---

## 📋 Implementation Phases

### Phase 1: ESPN API Fix & Core Stability (Week 1) ✅
- [x] Fix ESPN API 403 issue (curl workaround)
- [x] Verify backfill system works end-to-end
- [x] Test player stats ingestion
- [x] Validate database schema

### Phase 2: Database Schema Improvements & Multi-Sport Foundation (Week 2)
- [ ] **Database Improvements** (Priority: High)
  - [ ] Add `player_team_history` table for temporal tracking
  - [ ] Add check constraints for data validation
  - [ ] Create database triggers for auto-calculated fields
  - [ ] Add GIN index for full-text player search
  - [ ] Create `player_season_averages` materialized view
  - [ ] Add composite and partial indexes
  - [ ] Write migration scripts for all improvements
- [ ] **Multi-Sport Architecture**
  - [ ] Create `internal/sports` package with Sport interface
  - [ ] Implement `basketball_nba` sport module
  - [ ] Create sport registry
  - [ ] Refactor services to be sport-aware
  - [ ] Update API endpoints with `/{sport}/` prefix
  - [ ] Add sport selector to frontend

### Phase 3: Enhanced Player Features (Week 3)
- [ ] Implement player search API
- [ ] Create PlayerCard component
- [ ] Build player search page
- [ ] Add player profile page with season stats
- [ ] Implement player comparison tool
- [ ] Add player trends visualization

### Phase 4: Team Features (Week 4)
- [ ] Implement team roster API
- [ ] Create team schedule API
- [ ] Build team page with roster
- [ ] Add team stats aggregation
- [ ] Implement team comparison

### Phase 5: ML Features & Analytics (Week 5)
- [ ] Implement ML features endpoint
- [ ] Create feature calculation service
- [ ] Add advanced metrics (PER, usage rate, etc.)
- [ ] Build analytics dashboard
- [ ] Add trend visualization
- [ ] Create export functionality for ML training

### Phase 6: UI Polish & Integration (Week 6)
- [ ] Redesign main Minerva page
- [ ] Improve box score modal
- [ ] Add loading skeletons
- [ ] Implement error boundaries
- [ ] Add toast notifications
- [ ] Mobile responsive design
- [ ] Dark mode refinements

### Phase 7: Testing & Documentation (Week 7)
- [ ] Write unit tests for sport modules
- [ ] Integration tests for API endpoints
- [ ] Frontend component tests
- [ ] Load testing for backfill system
- [ ] API documentation
- [ ] User guide

### Phase 8: Performance Optimization & Partitioning (Week 8)
- [ ] **Table Partitioning** (When data volume justifies it)
  - [ ] Partition `games` by season_id
  - [ ] Partition `player_game_stats` by season_id
  - [ ] Partition `team_game_stats` by season_id
  - [ ] Create partition maintenance scripts
  - [ ] Test query performance improvements
- [ ] **Query Optimization**
  - [ ] Analyze slow query log
  - [ ] Add missing indexes based on query patterns
  - [ ] Optimize materialized view refresh
  - [ ] Implement connection pooling tuning
- [ ] **Monitoring & Alerting**
  - [ ] Set up table size monitoring
  - [ ] Monitor index usage
  - [ ] Track query performance metrics
  - [ ] Alert on slow queries

### Phase 9: Future Sports Expansion (Week 9+)
- [ ] Add NFL support
- [ ] Add MLB support
- [ ] Add NHL support
- [ ] Add soccer leagues

---

## 🎯 Success Metrics

### Technical Metrics
- **API Response Time**: p95 < 200ms for game queries
- **Backfill Speed**: Complete season in < 5 minutes
- **WebSocket Latency**: Live updates within 2 seconds
- **Database Query Performance**: Complex stats queries < 100ms
- **Uptime**: 99.9% availability

### User Experience Metrics
- **Page Load Time**: < 2 seconds
- **Search Response**: < 500ms
- **UI Responsiveness**: 60fps animations
- **Mobile Performance**: Lighthouse score > 90

### Data Quality Metrics
- **Data Accuracy**: 99.9% match with ESPN
- **Missing Data**: < 1% of expected records
- **Reconciliation Success**: > 95% automatic matching

---

## 🔒 Best Practices

### Code Quality
1. **Type Safety**: Use TypeScript/Go type systems fully
2. **Error Handling**: Comprehensive error boundaries
3. **Logging**: Structured logging with context
4. **Testing**: 80%+ code coverage
5. **Documentation**: Inline comments for complex logic

### Performance
1. **Database Indexes**: Strategic indexes on query patterns
2. **Caching**: Redis for frequently accessed data
3. **Pagination**: All list endpoints paginated
4. **Lazy Loading**: Frontend components load on demand
5. **Image Optimization**: WebP format, lazy loading

### Security
1. **Input Validation**: All user inputs sanitized
2. **SQL Injection**: Use parameterized queries
3. **Rate Limiting**: API endpoints rate limited
4. **CORS**: Proper CORS configuration
5. **Authentication**: JWT tokens for admin features

### Scalability
1. **Horizontal Scaling**: Stateless services
2. **Database Sharding**: Ready for sport-based sharding
3. **CDN**: Static assets on CDN
4. **Connection Pooling**: Efficient database connections
5. **Queue System**: Backfill jobs queued properly

---

## 📚 References

- [ESPN API Documentation](https://gist.github.com/akeaswaran/b48b02f1c94f873c6655e7129910fc3b)
- [NBA Stats Glossary](https://www.nba.com/stats/help/glossary)
- [PostgreSQL Performance Tuning](https://wiki.postgresql.org/wiki/Performance_Optimization)
- [React Performance Best Practices](https://react.dev/learn/render-and-commit)
- [Go Project Layout](https://github.com/golang-standards/project-layout)

---

**Last Updated**: November 14, 2025  
**Next Review**: Weekly during active development

