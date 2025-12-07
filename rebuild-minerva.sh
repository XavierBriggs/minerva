#!/bin/bash
# Quick script to rebuild and restart Minerva

cd "$(dirname "$0")/deploy"

echo "=== Stopping Minerva ==="
docker-compose stop minerva

echo ""
echo "=== Rebuilding Minerva (no cache) ==="
docker-compose build --no-cache minerva

echo ""
echo "=== Starting Minerva ==="
docker-compose up -d minerva

echo ""
echo "=== Waiting for health check ==="
sleep 10

echo ""
echo "=== Service Status ==="
docker-compose ps minerva

echo ""
echo "=== Testing Teams Endpoint ==="
curl -s http://localhost:8085/api/v1/teams | jq '.teams | length' 2>/dev/null || echo "Endpoint check (service may still be starting)"

echo ""
echo "=== Recent Logs ==="
docker-compose logs --tail=20 minerva


