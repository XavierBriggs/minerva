package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/fortuna/minerva/internal/cache"
	"github.com/fortuna/minerva/internal/ingest"
	"github.com/fortuna/minerva/internal/ingest/espn"
	"github.com/fortuna/minerva/internal/publisher"
	"github.com/fortuna/minerva/internal/store"
)

// Orchestrator manages scheduled tasks for data ingestion
type Orchestrator struct {
	db            *store.Database
	cache         *cache.RedisCache
	publisher     *publisher.RedisPublisher
	config        *Config
	liveIngester  *ingest.LiveIngester
	espnIngester  *espn.Ingester
	cancel        context.CancelFunc
	
	// Task coordination
	liveGamesCtx    context.Context
	liveGamesCancel context.CancelFunc
	dailyCtx        context.Context
	dailyCancel     context.CancelFunc
}

// Config holds scheduler configuration
type Config struct {
	LivePollInterval     time.Duration // Default: 10s
	DailyIngestionHour   int           // Default: 3 (3 AM)
	CurrentSeasonID      string        // e.g., "2024-25"
	EnableLivePolling    bool          // Default: true
	EnableDailyIngestion bool          // Default: true
	MaxRetries           int           // Default: 3
	RetryDelay           time.Duration // Default: 5s
}

// DefaultConfig returns default scheduler configuration
func DefaultConfig() *Config {
	return &Config{
		LivePollInterval:     10 * time.Second,
		DailyIngestionHour:   3,
		CurrentSeasonID:      "2025-26",
		EnableLivePolling:    true,
		EnableDailyIngestion: true,
		MaxRetries:           3,
		RetryDelay:           5 * time.Second,
	}
}

// NewOrchestrator creates a new scheduler orchestrator
func NewOrchestrator(db *store.Database, cache *cache.RedisCache, redisPublisher *publisher.RedisPublisher, config *Config) (*Orchestrator, error) {
	if config == nil {
		config = DefaultConfig()
	}
	
	// Create stream publisher from Redis cache client
	streamPublisher := publisher.NewRedisStreamPublisher(cache.Client())
	
	// Initialize live ingester (Google + ESPN with fallback)
	liveIngester, err := ingest.NewLiveIngester(cache, streamPublisher, db)
	if err != nil {
		return nil, err
	}
	
	// Initialize ESPN ingester for daily/historical tasks
	espnIngester := espn.NewIngester(db)
	
	return &Orchestrator{
		db:           db,
		cache:        cache,
		publisher:    redisPublisher,
		config:       config,
		liveIngester: liveIngester,
		espnIngester: espnIngester,
	}, nil
}

// Start begins all scheduled tasks
func (o *Orchestrator) Start(ctx context.Context) {
	log.Println("╔════════════════════════════════════════╗")
	log.Println("║   Minerva Scheduler Orchestrator      ║")
	log.Println("╚════════════════════════════════════════╝")
	log.Printf("Live polling: %v (interval: %v)", o.config.EnableLivePolling, o.config.LivePollInterval)
	log.Printf("Daily ingestion: %v (at %02d:00)", o.config.EnableDailyIngestion, o.config.DailyIngestionHour)
	log.Printf("Season: %s", o.config.CurrentSeasonID)
	log.Println()
	
	// Create cancellable context for the orchestrator
	ctx, cancel := context.WithCancel(ctx)
	o.cancel = cancel
	
	// Start live game polling
	if o.config.EnableLivePolling {
		o.liveGamesCtx, o.liveGamesCancel = context.WithCancel(ctx)
		go o.runLiveGamePolling(o.liveGamesCtx)
	}
	
	// Start daily ingestion scheduler
	if o.config.EnableDailyIngestion {
		o.dailyCtx, o.dailyCancel = context.WithCancel(ctx)
		go o.runDailyIngestion(o.dailyCtx)
	}
	
	// Wait for context cancellation
	<-ctx.Done()
	log.Println("Scheduler orchestrator stopping...")
}

// runLiveGamePolling polls for live game updates
func (o *Orchestrator) runLiveGamePolling(ctx context.Context) {
	log.Printf("→ Live game polling started (interval: %v)", o.config.LivePollInterval)
	log.Println("  Source priority: Google (primary) → ESPN (fallback)")
	
	ticker := time.NewTicker(o.config.LivePollInterval)
	defer ticker.Stop()
	
	consecutiveErrors := 0
	maxConsecutiveErrors := 5
	
	// Run immediately on start
	o.pollLiveGamesWithRetry(ctx, &consecutiveErrors, maxConsecutiveErrors)
	
	for {
		select {
		case <-ctx.Done():
			log.Println("→ Live game polling stopped")
			return
		case <-ticker.C:
			o.pollLiveGamesWithRetry(ctx, &consecutiveErrors, maxConsecutiveErrors)
		}
	}
}

// pollLiveGamesWithRetry polls live games with retry logic
func (o *Orchestrator) pollLiveGamesWithRetry(ctx context.Context, consecutiveErrors *int, maxConsecutiveErrors int) {
	var games []*store.Game
	var err error
	
	// Retry loop
	for attempt := 1; attempt <= o.config.MaxRetries; attempt++ {
		games, err = o.liveIngester.IngestLiveGames(ctx, o.config.CurrentSeasonID)
		
		if err == nil {
			*consecutiveErrors = 0 // Reset on success
			break
		}
		
		// Log error and retry
		log.Printf("  ⚠️  Polling attempt %d/%d failed: %v", attempt, o.config.MaxRetries, err)
		
		if attempt < o.config.MaxRetries {
			log.Printf("  Retrying in %v...", o.config.RetryDelay)
			select {
			case <-ctx.Done():
				return
			case <-time.After(o.config.RetryDelay):
				// Continue to next attempt
			}
		}
	}
	
	// All retries exhausted
	if err != nil {
		*consecutiveErrors++
		log.Printf("  ❌ All %d retry attempts failed. Consecutive errors: %d/%d",
			o.config.MaxRetries, *consecutiveErrors, maxConsecutiveErrors)
		
		// If too many consecutive errors, reduce polling frequency
		if *consecutiveErrors >= maxConsecutiveErrors {
			log.Printf("  ⚠️  High error rate detected. Slowing polling to 30s...")
			time.Sleep(20 * time.Second) // Additional delay
		}
		return
	}
	
	// Success - publish games
	liveGameCount := 0
	for _, game := range games {
		if game.Status == "in_progress" {
			liveGameCount++
			if err := o.publisher.PublishLiveGameUpdate(ctx, game); err != nil {
				log.Printf("  ⚠️  Failed to publish game %s: %v", game.GameID, err)
			}
		} else if game.Status == "final" {
			// Publish final stats
			if err := o.publisher.PublishGameStats(ctx, game); err != nil {
				log.Printf("  ⚠️  Failed to publish final stats for game %s: %v", game.GameID, err)
			}
		}
	}
	
	if liveGameCount > 0 {
		log.Printf("  ✓ Published %d live games to Redis streams", liveGameCount)
	}
}

// runDailyIngestion runs daily historical data ingestion
func (o *Orchestrator) runDailyIngestion(ctx context.Context) {
	log.Printf("→ Daily ingestion scheduler started (runs at %02d:00 daily)", o.config.DailyIngestionHour)
	
	for {
		// Calculate time until next run
		now := time.Now()
		nextRun := time.Date(now.Year(), now.Month(), now.Day(), o.config.DailyIngestionHour, 0, 0, 0, now.Location())
		
		// If we've passed today's run time, schedule for tomorrow
		if now.After(nextRun) {
			nextRun = nextRun.Add(24 * time.Hour)
		}
		
		waitDuration := time.Until(nextRun)
		log.Printf("  Next daily ingestion: %s (in %v)", nextRun.Format("2006-01-02 15:04:05"), waitDuration.Round(time.Second))
		
		// Wait until next run time
		select {
		case <-ctx.Done():
			log.Println("→ Daily ingestion scheduler stopped")
			return
		case <-time.After(waitDuration):
			log.Println()
			log.Println("═══ Daily Ingestion Starting ═══")
			o.runDailyIngestionTask(ctx)
			log.Println("═══ Daily Ingestion Complete ═══")
			log.Println()
		}
	}
}

// runDailyIngestionTask performs the daily ingestion
func (o *Orchestrator) runDailyIngestionTask(ctx context.Context) {
	startTime := time.Now()
	
	// Ingest yesterday's games (ESPN has complete data by now)
	yesterday := time.Now().Add(-24 * time.Hour)
	log.Printf("Ingesting games from %s", yesterday.Format("2006-01-02"))
	
	// Lookup season_id from season_year
	seasonID, err := o.lookupSeasonID(ctx, o.config.CurrentSeasonID)
	if err != nil {
		log.Printf("❌ Failed to lookup season ID: %v", err)
		return
	}
	
	err = o.espnIngester.IngestTodaysGames(ctx, seasonID)
	if err != nil {
		log.Printf("❌ Daily ingestion failed: %v", err)
		return
	}
	
	duration := time.Since(startTime)
	log.Printf("✓ Daily ingestion complete in %v", duration.Round(time.Second))
}

// Stop gracefully stops the scheduler
func (o *Orchestrator) Stop() {
	log.Println("Stopping scheduler orchestrator...")
	
	// Cancel live polling
	if o.liveGamesCancel != nil {
		o.liveGamesCancel()
	}
	
	// Cancel daily ingestion
	if o.dailyCancel != nil {
		o.dailyCancel()
	}
	
	// Cancel main orchestrator
	if o.cancel != nil {
		o.cancel()
	}
	
	// Close live ingester
	if o.liveIngester != nil {
		o.liveIngester.Close()
	}
	
	log.Println("✓ Scheduler orchestrator stopped")
}

// TriggerManualIngestion manually triggers an ingestion for a specific date
func (o *Orchestrator) TriggerManualIngestion(ctx context.Context, date time.Time) error {
	log.Printf("Manual ingestion triggered for %s", date.Format("2006-01-02"))
	
	// Lookup season_id from season_year
	seasonID, err := o.lookupSeasonID(ctx, o.config.CurrentSeasonID)
	if err != nil {
		return fmt.Errorf("lookup season ID: %w", err)
	}
	
	// This would use the backfill system or ESPN ingester
	// For now, delegate to ESPN ingester
	err = o.espnIngester.IngestTodaysGames(ctx, seasonID)
	if err != nil {
		return err
	}
	
	log.Printf("✓ Manual ingestion complete for %s", date.Format("2006-01-02"))
	return nil
}

// GetStatus returns current scheduler status
func (o *Orchestrator) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"live_polling_enabled":    o.config.EnableLivePolling,
		"live_poll_interval":      o.config.LivePollInterval.String(),
		"daily_ingestion_enabled": o.config.EnableDailyIngestion,
		"daily_ingestion_hour":    o.config.DailyIngestionHour,
		"current_season":          o.config.CurrentSeasonID,
	}
}

// lookupSeasonID queries the database to get season_id (INT) from season_year (STRING)
func (o *Orchestrator) lookupSeasonID(ctx context.Context, seasonYear string) (int, error) {
	query := `SELECT season_id FROM seasons WHERE season_year = $1 AND sport = 'basketball_nba' LIMIT 1`
	
	var seasonID int
	err := o.db.DB().QueryRowContext(ctx, query, seasonYear).Scan(&seasonID)
	if err != nil {
		return 0, fmt.Errorf("season '%s' not found in database: %w", seasonYear, err)
	}
	
	return seasonID, nil
}
