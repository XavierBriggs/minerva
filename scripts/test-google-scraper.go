package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/fortuna/minerva/internal/ingest/google"
)

// Simple test utility to verify Google Sports scraper works
func main() {
	log.Println("Testing Google Sports Scraper")
	log.Println("===============================")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create scraper client
	client, err := google.NewClient()
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Test fetching live games
	log.Println("\n1. Fetching live NBA games...")
	htmlContent, err := client.FetchLiveGames(ctx)
	if err != nil {
		log.Fatalf("Failed to fetch live games: %v", err)
	}

	log.Printf("✓ Retrieved HTML content (%d bytes)", len(htmlContent))

	// Parse HTML
	doc, err := google.ParseHTML(htmlContent)
	if err != nil {
		log.Fatalf("Failed to parse HTML: %v", err)
	}

	// Extract games
	games, err := google.ParseLiveGames(doc)
	if err != nil {
		log.Fatalf("Failed to parse games: %v", err)
	}

	log.Printf("✓ Found %d games\n", len(games))

	// Display results
	if len(games) == 0 {
		log.Println("No live games currently available")
		log.Println("(This is expected if run when no NBA games are scheduled)")
	} else {
		for i, game := range games {
			log.Printf("\nGame %d:", i+1)
			log.Printf("  %s vs %s", game.AwayTeam, game.HomeTeam)
			log.Printf("  Score: %d - %d", game.AwayScore, game.HomeScore)
			log.Printf("  Status: %s", game.GameStatus)
			if game.Period > 0 {
				log.Printf("  Period: %d", game.Period)
			}
			if game.TimeRemaining != "" {
				log.Printf("  Time: %s", game.TimeRemaining)
			}
			log.Printf("  Live: %v", game.IsLive)
		}
	}

	// Test specific game fetch (Lakers vs Celtics example)
	log.Println("\n2. Testing specific game fetch (Lakers vs Celtics)...")
	_, err = client.FetchGameDetails(ctx, "Lakers", "Celtics")
	if err != nil {
		log.Printf("Note: Specific game fetch may fail if no such game is scheduled")
		log.Printf("Error: %v", err)
	} else {
		log.Println("✓ Successfully fetched game details")
	}

	log.Println("\n===============================")
	log.Println("✓ Google Sports Scraper Test Complete")
	
	os.Exit(0)
}


