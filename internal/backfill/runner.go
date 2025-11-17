package backfill

import (
	"context"
	"fmt"
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

	// Lookup season_id (INT) from season_year (STRING)
	seasonID, err := r.lookupSeasonID(ctx, spec.SeasonID)
	if err != nil {
		if reporter != nil {
			reporter.OnJobError(fmt.Errorf("lookup season ID: %w", err))
		}
		return fmt.Errorf("lookup season ID for '%s': %w", spec.SeasonID, err)
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

			if _, err := r.ingester.IngestGamesByDate(ctx, seasonID, date); err != nil {
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
func (r *Runner) lookupSeasonID(ctx context.Context, seasonYear string) (int, error) {
	query := `SELECT season_id FROM seasons WHERE season_year = $1 AND sport = 'basketball_nba' LIMIT 1`
	
	var seasonID int
	err := r.db.DB().QueryRowContext(ctx, query, seasonYear).Scan(&seasonID)
	if err != nil {
		return 0, fmt.Errorf("season '%s' not found in database: %w", seasonYear, err)
	}
	
	return seasonID, nil
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


