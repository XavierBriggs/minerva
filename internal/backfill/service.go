package backfill

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fortuna/minerva/internal/store"
)

// Request represents a backfill invocation request.
type Request struct {
	Sport     string
	SeasonID  string
	StartDate *time.Time
	EndDate   *time.Time
	GameIDs   []string
	DryRun    bool
}

// DeriveType infers the job type based on populated fields.
func (r Request) DeriveType() (JobType, error) {
	if len(r.GameIDs) > 0 {
		return JobTypeGame, nil
	}
	if r.StartDate != nil && r.EndDate != nil {
		return JobTypeDateRange, nil
	}
	if r.SeasonID != "" {
		return JobTypeSeason, nil
	}
	return "", fmt.Errorf("unable to determine job type from request")
}

// Service coordinates job persistence, execution, and status reporting.
type Service struct {
	repo   *Repository
	runner *Runner

	historyLimit int

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	logger *log.Logger
}

// NewService constructs a Service. Call Start to launch workers.
func NewService(db *store.Database, espnBaseURL string, logger *log.Logger) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	var runner *Runner
	if strings.TrimSpace(espnBaseURL) != "" {
		runner = NewRunnerWithBaseURL(db, espnBaseURL)
	} else {
		runner = NewRunner(db)
	}

	if logger == nil {
		logger = log.New(log.Writer(), "[backfill] ", log.LstdFlags)
	}

	return &Service{
		repo:         NewRepository(db),
		runner:       runner,
		historyLimit: 10,
		ctx:          ctx,
		cancel:       cancel,
		logger:       logger,
	}
}

// Start launches the background worker loop.
func (s *Service) Start() {
	if err := s.repo.ResetStuckJobs(s.ctx); err != nil {
		s.logger.Printf("failed to reset jobs: %v", err)
	}

	s.wg.Add(1)
	go s.worker()
}

// Shutdown stops workers and waits for completion.
func (s *Service) Shutdown(ctx context.Context) error {
	s.cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		s.wg.Wait()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

// Enqueue creates a new job from the provided request.
func (s *Service) Enqueue(ctx context.Context, req Request) (*Job, error) {
	if req.Sport == "" {
		req.Sport = "basketball_nba"
	}

	jobType, err := req.DeriveType()
	if err != nil {
		return nil, err
	}

	job := &Job{
		JobType:        jobType,
		Sport:          req.Sport,
		Status:         JobStatusQueued,
		StatusMessage:  sql.NullString{String: "Queued", Valid: true},
		ProgressCurrent: 0,
	}

	switch jobType {
	case JobTypeGame:
		if len(req.GameIDs) == 0 {
			return nil, fmt.Errorf("game job requires at least one game id")
		}
		job.GameIDs = req.GameIDs
		job.SeasonID = sql.NullString{String: req.SeasonID, Valid: req.SeasonID != ""}
		job.ProgressTotal = len(req.GameIDs)
	case JobTypeSeason:
		if req.SeasonID == "" {
			return nil, fmt.Errorf("season job requires season_id")
		}
		start, end := seasonWindow(req.SeasonID)
		job.SeasonID = sql.NullString{String: req.SeasonID, Valid: true}
		job.StartDate = sql.NullTime{Time: start, Valid: true}
		job.EndDate = sql.NullTime{Time: end, Valid: true}
		job.ProgressTotal = len(enumerateDates(start, end))
	case JobTypeDateRange:
		if req.StartDate == nil || req.EndDate == nil {
			return nil, fmt.Errorf("date range job requires start_date and end_date")
		}
		job.SeasonID = sql.NullString{String: req.SeasonID, Valid: req.SeasonID != ""}
		job.StartDate = sql.NullTime{Time: truncateDate(*req.StartDate), Valid: true}
		job.EndDate = sql.NullTime{Time: truncateDate(*req.EndDate), Valid: true}
		job.ProgressTotal = len(enumerateDates(job.StartDate.Time, job.EndDate.Time))
	}

	stored, err := s.repo.CreateJob(ctx, job)
	if err != nil {
		return nil, err
	}

	_ = s.repo.AppendEvent(ctx, stored.JobID, "queued", "Job queued", nil, nil)

	return stored, nil
}

// GetStatus returns the currently running job plus recent history.
func (s *Service) GetStatus(ctx context.Context) (*StatusSummary, error) {
	active, err := s.repo.GetActiveJob(ctx)
	if err != nil {
		return nil, err
	}

	history, err := s.repo.ListRecentJobs(ctx, s.historyLimit)
	if err != nil {
		return nil, err
	}

	return &StatusSummary{
		ActiveJob: active,
		History:   history,
	}, nil
}

func (s *Service) worker() {
	defer s.wg.Done()

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			job, err := s.repo.MarkNextJobRunning(s.ctx)
			if err != nil {
				s.logger.Printf("claim job error: %v", err)
				time.Sleep(time.Second)
				continue
			}
			if job == nil {
				select {
				case <-s.ctx.Done():
					return
				case <-ticker.C:
					continue
				}
			}

			s.executeJob(job)
		}
	}
}

func (s *Service) executeJob(job *Job) {
	spec, err := s.buildSpec(job)
	if err != nil {
		s.logger.Printf("invalid job spec %s: %v", job.JobID, err)
		_ = s.repo.UpdateStatus(s.ctx, job.JobID, JobStatusFailed, "Invalid job specification", err)
		return
	}

	reporter := &jobReporter{
		ctx:   s.ctx,
		repo:  s.repo,
		jobID: job.JobID,
		total: specProgressUnits(spec),
	}

	if job.ProgressTotal == 0 {
		_ = s.repo.UpdateProgress(s.ctx, job.JobID, 0, reporter.total, "Starting job...")
	}

	if err := s.runner.Run(s.ctx, spec, reporter); err != nil {
		_ = s.repo.UpdateStatus(s.ctx, job.JobID, JobStatusFailed, "Job failed", err)
		return
	}

	_ = s.repo.UpdateStatus(s.ctx, job.JobID, JobStatusCompleted, "Job completed", nil)
}

func (s *Service) buildSpec(job *Job) (JobSpec, error) {
	spec := JobSpec{
		Type:     job.JobType,
		Sport:    job.Sport,
		SeasonID: job.SeasonID.String,
	}

	switch job.JobType {
	case JobTypeGame:
		if len(job.GameIDs) == 0 {
			return spec, fmt.Errorf("game job missing game_ids")
		}
		spec.GameIDs = job.GameIDs
	case JobTypeSeason, JobTypeDateRange:
		if !job.StartDate.Valid || !job.EndDate.Valid {
			return spec, fmt.Errorf("job missing start/end dates")
		}
		spec.Start = job.StartDate.Time
		spec.End = job.EndDate.Time
	default:
		return spec, fmt.Errorf("unknown job type %s", job.JobType)
	}

	return spec, nil
}

type jobReporter struct {
	ctx   context.Context
	repo  *Repository
	jobID string
	total int
}

func (r *jobReporter) OnJobStart(spec JobSpec) {
	if r.total == 0 {
		r.total = specProgressUnits(spec)
	}
	_ = r.repo.UpdateProgress(r.ctx, r.jobID, 0, r.total, "Job starting")
}

func (r *jobReporter) OnDateStart(date time.Time, index int, total int) {
	msg := fmt.Sprintf("Processing %s (%d/%d)", date.Format("Jan 2, 2006"), index+1, total)
	cur := index
	if index == 0 {
		cur = 0
	}
	_ = r.repo.UpdateProgress(r.ctx, r.jobID, cur, valueOr(total, r.total), msg)
}

func (r *jobReporter) OnGameProcessed(gameID string) {
	_ = r.repo.AppendEvent(r.ctx, r.jobID, "game", fmt.Sprintf("Game %s processed", gameID), nil, nil)
}

func (r *jobReporter) OnProgress(message string, current int, total int) {
	_ = r.repo.UpdateProgress(r.ctx, r.jobID, current, valueOr(total, r.total), message)
}

func (r *jobReporter) OnJobComplete() {
	_ = r.repo.UpdateProgress(r.ctx, r.jobID, r.total, r.total, "Job complete")
}

func (r *jobReporter) OnJobError(err error) {
	_ = r.repo.AppendEvent(r.ctx, r.jobID, "error", err.Error(), nil, nil)
}

func specProgressUnits(spec JobSpec) int {
	switch spec.Type {
	case JobTypeGame:
		return len(spec.GameIDs)
	case JobTypeSeason, JobTypeDateRange:
		return len(enumerateDates(spec.Start, spec.End))
	default:
		return 0
	}
}

func valueOr(val, fallback int) int {
	if val > 0 {
		return val
	}
	return fallback
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

func truncateDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}


