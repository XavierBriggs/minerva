#!/bin/bash
# Minerva Backfill Examples
# Usage examples for the backfill CLI tool

set -e

ATLAS_DSN="${ATLAS_DSN:-postgres://fortuna:fortuna_pw@localhost:5434/atlas?sslmode=disable}"

echo "=== Minerva Historical Data Backfill Examples ==="
echo ""

# Example 1: Backfill entire 2024-25 season
echo "Example 1: Backfill entire 2024-25 season"
echo "go run ./cmd/backfill --season 2024-25"
echo ""

# Example 2: Backfill date range (last 7 days)
echo "Example 2: Backfill last 7 days"
YESTERDAY=$(date -v-1d +%Y-%m-%d)
WEEK_AGO=$(date -v-7d +%Y-%m-%d)
echo "go run ./cmd/backfill --start $WEEK_AGO --end $YESTERDAY"
echo ""

# Example 3: Backfill specific game
echo "Example 3: Backfill specific game (use actual ESPN game ID)"
echo "go run ./cmd/backfill --game 401584894"
echo ""

# Example 4: Dry run for 2023-24 season
echo "Example 4: Dry run for 2023-24 season (preview only)"
echo "go run ./cmd/backfill --season 2023-24 --dry-run"
echo ""

# Example 5: Backfill via API
echo "Example 5: Trigger backfill via REST API"
echo "curl -X POST http://localhost:8085/api/v1/backfill \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -d '{\"season\": \"2024-25\"}'"
echo ""

# Example 6: Check backfill status
echo "Example 6: Check backfill status"
echo "curl http://localhost:8085/api/v1/backfill/status"
echo ""

echo "=== Season Ranges ==="
echo "2024-25: October 2024 - June 2025 (current)"
echo "2023-24: October 2023 - June 2024"
echo "2022-23: October 2022 - June 2023"
echo ""

echo "=== Usage ==="
echo "CLI Tool: go run ./cmd/backfill [options]"
echo "  --season    Season to backfill (e.g., 2024-25)"
echo "  --start     Start date (YYYY-MM-DD)"
echo "  --end       End date (YYYY-MM-DD)"
echo "  --game      Specific ESPN game ID"
echo "  --dry-run   Preview without writing to database"
echo "  --dsn       Atlas database connection string"
echo ""
echo "API Endpoints:"
echo "  POST /api/v1/backfill       Trigger backfill job"
echo "  GET  /api/v1/backfill/status  Check backfill status"


