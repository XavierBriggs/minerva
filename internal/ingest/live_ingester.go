package ingest

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/fortuna/minerva/internal/cache"
	"github.com/fortuna/minerva/internal/ingest/espn"
	"github.com/fortuna/minerva/internal/ingest/google"
	"github.com/fortuna/minerva/internal/publisher"
	"github.com/fortuna/minerva/internal/reconciliation"
	"github.com/fortuna/minerva/internal/store"
	"github.com/fortuna/minerva/internal/store/repository"
)

// LiveIngester handles live game data ingestion with proper fallback logic
// Primary: Google (low latency)
// Fallback: ESPN (authoritative, reliable)
type LiveIngester struct {
	googleIngester *google.Ingester
	espnIngester   *espn.Ingester
	reconciler     *reconciliation.Engine
	matcher        *reconciliation.Matcher
	cache          *cache.RedisCache
	publisher      *publisher.RedisStreamPublisher
	db             *store.Database
}

// NewLiveIngester creates a new live game ingester with fallback support
func NewLiveIngester(cache *cache.RedisCache, publisher *publisher.RedisStreamPublisher, db *store.Database) (*LiveIngester, error) {
	// Initialize Google ingester (primary)
	googleIngester, err := google.NewIngester(cache, db)
	if err != nil {
		log.Printf("Warning: Failed to initialize Google ingester: %v", err)
		// Continue without Google - ESPN will be the only source
	}

	// Initialize ESPN ingester (fallback)
	espnIngester := espn.NewIngester(db)

	// Initialize reconciliation engine
	reconciler := reconciliation.NewEngine(reconciliation.SmartMerge)

	// Load teams for matching
	teamRepo := repository.NewTeamRepository(db)
	teams, err := teamRepo.GetAll(context.Background())
	if err != nil {
		return nil, err
	}
	matcher := reconciliation.NewMatcher(teams)

	return &LiveIngester{
		googleIngester: googleIngester,
		espnIngester:   espnIngester,
		reconciler:     reconciler,
		matcher:        matcher,
		cache:          cache,
		publisher:      publisher,
		db:             db,
	}, nil
}

// Close releases resources
func (li *LiveIngester) Close() {
	if li.googleIngester != nil {
		li.googleIngester.Close()
	}
}

// IngestLiveGames fetches and reconciles live games from both sources
// Google is primary (fast), ESPN is fallback (reliable)
func (li *LiveIngester) IngestLiveGames(ctx context.Context, seasonID string) ([]*store.Game, error) {
	log.Println("Ingesting live games (Google primary, ESPN fallback)...")

	// Convert seasonID string to int for database operations
	seasonIDInt, err := li.lookupSeasonID(ctx, seasonID)
	if err != nil {
		return nil, fmt.Errorf("lookup season ID: %w", err)
	}

	var googleGames []google.LiveGame
	var espnGames []*store.Game
	var googleErr, espnErr error

	// Try Google first (primary source for live games)
	if li.googleIngester != nil {
		googleGames, googleErr = li.googleIngester.IngestLiveGames(ctx, seasonID)
		if googleErr != nil {
			log.Printf("⚠️  Google ingestion failed: %v (falling back to ESPN)", googleErr)
		} else {
			log.Printf("✓ Google: Retrieved %d games", len(googleGames))
		}
	} else {
		log.Println("⚠️  Google ingester unavailable (falling back to ESPN)")
	}

	// Always fetch from ESPN (fallback + authoritative data)
	espnErr = li.espnIngester.IngestTodaysGames(ctx, seasonIDInt)
	if espnErr != nil {
		log.Printf("⚠️  ESPN ingestion failed: %v", espnErr)
	} else {
		// Fetch today's games from database (all statuses)
		gameRepo := repository.NewGameRepository(li.db)
		today := time.Now().Truncate(24 * time.Hour)
		espnGames, _ = gameRepo.GetByDate(ctx, today)
		log.Printf("✓ ESPN: Ingested %d games for today", len(espnGames))
	}

	// Handle complete failure (both sources failed)
	if (googleErr != nil || len(googleGames) == 0) && (espnErr != nil || len(espnGames) == 0) {
		log.Println("❌ Both Google and ESPN failed - no live game data available")
		return []*store.Game{}, nil
	}

	// If only ESPN available, use it directly (fallback)
	if (googleErr != nil || len(googleGames) == 0) && len(espnGames) > 0 {
		log.Println("→ Using ESPN data only (Google unavailable)")
		return espnGames, nil
	}

	// If only Google available (rare), use it
	if len(googleGames) > 0 && len(espnGames) == 0 {
		log.Println("→ Using Google data only (ESPN unavailable - unusual)")
		// Convert Google games to store.Game format
		var games []*store.Game
		for _, g := range googleGames {
			games = append(games, google.ConvertToStoreGame(g, seasonIDInt))
		}
		return games, nil
	}

	// Both sources available - reconcile
	log.Println("→ Reconciling data from both sources...")
	reconciledGames, err := li.matcher.MatchAndReconcileAll(espnGames, googleGames, li.reconciler)
	if err != nil {
		log.Printf("⚠️  Reconciliation error: %v (falling back to ESPN)", err)
		return espnGames, nil
	}

	// Log metrics
	metrics := li.reconciler.GetMetrics()
	log.Printf("✓ Reconciliation complete: Total=%d, Conflicts=%d, Google=%d, ESPN=%d",
		metrics.TotalReconciliations,
		metrics.Conflicts,
		metrics.GooglePreferred,
		metrics.ESPNPreferred)

	return reconciledGames, nil
}

// PollLiveGames continuously polls for live game updates
func (li *LiveIngester) PollLiveGames(ctx context.Context, seasonID string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Starting live game polling (interval: %v)", interval)
	log.Println("Source priority: Google (primary) → ESPN (fallback)")

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping live game polling")
			return
		case <-ticker.C:
			games, err := li.IngestLiveGames(ctx, seasonID)
			if err != nil {
				log.Printf("Polling error: %v", err)
				continue
			}

			// Publish to Redis streams
			for _, game := range games {
				if game.Status == "in_progress" {
					if err := li.publisher.PublishLiveGameUpdate(ctx, game); err != nil {
						log.Printf("Error publishing game %s: %v", game.GameID, err)
					}
				}
			}

			log.Printf("✓ Polling cycle complete: %d games", len(games))
		}
	}
}

// lookupSeasonID queries the database to get season_id (INT) from season_year (STRING)
func (li *LiveIngester) lookupSeasonID(ctx context.Context, seasonYear string) (int, error) {
	query := `SELECT season_id FROM seasons WHERE season_year = $1 AND sport = 'basketball_nba' LIMIT 1`
	
	var seasonID int
	err := li.db.DB().QueryRowContext(ctx, query, seasonYear).Scan(&seasonID)
	if err != nil {
		return 0, fmt.Errorf("season '%s' not found in database: %w", seasonYear, err)
	}
	
	return seasonID, nil
}

