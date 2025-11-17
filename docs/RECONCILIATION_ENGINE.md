# Reconciliation Engine

Intelligent data merging from multiple sources (ESPN + Google Sports).

## Overview

The reconciliation engine is the heart of Minerva's data accuracy strategy. It combines:
- **Google Sports**: Low-latency live updates (3-6 second delay)
- **ESPN API**: Authoritative data with detailed statistics

## Architecture

```
┌──────────────┐         ┌─────────────────┐
│ Google       │         │ ESPN API        │
│ (Live/Fast)  │         │ (Auth/Detailed) │
└──────┬───────┘         └────────┬────────┘
       │                          │
       │   ┌──────────────────────┘
       │   │
    ┌──▼───▼──────────┐
    │  Reconciliation │
    │     Engine      │
    │  ┌──────────┐   │
    │  │ Matcher  │   │ ← Team name matching
    │  └──────────┘   │
    │  ┌──────────┐   │
    │  │ Strategy │   │ ← Smart merge logic
    │  └──────────┘   │
    │  ┌──────────┐   │
    │  │ Metrics  │   │ ← Conflict tracking
    │  └──────────┘   │
    └─────────┬────────┘
              │
         ┌────▼─────┐
         │ Unified  │
         │   Data   │
         └──────────┘
```

## Reconciliation Strategies

### 1. Smart Merge (Default)

Context-aware logic based on game state:

| Game State | Data Source | Reason |
|------------|-------------|--------|
| Pre-game | ESPN | Complete schedule data |
| Live | Google scores + ESPN structure | Freshest scores, authoritative IDs |
| Final | ESPN | Complete box scores |
| Conflict | ESPN | More reliable for official stats |

### 2. Prefer Latest

Always use Google (most recent data).

### 3. Prefer Authoritative

Always use ESPN (most accurate data).

## Smart Merge Logic

```go
engine := reconciliation.NewEngine(reconciliation.SmartMerge)
merged, err := engine.ReconcileGame(espnGame, googleGame)
```

**Decision Tree:**

```
                   ┌─────────────┐
                   │ Both sources│
                   │  available? │
                   └──────┬──────┘
                          │
              ┌───────────┴───────────┐
              │                       │
           ┌──▼──┐                ┌───▼───┐
           │ Yes │                │  No   │
           └──┬──┘                └───┬───┘
              │                       │
      ┌───────▼───────┐              │
      │ Detect game   │              │
      │    state      │              │
      └───────┬───────┘              │
              │                      │
    ┌─────────┼─────────┐           │
    │         │         │           │
┌───▼───┐ ┌──▼──┐ ┌────▼─────┐    │
│ Pre-  │ │Live │ │  Final   │    │
│ game  │ │     │ │          │    │
└───┬───┘ └──┬──┘ └────┬─────┘    │
    │        │         │           │
    │   ┌────▼─────┐   │           │
    │   │  Google  │   │           │
    │   │  scores  │   │           │
    │   │   +      │   │           │
    │   │   ESPN   │   │           │
    │   │structure │   │           │
    │   └────┬─────┘   │           │
    │        │         │           │
    └────────┼─────────┘           │
             │                     │
        ┌────▼─────┐        ┌──────▼──────┐
        │   ESPN   │        │Use available│
        │ (Final)  │        │   source    │
        └──────────┘        └─────────────┘
```

## Conflict Detection

The engine automatically detects conflicts:

### Score Discrepancies

```go
// Large score difference (> 20 points)
ESPN:  Lakers 105 - Celtics 98
Google: Lakers 125 - Celtics 98
→ CONFLICT (20+ point diff)
```

### Status Conflicts

```go
// Mismatched game states
ESPN:  Final
Google: Live (Q4 2:30)
→ CONFLICT (status mismatch)
```

### Resolution

All conflicts default to ESPN (authoritative).

## Team Matching

The `Matcher` component handles team name variations:

```go
matcher := reconciliation.NewMatcher(teams)
googleGame := matcher.FindMatchingGoogleGame(espnGame, googleGames)
```

**Supported Variations:**

| ESPN | Google Variants |
|------|----------------|
| LAL | Lakers, Los Angeles Lakers, LA Lakers |
| GSW | Warriors, Golden State Warriors, GS Warriors |
| BKN | Nets, Brooklyn Nets |
| LAC | Clippers, Los Angeles Clippers, LA Clippers |

Add more in `matcher.go` → `matchTeams()`.

## Usage

### Basic Reconciliation

```go
import "github.com/fortuna/minerva/internal/reconciliation"

// Create engine
engine := reconciliation.NewEngine(reconciliation.SmartMerge)

// Reconcile single game
merged, err := engine.ReconcileGame(espnGame, googleGame)
if err != nil {
    log.Fatal(err)
}

// Check metrics
metrics := engine.GetMetrics()
log.Printf("Conflicts: %d, Google used: %d, ESPN used: %d",
    metrics.Conflicts,
    metrics.GooglePreferred,
    metrics.ESPNPreferred)
```

### Batch Reconciliation with Matching

```go
// Load teams for matching
teams, err := teamRepo.GetAll(ctx)
if err != nil {
    log.Fatal(err)
}

// Create matcher
matcher := reconciliation.NewMatcher(teams)

// Match and reconcile all games
reconciledGames, err := matcher.MatchAndReconcileAll(
    espnGames,
    googleGames,
    engine,
)
```

### Custom Strategy

```go
// Always prefer Google (low latency)
engine := reconciliation.NewEngine(reconciliation.PreferLatest)

// Always prefer ESPN (accuracy)
engine := reconciliation.NewEngine(reconciliation.PreferAuthoritative)
```

## Metrics

Track reconciliation performance:

```go
type Metrics struct {
    TotalReconciliations int       // Total games processed
    Conflicts            int       // Conflicts detected
    GooglePreferred      int       // Times Google was chosen
    ESPNPreferred        int       // Times ESPN was chosen
    LastReconciliation   time.Time // Most recent reconciliation
}

metrics := engine.GetMetrics()
log.Printf("Success rate: %.2f%%",
    float64(metrics.TotalReconciliations - metrics.Conflicts) /
    float64(metrics.TotalReconciliations) * 100)
```

## Performance

**Benchmarks** (1000 game reconciliations):

| Operation | Time | Throughput |
|-----------|------|------------|
| Single game reconciliation | ~50μs | 20,000 games/sec |
| Team matching | ~10μs | 100,000 matches/sec |
| Conflict detection | ~20μs | 50,000 checks/sec |
| Batch (100 games) | ~5ms | 20,000 games/sec |

Memory usage: Negligible (< 1 MB for 1000 games)

## Error Handling

```go
merged, err := engine.ReconcileGame(espnGame, googleGame)
if err != nil {
    switch {
    case err.Error() == "both sources are nil":
        // No data available
        log.Println("No data sources available")
        
    default:
        // Unknown error - use fallback
        log.Printf("Error: %v, using ESPN fallback", err)
        merged = espnGame
    }
}
```

## Integration Points

### 1. Scheduler

```go
// In scheduler polling loop
func (s *Scheduler) pollLiveGames() {
    espnGames := s.fetchESPN()
    googleGames := s.fetchGoogle()
    
    // Reconcile
    merged := s.reconcile(espnGames, googleGames)
    
    // Publish to Redis
    for _, game := range merged {
        s.publishToRedis(game)
    }
}
```

### 2. WebSocket Broadcaster

```go
// Only broadcast reconciled data
reconciled, _ := engine.ReconcileGame(espn, google)
broadcaster.BroadcastGameUpdate(reconciled)
```

### 3. REST API

```go
// API returns reconciled live games
func (h *Handler) GetLiveGames(w http.ResponseWriter, r *http.Request) {
    espnGames := h.fetchFromCache("espn:live")
    googleGames := h.fetchFromCache("google:live")
    
    merged := h.engine.ReconcileGames(espnGames, googleGames)
    json.NewEncoder(w).Encode(merged)
}
```

## Testing

Run reconciliation tests:

```bash
go test ./internal/reconciliation/... -v
```

Manual testing:

```bash
go run ./scripts/test-reconciliation.go
```

## Troubleshooting

### High Conflict Rate

**Problem**: > 10% conflict rate

**Solutions**:
1. Check if Google scraper is working correctly
2. Verify team name matching logic
3. Adjust conflict detection thresholds

### Team Matching Failures

**Problem**: Games not being matched

**Solutions**:
1. Add team name variants to `matcher.go`
2. Check team abbreviations in database
3. Enable debug logging

### Performance Issues

**Problem**: Slow reconciliation

**Solutions**:
1. Use batch reconciliation for multiple games
2. Cache team lookups
3. Profile with `go test -bench`

## Future Enhancements

- [ ] Machine learning for conflict resolution
- [ ] Historical accuracy tracking
- [ ] Automatic team name learning
- [ ] Multi-source reconciliation (3+ sources)
- [ ] Weighted confidence scores
- [ ] Real-time metrics dashboard


