package backfill

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/fortuna/minerva/internal/ingest/espn"
	"github.com/fortuna/minerva/internal/store"
)

// Runner executes backfill specs using the ESPN ingester.
type Runner struct {
	ingester *espn.Ingester
	db       *store.Database
}

// NewRunner constructs a runner with the default ESPN base URL.
func NewRunner(db *store.Database) *Runner {
	return &Runner{
		ingester: espn.NewIngester(db),
		db:       db,
	}
}

// NewRunnerWithBaseURL overrides the ESPN API base URL (useful for tests).
func NewRunnerWithBaseURL(db *store.Database, baseURL string) *Runner {
	return &Runner{
		ingester: espn.NewIngesterWithBaseURL(db, baseURL),
		db:       db,
	}
}

// Run executes the job spec, reporting progress via the Reporter if provided.
func (r *Runner) Run(ctx context.Context, spec JobSpec, reporter Reporter) error {
	if reporter != nil {
		reporter.OnJobStart(spec)
	}

	if spec.DryRun {
		if reporter != nil {
			reporter.OnProgress("Dry-run mode: no data will be written", 0, 0)
			reporter.OnJobComplete()
		}
		return nil
	}

	// Lookup season_id (INT) from season_year (STRING) or derive from date
	var seasonID int
	var err error

	if spec.SeasonID != "" {
		seasonID, err = r.lookupSeasonID(ctx, spec.SeasonID)
		if err != nil {
			if reporter != nil {
				reporter.OnJobError(fmt.Errorf("lookup season ID: %w", err))
			}
			return fmt.Errorf("lookup season ID for '%s': %w", spec.SeasonID, err)
		}
	} else if !spec.Start.IsZero() {
		// Auto-detect season from start date
		var seasonYear string
		seasonID, seasonYear, err = r.lookupSeasonIDByDate(ctx, spec.Start)
		if err != nil {
			if reporter != nil {
				reporter.OnJobError(fmt.Errorf("auto-detect season: %w", err))
			}
			return fmt.Errorf("auto-detect season for date %s: %w", spec.Start.Format("2006-01-02"), err)
		}
		if reporter != nil {
			reporter.OnProgress(fmt.Sprintf("Auto-detected season: %s", seasonYear), 0, 0)
		}
	} else {
		return fmt.Errorf("no season_id provided and cannot auto-detect without date range")
	}

	switch spec.Type {
	case JobTypeGame:
		if len(spec.GameIDs) == 0 {
			return fmt.Errorf("no game IDs provided for job type 'game'")
		}
		total := len(spec.GameIDs)
		for idx, gameID := range spec.GameIDs {
			if err := ctx.Err(); err != nil {
				return err
			}

			if reporter != nil {
				reporter.OnProgress(fmt.Sprintf("Processing game %s (%d/%d)", gameID, idx+1, total), idx, total)
			}

			if _, err := r.ingester.IngestGameByID(ctx, seasonID, gameID); err != nil {
				if reporter != nil {
					reporter.OnJobError(err)
				}
				return err
			}

			if reporter != nil {
				reporter.OnGameProcessed(gameID)
				reporter.OnProgress(fmt.Sprintf("✓ Game %s complete", gameID), idx+1, total)
			}
		}
	case JobTypeSeason, JobTypeDateRange:
		dates := enumerateDates(spec.Start, spec.End)
		if len(dates) == 0 {
			if reporter != nil {
				reporter.OnProgress("No dates to process", 0, 0)
			}
			break
		}

		total := len(dates)
		for idx, date := range dates {
			if err := ctx.Err(); err != nil {
				return err
			}

			if reporter != nil {
				reporter.OnDateStart(date, idx, total)
			}

			// For date range jobs, dynamically detect season type from ESPN
			// This handles dates that cross preseason/regular/playoffs boundaries
			dateSeasonID := seasonID
			if spec.Type == JobTypeDateRange || spec.SeasonID == "" {
				detectedID, seasonType, err := r.detectSeasonForDate(ctx, date)
				if err != nil {
					log.Printf("[backfill] Warning: Could not detect season type for %s, using fallback: %v",
						date.Format("2006-01-02"), err)
				} else {
					dateSeasonID = detectedID
					log.Printf("[backfill] Date %s -> season type: %s (id: %d)",
						date.Format("2006-01-02"), seasonType, detectedID)
				}
			}

			if _, err := r.ingester.IngestGamesByDate(ctx, dateSeasonID, date); err != nil {
				if reporter != nil {
					reporter.OnJobError(err)
				}
				return err
			}

			if reporter != nil {
				reporter.OnProgress(fmt.Sprintf("Processed %s", date.Format("Jan 2, 2006")), idx+1, total)
			}
		}
	default:
		return fmt.Errorf("unsupported job type %s", spec.Type)
	}

	if reporter != nil {
		reporter.OnJobComplete()
	}

	return nil
}

// lookupSeasonID queries the database to get season_id (INT) from season_year (STRING)
// Defaults to 'regular' season type
func (r *Runner) lookupSeasonID(ctx context.Context, seasonYear string) (int, error) {
	return r.lookupSeasonIDWithType(ctx, seasonYear, "regular")
}

// lookupSeasonIDWithType queries the database to get season_id by year and type
func (r *Runner) lookupSeasonIDWithType(ctx context.Context, seasonYear, seasonType string) (int, error) {
	query := `SELECT season_id FROM seasons WHERE season_year = $1 AND season_type = $2 AND sport = 'basketball_nba' LIMIT 1`

	var seasonID int
	err := r.db.DB().QueryRowContext(ctx, query, seasonYear, seasonType).Scan(&seasonID)
	if err != nil {
		return 0, fmt.Errorf("season '%s' type '%s' not found in database: %w", seasonYear, seasonType, err)
	}

	return seasonID, nil
}

// detectSeasonForDate fetches ESPN scoreboard for the date and determines the correct season
// ESPN provides season type in scoreboard response: 1=preseason, 2=regular, 3=playoffs
func (r *Runner) detectSeasonForDate(ctx context.Context, date time.Time) (int, string, error) {
	client := espn.NewClient()
	scoreboard, err := client.FetchScoreboard(ctx, espn.BasketballNBA, date)
	if err != nil {
		// Fallback to date-based lookup
		return r.lookupSeasonIDByDate(ctx, date)
	}

	// Extract season info from ESPN response
	// Structure: leagues[0].season.type.id and leagues[0].season.displayName
	seasonYear := ""
	seasonType := "regular" // default

	if leagues, ok := scoreboard["leagues"].([]interface{}); ok && len(leagues) > 0 {
		if league, ok := leagues[0].(map[string]interface{}); ok {
			if season, ok := league["season"].(map[string]interface{}); ok {
				// Get season year (e.g., "2024-25")
				if displayName, ok := season["displayName"].(string); ok {
					seasonYear = displayName
				}
				// Get season type
				if typeInfo, ok := season["type"].(map[string]interface{}); ok {
					if typeID, ok := typeInfo["id"].(string); ok {
						switch typeID {
						case "1":
							seasonType = "preseason"
						case "2":
							seasonType = "regular"
						case "3":
							seasonType = "playoffs"
						}
					} else if typeIDFloat, ok := typeInfo["id"].(float64); ok {
						switch int(typeIDFloat) {
						case 1:
							seasonType = "preseason"
						case 2:
							seasonType = "regular"
						case 3:
							seasonType = "playoffs"
						}
					}
				}
			}
		}
	}

	if seasonYear == "" {
		// Fallback if we couldn't parse ESPN response
		return r.lookupSeasonIDByDate(ctx, date)
	}

	// Look up season by year and type
	seasonID, err := r.lookupSeasonIDWithType(ctx, seasonYear, seasonType)
	if err != nil {
		// If specific type not found, try regular season
		log.Printf("[backfill] Season %s type %s not found, trying regular", seasonYear, seasonType)
		seasonID, err = r.lookupSeasonIDWithType(ctx, seasonYear, "regular")
		if err != nil {
			return r.lookupSeasonIDByDate(ctx, date)
		}
		seasonType = "regular"
	}

	return seasonID, seasonType, nil
}

// lookupSeasonIDByDate finds the season that contains the given date
// NBA seasons run Oct-Apr, so dates in the off-season (May-Sep) map to the most recent completed season
func (r *Runner) lookupSeasonIDByDate(ctx context.Context, date time.Time) (int, string, error) {
	// First try exact match within season dates
	query := `
		SELECT season_id, season_year 
		FROM seasons 
		WHERE sport = 'basketball_nba' 
		  AND start_date <= $1 
		  AND end_date >= $1
		ORDER BY start_date DESC
		LIMIT 1
	`

	var seasonID int
	var seasonYear string
	err := r.db.DB().QueryRowContext(ctx, query, date).Scan(&seasonID, &seasonYear)
	if err == nil {
		return seasonID, seasonYear, nil
	}

	// No exact match - likely in off-season (May-Sep) or playoffs extending past regular season
	// For dates after a season ends but before the next starts, use the most recent season that ended
	// This handles playoff/finals data and off-season queries
	fallbackQuery := `
		SELECT season_id, season_year 
		FROM seasons 
		WHERE sport = 'basketball_nba'
		  AND end_date < $1
		ORDER BY end_date DESC
		LIMIT 1
	`
	err = r.db.DB().QueryRowContext(ctx, fallbackQuery, date).Scan(&seasonID, &seasonYear)
	if err == nil {
		return seasonID, seasonYear, nil
	}

	// Still no match - try finding the next upcoming season (for future dates)
	futureQuery := `
		SELECT season_id, season_year 
		FROM seasons 
		WHERE sport = 'basketball_nba'
		  AND start_date > $1
		ORDER BY start_date ASC
		LIMIT 1
	`
	err = r.db.DB().QueryRowContext(ctx, futureQuery, date).Scan(&seasonID, &seasonYear)
	if err != nil {
		return 0, "", fmt.Errorf("no season found for date %s: %w", date.Format("2006-01-02"), err)
	}

	return seasonID, seasonYear, nil
}

func enumerateDates(start, end time.Time) []time.Time {
	if end.Before(start) {
		start, end = end, start
	}

	var dates []time.Time
	current := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	final := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)

	for !current.After(final) {
		dates = append(dates, current)
		current = current.AddDate(0, 0, 1)
	}

	return dates
}
