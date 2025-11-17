package google

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/fortuna/minerva/internal/cache"
	"github.com/fortuna/minerva/internal/store"
)

// Ingester handles ingestion of Google Sports data
type Ingester struct {
	client *Client
	cache  *cache.RedisCache
	db     *store.Database
}

// NewIngester creates a new Google Sports ingester
func NewIngester(cache *cache.RedisCache, db *store.Database) (*Ingester, error) {
	client, err := NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Google client: %w", err)
	}
	
	return &Ingester{
		client: client,
		cache:  cache,
		db:     db,
	}, nil
}

// Close releases resources
func (i *Ingester) Close() {
	if i.client != nil {
		i.client.Close()
	}
}

// IngestLiveGames fetches and caches current live NBA games
func (i *Ingester) IngestLiveGames(ctx context.Context, seasonID string) ([]LiveGame, error) {
	log.Println("Ingesting live games from Google Sports...")
	
	// Check cache first
	cacheKey := "google:live_games:nba"
	if i.cache != nil {
		cached, err := i.cache.Get(ctx, cacheKey)
		if err == nil && cached != "" {
			log.Println("  Using cached live games data")
			// TODO: Deserialize and return cached data
		}
	}
	
	// Fetch from Google
	htmlContent, err := i.client.FetchLiveGames(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch live games: %w", err)
	}
	
	// Parse HTML
	doc, err := ParseHTML(htmlContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}
	
	// Extract games
	games, err := ParseLiveGames(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to parse games: %w", err)
	}
	
	log.Printf("  Found %d live games", len(games))
	
	// Cache results (5 second TTL for live data)
	if i.cache != nil {
		// TODO: Serialize and cache games
		_ = i.cache.Set(ctx, cacheKey, fmt.Sprintf("%d games", len(games)), 5*time.Second)
	}
	
	return games, nil
}

// IngestGameDetails fetches detailed information for a specific matchup
func (i *Ingester) IngestGameDetails(ctx context.Context, homeTeam, awayTeam string) (*LiveGame, error) {
	log.Printf("Ingesting game details for %s vs %s from Google...", homeTeam, awayTeam)
	
	htmlContent, err := i.client.FetchGameDetails(ctx, homeTeam, awayTeam)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch game details: %w", err)
	}
	
	doc, err := ParseHTML(htmlContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}
	
	games, err := ParseLiveGames(doc)
	if err != nil || len(games) == 0 {
		return nil, fmt.Errorf("no game found for matchup")
	}
	
	return &games[0], nil
}

// PollLiveGames continuously polls for live game updates
func (i *Ingester) PollLiveGames(ctx context.Context, seasonID string, interval time.Duration, callback func([]LiveGame)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	log.Printf("Starting Google Sports live game polling (interval: %v)", interval)
	
	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping Google Sports polling")
			return
		case <-ticker.C:
			games, err := i.IngestLiveGames(ctx, seasonID)
			if err != nil {
				log.Printf("Error polling live games: %v", err)
				continue
			}
			
			if callback != nil && len(games) > 0 {
				callback(games)
			}
		}
	}
}


