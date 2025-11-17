package backfill

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/fortuna/minerva/internal/store"
)

// Repository handles persistence for backfill jobs and events.
type Repository struct {
	db *store.Database
}

// NewRepository constructs a Repository.
func NewRepository(db *store.Database) *Repository {
	return &Repository{db: db}
}

// CreateJob inserts a new job row and returns the stored record.
func (r *Repository) CreateJob(ctx context.Context, job *Job) (*Job, error) {
	query := `
		INSERT INTO backfill_jobs (
			job_type, sport, season_id, start_date, end_date, game_ids,
			status, status_message, progress_current, progress_total
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING job_id, job_type, sport, season_id, start_date, end_date, game_ids,
			status, status_message, progress_current, progress_total,
			last_error, retry_count, created_at, updated_at, started_at, completed_at
	`

	row := r.db.DB().QueryRowContext(ctx, query,
		job.JobType, job.Sport, job.SeasonID, job.StartDate, job.EndDate, job.GameIDs,
		job.Status, job.StatusMessage, job.ProgressCurrent, job.ProgressTotal,
	)

	return scanJob(row)
}

// UpdateStatus updates status, message and optional error.
func (r *Repository) UpdateStatus(ctx context.Context, jobID string, status JobStatus, message string, lastErr error) error {
	query := `
		UPDATE backfill_jobs
		SET status = $2::varchar,
			status_message = $3,
			last_error = $4,
			updated_at = NOW(),
			completed_at = CASE WHEN $2::varchar IN ('completed','failed','cancelled') THEN NOW() ELSE completed_at END
		WHERE job_id = $1
	`

	var errText sql.NullString
	if lastErr != nil {
		errText = sql.NullString{String: lastErr.Error(), Valid: true}
	}

	if _, err := r.db.DB().ExecContext(ctx, query, jobID, string(status), message, errText); err != nil {
		return fmt.Errorf("update job status: %w", err)
	}

	return nil
}

// UpdateProgress updates the progress counters and optional message.
func (r *Repository) UpdateProgress(ctx context.Context, jobID string, current, total int, message string) error {
	query := `
		UPDATE backfill_jobs
		SET progress_current = $2,
			progress_total = $3,
			status_message = $4,
			updated_at = NOW()
		WHERE job_id = $1
	`

	if _, err := r.db.DB().ExecContext(ctx, query, jobID, current, total, message); err != nil {
		return fmt.Errorf("update job progress: %w", err)
	}

	return nil
}

// AppendEvent stores a log entry for a job.
func (r *Repository) AppendEvent(ctx context.Context, jobID string, eventType, message string, current, total *int) error {
	query := `
		INSERT INTO backfill_job_events (job_id, event_type, message, progress_current, progress_total)
		VALUES ($1,$2,$3,$4,$5)
	`

	var currentVal interface{}
	if current != nil {
		currentVal = *current
	}
	var totalVal interface{}
	if total != nil {
		totalVal = *total
	}

	if _, err := r.db.DB().ExecContext(ctx, query, jobID, eventType, message, currentVal, totalVal); err != nil {
		return fmt.Errorf("insert job event: %w", err)
	}
	return nil
}

// ResetStuckJobs moves running jobs back to queued (used during service restarts).
func (r *Repository) ResetStuckJobs(ctx context.Context) error {
	_, err := r.db.DB().ExecContext(ctx, `
		UPDATE backfill_jobs
		SET status = 'queued',
			status_message = 'Reset after service restart',
			updated_at = NOW()
		WHERE status = 'running'
	`)
	if err != nil {
		return fmt.Errorf("reset stuck jobs: %w", err)
	}
	return nil
}

// MarkNextJobRunning atomically claims the next queued job.
func (r *Repository) MarkNextJobRunning(ctx context.Context) (*Job, error) {
	query := `
		WITH next_job AS (
			SELECT job_id
			FROM backfill_jobs
			WHERE status = 'queued'
			ORDER BY created_at
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE backfill_jobs
		SET status = 'running',
			status_message = 'Starting job...',
			started_at = COALESCE(started_at, NOW()),
			updated_at = NOW()
		FROM next_job
		WHERE backfill_jobs.job_id = next_job.job_id
		RETURNING backfill_jobs.job_id, backfill_jobs.job_type, backfill_jobs.sport,
			backfill_jobs.season_id, backfill_jobs.start_date, backfill_jobs.end_date,
			backfill_jobs.game_ids, backfill_jobs.status, backfill_jobs.status_message,
			backfill_jobs.progress_current, backfill_jobs.progress_total,
			backfill_jobs.last_error, backfill_jobs.retry_count,
			backfill_jobs.created_at, backfill_jobs.updated_at,
			backfill_jobs.started_at, backfill_jobs.completed_at
	`

	row := r.db.DB().QueryRowContext(ctx, query)
	job, err := scanJob(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return job, nil
}

// GetActiveJob returns the currently running job, if any.
func (r *Repository) GetActiveJob(ctx context.Context) (*Job, error) {
	query := `
		SELECT job_id, job_type, sport, season_id, start_date, end_date, game_ids,
			status, status_message, progress_current, progress_total,
			last_error, retry_count, created_at, updated_at, started_at, completed_at
		FROM backfill_jobs
		WHERE status = 'running'
		ORDER BY started_at DESC
		LIMIT 1
	`

	row := r.db.DB().QueryRowContext(ctx, query)
	job, err := scanJob(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get active job: %w", err)
	}
	return job, nil
}

// ListRecentJobs returns the most recent completed jobs.
func (r *Repository) ListRecentJobs(ctx context.Context, limit int) ([]*Job, error) {
	query := `
		SELECT job_id, job_type, sport, season_id, start_date, end_date, game_ids,
			status, status_message, progress_current, progress_total,
			last_error, retry_count, created_at, updated_at, started_at, completed_at
		FROM backfill_jobs
		ORDER BY created_at DESC
		LIMIT $1
	`

	rows, err := r.db.DB().QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

func scanJob(scanner interface {
	Scan(dest ...interface{}) error
}) (*Job, error) {
	job := &Job{}
	err := scanner.Scan(
		&job.JobID,
		&job.JobType,
		&job.Sport,
		&job.SeasonID,
		&job.StartDate,
		&job.EndDate,
		&job.GameIDs,
		&job.Status,
		&job.StatusMessage,
		&job.ProgressCurrent,
		&job.ProgressTotal,
		&job.LastError,
		&job.RetryCount,
		&job.CreatedAt,
		&job.UpdatedAt,
		&job.StartedAt,
		&job.CompletedAt,
	)
	if err != nil {
		return nil, err
	}
	return job, nil
}


