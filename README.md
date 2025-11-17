# Minerva v2 - Sports Analytics Service

**Professional sports data ingestion and analytics system for Fortuna**

Minerva is a Go-based microservice that provides comprehensive NBA statistics through dual data pipelines:
- **Live Pipeline**: Ultra-low latency live game data via Google Sports scraping
- **Historical Pipeline**: Accurate, complete box scores and player stats via ESPN API

## Features

- **Dual Data Pipelines**: Google scraping for live data, ESPN API for historical accuracy
- **PostgreSQL Storage**: All data stored in Atlas database (normalized 2NF/3NF schema)
- **Redis Integration**: Caching layer and real-time streams for live updates
- **REST API**: Comprehensive endpoints for games, players, teams, and stats
- **WebSocket Server**: Real-time live game updates for web clients
- **ML-Ready**: Feature extraction endpoints for model training
- **Multi-Sport Ready**: Extensible architecture for adding NFL, MLB, etc.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    MINERVA (Go Service)                      │
│                                                              │
│  Ingestion Layer:                                           │
│  ├─ ESPN Client (historical + backup live)                 │
│  ├─ Google Sports Scraper (primary live data)              │
│  └─ Reconciliation Engine (data accuracy)                  │
│                                                              │
│  Service Layer:                                             │
│  ├─ Historical Stats Service                               │
│  ├─ Live Game Service                                       │
│  ├─ Player/Team Query Service                              │
│  └─ Analytics Service (advanced metrics)                   │
│                                                              │
│  API Layer:                                                 │
│  ├─ REST API (port 8080)                                   │
│  └─ WebSocket (port 8081)                                  │
└─────────────────────────────────────────────────────────────┘
```

## Quick Start

### Prerequisites

- Go 1.21+
- PostgreSQL 16
- Redis

### Local Development

```bash
# Install dependencies
make deps

# Run database migrations
export ATLAS_DSN="postgres://fortuna:fortuna_pw@localhost:5434/atlas?sslmode=disable"
make migrate-up

# Run service
make run
```

### Docker

```bash
# Build image
make docker-build

# Run container
make docker-run
```

## Configuration

Environment variables:

```bash
ATLAS_DSN=postgres://fortuna:fortuna_pw@atlas:5432/atlas
REDIS_URL=redis://redis:6379
REST_PORT=8080
WS_PORT=8081
ESPN_API_BASE=https://site.api.espn.com
LOG_LEVEL=info
```

## API Endpoints

### Games
```
GET  /api/v1/games/today           - Today's NBA games
GET  /api/v1/games/live            - Currently live games
GET  /api/v1/games/{game_id}       - Game details
GET  /api/v1/games/{game_id}/boxscore - Full box score
```

### Players
```
GET  /api/v1/players/{player_id}         - Player profile
GET  /api/v1/players/{player_id}/stats   - Season stats
GET  /api/v1/players/{player_id}/recent  - Last N games
GET  /api/v1/players/search?q={name}     - Search players
```

### Teams
```
GET  /api/v1/teams/{team_id}          - Team info
GET  /api/v1/teams/{team_id}/roster   - Current roster
GET  /api/v1/teams/{team_id}/schedule - Season schedule
```

### WebSocket
```
WS   /ws/games/live                   - Live game updates stream
```

## Database Schema

### Atlas (PostgreSQL)

**Core Tables:**
- `seasons` - NBA season metadata
- `teams` - 30 NBA franchises
- `players` - Player profiles
- `player_seasons` - Season-by-season participation
- `games` - Every NBA game
- `player_game_stats` - Player box scores
- `team_game_stats` - Team box scores
- `odds_mappings` - Links to Alexandria odds data

## Redis Streams

**Published Streams:**
- `games.live.basketball_nba` - Live score updates (10s polling)
- `games.stats.basketball_nba` - Final box scores
- `games.schedule.basketball_nba` - Schedule updates

## Testing

```bash
# Run tests
make test

# Run with coverage
make test-coverage
```

## Integration with Fortuna

Minerva integrates with:
- **API Gateway**: Routes proxied through `/api/minerva/*`
- **WS Broadcaster**: Subscribes to Minerva's Redis streams
- **Holocron**: Links games to betting opportunities via `odds_mappings`
- **Alexandria**: Cross-references with odds data
- **Web Client**: Consumes REST API and WebSocket for live updates

## Development

```bash
# Format code
make fmt

# Lint code
make lint

# Clean build artifacts
make clean
```

## Deployment

Minerva runs as a Docker container in the Fortuna stack:

```yaml
minerva:
  image: fortuna-minerva:latest
  ports:
    - "8085:8080"  # REST API
    - "8086:8081"  # WebSocket
  depends_on:
    - atlas
    - redis
```

## License

Part of the Fortuna betting platform.
