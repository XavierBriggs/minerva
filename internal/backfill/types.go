package backfill

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
)

// JobType enumerates the supported backfill job variants.
type JobType string

const (
	JobTypeSeason    JobType = "season"
	JobTypeDateRange JobType = "date_range"
	JobTypeGame      JobType = "game"
)

// JobStatus represents the lifecycle state for a job.
type JobStatus string

const (
	JobStatusQueued    JobStatus = "queued"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// Job models the database representation of a backfill job.
type Job struct {
	JobID          string
	JobType        JobType
	Sport          string
	SeasonID       sql.NullString
	StartDate      sql.NullTime
	EndDate        sql.NullTime
	GameIDs        pq.StringArray
	Status         JobStatus
	StatusMessage  sql.NullString
	ProgressCurrent int
	ProgressTotal   int
	LastError      sql.NullString
	RetryCount     int
	CreatedAt      time.Time
	UpdatedAt      time.Time
	StartedAt      sql.NullTime
	CompletedAt    sql.NullTime
}

// Copy returns a shallow copy to prevent external mutation.
func (j *Job) Copy() *Job {
	if j == nil {
		return nil
	}
	cpy := *j
	return &cpy
}

// JobSpec describes the work to be performed by the runner.
type JobSpec struct {
	Type     JobType
	Sport    string
	SeasonID string
	Start    time.Time
	End      time.Time
	GameIDs  []string
	DryRun   bool
}

// Reporter receives lifecycle callbacks from the runner.
type Reporter interface {
	OnJobStart(spec JobSpec)
	OnDateStart(date time.Time, index int, total int)
	OnGameProcessed(gameID string)
	OnProgress(message string, current int, total int)
	OnJobComplete()
	OnJobError(err error)
}

// StatusSummary is returned to API callers.
type StatusSummary struct {
	ActiveJob *Job   `json:"active_job,omitempty"`
	History   []*Job `json:"recent_jobs,omitempty"`
}


