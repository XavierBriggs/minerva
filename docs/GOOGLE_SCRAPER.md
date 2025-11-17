# Google Sports Scraper

Low-latency live game data ingestion from Google Sports.

## Overview

The Google Sports scraper provides real-time NBA game data with minimal latency (< 5 seconds from actual game events). It complements the ESPN API by providing faster updates during live games, while ESPN provides authoritative historical data and detailed statistics.

## Architecture

```
┌─────────────────┐
│  Google Sports  │
│   (Web Page)    │
└────────┬────────┘
         │ chromedp (headless Chrome)
         │
    ┌────▼────────────────┐
    │  Google Scraper     │
    │  - client.go        │ ← Rate limiting (2s between requests)
    │  - parser.go        │ ← HTML extraction
    │  - ingester.go      │ ← Orchestration
    └────────┬────────────┘
             │
        ┌────▼─────────┐
        │ Redis Cache  │ ← 5 second TTL
        │ (Live Data)  │
        └──────────────┘
```

## Components

### Client (`client.go`)

- **Headless Browser**: Uses `chromedp` to render JavaScript-heavy Google pages
- **Rate Limiting**: Enforces 2-second minimum between requests
- **User Agent**: Mimics modern Chrome browser
- **Resource Management**: Proper context and browser cleanup

Key methods:
- `NewClient()` - Initialize headless Chrome instance
- `FetchLiveGames(ctx)` - Fetch current NBA games
- `FetchGameDetails(ctx, homeTeam, awayTeam)` - Fetch specific matchup
- `Close()` - Release browser resources

### Parser (`parser.go`)

Extracts structured data from Google's HTML:

```go
type LiveGame struct {
    HomeTeam      string
    AwayTeam      string
    HomeScore     int
    AwayScore     int
    GameStatus    string  // "Live", "Final", "Q3 2:30"
    Period        int     // 1-4 (5 for OT)
    TimeRemaining string  // "2:30"
    IsLive        bool
}
```

**Parsing Strategies**:
1. **Primary**: Sports card widgets (`div.imso_mh__lv-m-stl-cont`)
2. **Fallback**: Generic sports divs with regex patterns

**Game Clock Parsing**:
- Recognizes: Q1-Q4, 1st-4th, OT, Overtime
- Extracts time remaining in MM:SS format
- Detects halftime status

### Ingester (`ingester.go`)

Orchestrates scraping operations:

- **Cache Integration**: 5-second TTL for live data
- **Continuous Polling**: `PollLiveGames()` for real-time updates
- **Error Handling**: Graceful degradation on scraping failures

## Usage

### Basic Scraping

```go
import "github.com/fortuna/minerva/internal/ingest/google"

// Create client
client, err := google.NewClient()
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Fetch live games
ctx := context.Background()
htmlContent, err := client.FetchLiveGames(ctx)
if err != nil {
    log.Fatal(err)
}

// Parse HTML
doc, err := google.ParseHTML(htmlContent)
if err != nil {
    log.Fatal(err)
}

// Extract games
games, err := google.ParseLiveGames(doc)
if err != nil {
    log.Fatal(err)
}

// Process games
for _, game := range games {
    fmt.Printf("%s vs %s: %d-%d (%s)\n",
        game.AwayTeam, game.HomeTeam,
        game.AwayScore, game.HomeScore,
        game.GameStatus)
}
```

### Continuous Polling

```go
ingester, err := google.NewIngester(cache, db)
if err != nil {
    log.Fatal(err)
}
defer ingester.Close()

// Poll every 10 seconds
ctx := context.Background()
ingester.PollLiveGames(ctx, "2024-25", 10*time.Second, func(games []google.LiveGame) {
    log.Printf("Found %d live games", len(games))
    // Process games...
})
```

### Testing

Run the test utility:

```bash
cd minerva-go
go run ./scripts/test-google-scraper.go
```

Expected output:
```
Testing Google Sports Scraper
===============================

1. Fetching live NBA games...
✓ Retrieved HTML content (235847 bytes)
✓ Found 3 games

Game 1:
  Celtics vs Lakers
  Score: 105 - 98
  Status: Q4 2:30
  Period: 4
  Time: 2:30
  Live: true

...
```

## Rate Limiting

**Default**: 2 seconds between requests

**Why Rate Limiting?**
- Prevents Google from blocking our scraper
- Reduces server load
- Still provides near-real-time updates (< 5s latency)

**Adjust if needed**:
```go
client.interval = 5 * time.Second  // More conservative
```

## Error Handling

Common errors and solutions:

| Error | Cause | Solution |
|-------|-------|----------|
| `chromedp error: timeout` | Page took too long to load | Increase timeout (30s default) |
| `empty HTML content` | Google blocked request | Check User-Agent, add delays |
| `no game found` | Incorrect team names | Use `GetTeamAbbreviation()` |
| `failed to parse HTML` | Google changed page structure | Update CSS selectors |

## Limitations

1. **HTML Structure Changes**: Google may update their page layout without notice
2. **Team Name Variations**: "Los Angeles Lakers" vs "Lakers" vs "LAL"
3. **Rate Limits**: Too many requests = temporary IP block
4. **JavaScript Requirement**: Needs headless browser (heavier than simple HTTP)
5. **No Detailed Stats**: Only basic scores, not player-level data

## Integration with Reconciliation Engine

The Google scraper feeds into the reconciliation engine:

```
Google Scraper (Fast, Live) ──┐
                              ├──> Reconciliation Engine
ESPN API (Authoritative)  ────┘
```

Reconciliation logic:
- **Live games**: Prefer Google for score/time (fresher)
- **Final games**: Prefer ESPN for stats (authoritative)
- **Conflicts**: ESPN overrides Google after game completion

## Performance

**Benchmarks** (on MacBook Pro M1):
- Single game fetch: ~3-5 seconds
- HTML parsing: ~50-100ms
- Total latency: 3-6 seconds from Google update

**Resource Usage**:
- Memory: ~100-150 MB per Chrome instance
- CPU: Negligible between requests
- Network: ~200-300 KB per fetch

## Maintenance

**Monitor for**:
- CSS selector changes (Google updates)
- Increased error rates
- Slow response times

**Update CSS selectors in**:
- `parser.go` → `parseSportsCard()`
- `parser.go` → `parseSportsDiv()`

## Future Enhancements

- [ ] Player-level live stats extraction
- [ ] Play-by-play event scraping
- [ ] Multi-sport support (NFL, MLB, NHL)
- [ ] Proxy rotation for higher request volumes
- [ ] Screenshot capture for debugging
- [ ] Machine learning for adaptive selector detection


