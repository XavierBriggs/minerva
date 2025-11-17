# Minerva Documentation

**Minerva** is Fortuna's sports analytics and data ingestion system, designed for ML training and real-time insights.

---

## 📚 Documentation Index

### Core Documentation

1. **[MINERVA_IMPLEMENTATION_PLAN.md](./MINERVA_IMPLEMENTATION_PLAN.md)** - Main implementation roadmap
   - System architecture overview
   - Multi-sport abstraction layer
   - Backend and frontend implementation details
   - 9-phase implementation plan
   - Success metrics and best practices

2. **[ATLAS_DATABASE_DESIGN.md](./ATLAS_DATABASE_DESIGN.md)** - Database schema and best practices
   - Research-based design principles
   - Complete table schemas with constraints
   - Performance optimization strategies
   - Partitioning and indexing guidelines
   - Database functions and triggers
   - Security and monitoring

### Quick Start

#### Current Status (November 14, 2025)
- ✅ **Phase 1 Complete**: ESPN API integration, backfill system, player stats ingestion
- 🚧 **Phase 2 In Progress**: Database improvements and multi-sport architecture
- 📋 **Phases 3-9**: Player features, team features, ML endpoints, UI polish, testing

#### What Works Now
- Live game data ingestion from ESPN
- Historical data backfill by season or date range
- Player game statistics storage
- Real-time WebSocket updates
- Web UI with live games, historical lookup, and backfill controls

#### What's Next
- Add `player_team_history` table for temporal tracking
- Implement multi-sport abstraction layer
- Enhanced player search and profile pages
- Team rosters and schedules
- ML features endpoint

---

## 🏗️ System Architecture

```
ESPN API → Minerva Ingester → Atlas DB → REST API → Fortuna Web Client
                                    ↓
                              Redis Streams → WS Broadcaster → Live Updates
```

### Key Components

- **Atlas Database**: PostgreSQL database for sports data
- **Ingester**: Data collection from ESPN (primary) and Google Sports (fallback)
- **Scheduler**: Orchestrates live polling and historical ingestion
- **REST API**: Exposes data to web client and other services
- **WebSocket Server**: Real-time updates via Redis streams
- **Backfill System**: Historical data loading with progress tracking

---

## 🎯 Design Principles

### 1. Multi-Sport Ready
- Sport-agnostic core schema
- JSONB for sport-specific attributes
- Sport interface for easy expansion

### 2. Performance Optimized
- Strategic indexing (composite, partial, GIN)
- Materialized views for common queries
- Table partitioning for large datasets
- Connection pooling and query optimization

### 3. ML-Friendly
- Raw game logs for feature engineering
- Pre-calculated advanced metrics
- Temporal data for trend analysis
- Export-ready data formats

### 4. Production Quality
- Comprehensive error handling
- Graceful degradation
- Monitoring and alerting
- Security best practices

---

## 📊 Database Schema Overview

### Core Tables
- `seasons` - Sports seasons across all leagues
- `teams` - Team information (30+ per sport)
- `players` - Player biographical data (5,000+ per sport)
- `player_team_history` - Temporal tracking of roster changes
- `games` - Game information (1,500+ per season)
- `player_game_stats` - Per-game player statistics (300,000+ per season)
- `team_game_stats` - Per-game team statistics
- `odds_mappings` - Links to Alexandria (Mercury) odds data
- `backfill_jobs` - Historical data loading tracking

### Key Design Features
- **Temporal Tracking**: `player_team_history` tracks trades and signings
- **Partitioning**: Large tables partitioned by season for performance
- **Materialized Views**: `player_season_averages` for fast queries
- **Full-Text Search**: GIN index on player names
- **Auto-Calculated Fields**: Triggers for shooting percentages

---

## 🚀 Implementation Phases

### Phase 1: Core Stability ✅
ESPN API integration, backfill system, player stats ingestion

### Phase 2: Database & Multi-Sport (Current)
Schema improvements, sport abstraction layer, sport registry

### Phase 3: Player Features
Search, profiles, stats display, comparisons

### Phase 4: Team Features
Rosters, schedules, stats aggregation

### Phase 5: ML Features
Advanced metrics, trends, feature endpoints

### Phase 6: UI Polish
Enhanced components, mobile responsive, dark mode

### Phase 7: Testing & Docs
Unit tests, integration tests, API docs, user guide

### Phase 8: Performance
Partitioning, query optimization, monitoring

### Phase 9: Sports Expansion
NFL, MLB, NHL, soccer

---

## 🔧 Development

### Prerequisites
- Go 1.23+
- PostgreSQL 14+
- Redis 7+
- Docker & Docker Compose

### Local Setup
```bash
# Start services
cd deploy
docker-compose --profile app up -d

# Check Minerva logs
docker-compose logs -f minerva

# Access services
# - Minerva REST API: http://localhost:8085
# - API Gateway: http://localhost:8081
# - Web Client: http://localhost:3000
```

### Database Migrations
```bash
# Migrations run automatically on startup
# Located in: minerva-go/infra/atlas/migrations/
```

### Backfill Data
```bash
# Via API
curl -X POST http://localhost:8081/api/v1/minerva/backfill \
  -H "Content-Type: application/json" \
  -d '{"sport": "basketball_nba", "season_id": "2024-25"}'

# Via Web UI
# Navigate to http://localhost:3000/minerva
# Click "Data Management" → Select season
```

---

## 📈 Performance Targets

### API Response Times
- Game queries: p95 < 200ms
- Player search: p95 < 100ms
- Stats queries: p95 < 150ms

### Data Ingestion
- Live game polling: Every 10 seconds
- Backfill speed: Complete season in < 5 minutes
- WebSocket latency: Updates within 2 seconds

### Database Performance
- Complex stats queries: < 100ms
- Materialized view refresh: < 30 seconds
- Full-text player search: < 50ms

---

## 🔒 Security

### Database Access
- Read-only role for analytics/ML
- Application role for read/write
- Admin role for migrations

### API Security
- Rate limiting on all endpoints
- Input validation and sanitization
- CORS configuration
- JWT tokens for admin features (future)

---

## 📚 Additional Resources

### External References
- [ESPN API Documentation](https://gist.github.com/akeaswaran/b48b02f1c94f873c6655e7129910fc3b)
- [NBA Stats Glossary](https://www.nba.com/stats/help/glossary)
- [PostgreSQL Performance Tuning](https://wiki.postgresql.org/wiki/Performance_Optimization)
- [Go Project Layout](https://github.com/golang-standards/project-layout)

### Internal References
- [Fortuna v0 Plan](../../docs/fortuna-v0.plan.md) - Overall system architecture
- [Mercury](../../mercury/) - Odds aggregation service
- [Alexandria DB](../../docs/fortuna-v0.plan.md#alexandria-db) - Odds database
- [Holocron DB](../../docs/fortuna-v0.plan.md#holocron-db) - Bet tracking

---

## 🤝 Contributing

### Code Style
- Go: Follow standard Go conventions
- TypeScript: ESLint + Prettier
- SQL: Lowercase keywords, descriptive names

### Testing
- Unit tests for all business logic
- Integration tests for API endpoints
- Frontend component tests

### Documentation
- Update docs when changing architecture
- Add comments for complex logic
- Keep README files current

---

**Last Updated**: November 14, 2025  
**Version**: 1.0  
**Status**: Active Development

