package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fortuna/minerva/internal/backfill"
	"github.com/fortuna/minerva/internal/store"
)

const (
	appName    = "minerva-backfill"
	appVersion = "2.0.0"
)

func main() {
	log.Printf("=== %s v%s ===", appName, appVersion)

	var (
		atlasDSN  = flag.String("dsn", getEnv("ATLAS_DSN", "postgres://fortuna:fortuna_pw@localhost:5434/atlas?sslmode=disable"), "Atlas DSN")
		espnBase  = flag.String("espn-url", getEnv("ESPN_API_BASE", "https://site.api.espn.com"), "ESPN API base URL")
		season    = flag.String("season", "", "Season to backfill (e.g., 2024-25)")
		startDate = flag.String("start", "", "Start date (YYYY-MM-DD)")
		endDate   = flag.String("end", "", "End date (YYYY-MM-DD)")
		gameID    = flag.String("game", "", "Single ESPN game ID to backfill")
		dryRun    = flag.Bool("dry-run", false, "Dry run (do not write to DB)")
	)

	flag.Parse()

	if *season == "" && *startDate == "" && *gameID == "" {
		log.Fatalf("Specify --season, --start/--end, or --game")
	}

	db, err := store.NewDatabase(*atlasDSN)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer db.Close()

	var runner *backfill.Runner
	if *espnBase != "" && *espnBase != "https://site.api.espn.com" {
		runner = backfill.NewRunnerWithBaseURL(db, *espnBase)
	} else {
		runner = backfill.NewRunner(db)
	}

	spec, err := buildSpec(*season, *startDate, *endDate, *gameID)
	if err != nil {
		log.Fatalf("build spec: %v", err)
	}
	spec.DryRun = *dryRun

	reporter := &consoleReporter{dryRun: *dryRun}

	if err := runner.Run(context.Background(), spec, reporter); err != nil {
		log.Fatalf("backfill failed: %v", err)
	}

	log.Println("✓ Backfill completed successfully")
}

func buildSpec(season, startStr, endStr, gameID string) (backfill.JobSpec, error) {
	spec := backfill.JobSpec{
		Sport:    "basketball_nba",
		SeasonID: season,
	}

	switch {
	case gameID != "":
		spec.Type = backfill.JobTypeGame
		spec.GameIDs = []string{gameID}
	case season != "":
		spec.Type = backfill.JobTypeSeason
		start, end := seasonWindow(season)
		spec.Start = start
		spec.End = end
	case startStr != "" && endStr != "":
		spec.Type = backfill.JobTypeDateRange
		start, err := time.Parse("2006-01-02", startStr)
		if err != nil {
			return spec, fmt.Errorf("invalid start date: %w", err)
		}
		end, err := time.Parse("2006-01-02", endStr)
		if err != nil {
			return spec, fmt.Errorf("invalid end date: %w", err)
		}
		spec.Start = start
		spec.End = end
	default:
		return spec, fmt.Errorf("unable to determine job type")
	}

	return spec, nil
}

func seasonWindow(seasonID string) (time.Time, time.Time) {
	parts := strings.Split(seasonID, "-")
	if len(parts) != 2 {
		year, _ := strconv.Atoi(seasonID)
		start := time.Date(year, time.October, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(year+1, time.July, 1, 0, 0, 0, 0, time.UTC)
		return start, end
	}

	startYear, _ := strconv.Atoi(parts[0])
	start := time.Date(startYear, time.October, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(startYear+1, time.July, 1, 0, 0, 0, 0, time.UTC)
	return start, end
}

type consoleReporter struct {
	dryRun bool
}

func (c *consoleReporter) OnJobStart(spec backfill.JobSpec) {
	log.Printf("Starting %s job (dry_run=%v)", spec.Type, c.dryRun)
}

func (c *consoleReporter) OnDateStart(date time.Time, index int, total int) {
	log.Printf("[%d/%d] %s", index+1, total, date.Format("2006-01-02"))
}

func (c *consoleReporter) OnGameProcessed(gameID string) {
	log.Printf("Processed game %s", gameID)
}

func (c *consoleReporter) OnProgress(message string, current int, total int) {
	log.Printf("Progress: %s (%d/%d)", message, current, total)
}

func (c *consoleReporter) OnJobComplete() {
	log.Println("Job complete")
}

func (c *consoleReporter) OnJobError(err error) {
	log.Printf("Job error: %v", err)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

