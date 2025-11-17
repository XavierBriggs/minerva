-- Drop all existing tables to prepare for schema redesign
-- This migration wipes the database to implement the improved schema from ATLAS_DATABASE_DESIGN.md

-- Drop dependent tables first (in reverse order of creation)
DROP TABLE IF EXISTS backfill_job_events CASCADE;
DROP TABLE IF EXISTS backfill_jobs CASCADE;
DROP TABLE IF EXISTS odds_mappings CASCADE;
DROP TABLE IF EXISTS team_game_stats CASCADE;
DROP TABLE IF EXISTS player_game_stats CASCADE;
DROP TABLE IF EXISTS games CASCADE;
DROP TABLE IF EXISTS player_seasons CASCADE;
DROP TABLE IF EXISTS players CASCADE;
DROP TABLE IF EXISTS teams CASCADE;
DROP TABLE IF EXISTS seasons CASCADE;

-- Drop any custom types if they exist
DROP TYPE IF EXISTS job_status CASCADE;
DROP TYPE IF EXISTS job_type CASCADE;
DROP TYPE IF EXISTS game_status CASCADE;


