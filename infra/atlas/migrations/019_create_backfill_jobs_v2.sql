-- Create backfill_jobs tables (v2 - improved schema)
-- Based on ATLAS_DATABASE_DESIGN.md

CREATE TABLE backfill_jobs (
  job_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  sport VARCHAR(50) NOT NULL,
  job_type VARCHAR(20) NOT NULL,           -- 'season', 'date_range', 'game'
  season_id VARCHAR(20),
  start_date DATE,
  end_date DATE,
  game_ids TEXT[],                         -- For specific game backfills
  status VARCHAR(20) NOT NULL DEFAULT 'queued',
  status_message TEXT,
  progress_current INTEGER DEFAULT 0,
  progress_total INTEGER DEFAULT 0,
  items_processed INTEGER DEFAULT 0,
  items_failed INTEGER DEFAULT 0,
  last_error TEXT,
  retry_count INTEGER DEFAULT 0,
  max_retries INTEGER DEFAULT 3,
  priority INTEGER DEFAULT 5,              -- 1-10 (10 = highest)
  created_at TIMESTAMP DEFAULT NOW(),
  started_at TIMESTAMP,
  completed_at TIMESTAMP,
  updated_at TIMESTAMP DEFAULT NOW(),
  
  CONSTRAINT backfill_jobs_valid_type CHECK (job_type IN ('season', 'date_range', 'game')),
  CONSTRAINT backfill_jobs_valid_status CHECK (status IN ('queued', 'running', 'completed', 'failed', 'cancelled')),
  CONSTRAINT backfill_jobs_valid_dates CHECK (end_date IS NULL OR end_date >= start_date)
);

CREATE INDEX idx_backfill_jobs_status ON backfill_jobs(status, priority DESC) WHERE status IN ('queued', 'running');
CREATE INDEX idx_backfill_jobs_sport ON backfill_jobs(sport, created_at DESC);
CREATE INDEX idx_backfill_jobs_created ON backfill_jobs(created_at DESC);

COMMENT ON TABLE backfill_jobs IS 'Tracks historical data backfill operations';
COMMENT ON COLUMN backfill_jobs.priority IS 'Job priority (1-10, higher = more urgent)';

-- Create backfill_job_events table
CREATE TABLE backfill_job_events (
  event_id BIGSERIAL PRIMARY KEY,
  job_id UUID NOT NULL REFERENCES backfill_jobs(job_id) ON DELETE CASCADE,
  event_type VARCHAR(50) NOT NULL,         -- 'started', 'progress', 'error', 'completed'
  message TEXT,
  details JSONB DEFAULT '{}',              -- Error details, progress info
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_backfill_job_events_job ON backfill_job_events(job_id, created_at);
CREATE INDEX idx_backfill_job_events_type ON backfill_job_events(event_type, created_at);

COMMENT ON TABLE backfill_job_events IS 'Detailed event log for backfill operations';


