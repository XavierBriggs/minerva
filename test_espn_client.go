package main

import (
	"context"
	"log"
	"time"

	"github.com/fortuna/minerva/internal/ingest/espn"
)

func main() {
	log.Println("Testing ESPN API client directly...")

	client := espn.NewClient()
	ctx := context.Background()

	// Test 1: Fetch scoreboard for Nov 13, 2025
	dateStr := "2025-11-13"
	date, _ := time.Parse("2006-01-02", dateStr)

	log.Printf("Fetching scoreboard for %s...", dateStr)
	data, err := client.FetchScoreboard(ctx, espn.BasketballNBA, date)

	if err != nil {
		log.Printf("❌ ERROR: %v", err)
		return
	}

	log.Printf("✅ SUCCESS!")
	
	// Check if we got events
	if events, ok := data["events"].([]interface{}); ok {
		log.Printf("   Retrieved %d events", len(events))
		if len(events) > 0 {
			// Show first event
			if event, ok := events[0].(map[string]interface{}); ok {
				log.Printf("   First event: %s", event["name"])
			}
		}
	} else {
		log.Printf("   No events array found in response")
	}

	// Test 2: Fetch game summary
	if events, ok := data["events"].([]interface{}); ok && len(events) > 0 {
		if event, ok := events[0].(map[string]interface{}); ok {
			if gameID, ok := event["id"].(string); ok {
				log.Printf("\nFetching game summary for %s...", gameID)
				summary, err := client.FetchGameSummary(ctx, espn.BasketballNBA, gameID)
				if err != nil {
					log.Printf("❌ ERROR: %v", err)
				} else {
					log.Printf("✅ SUCCESS! Got game summary")
					if boxscore, ok := summary["boxscore"].(map[string]interface{}); ok {
						log.Printf("   Has boxscore: yes")
						_ = boxscore
					}
				}
			}
		}
	}

	log.Println("\n✅ All tests passed!")
}

