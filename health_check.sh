#!/bin/bash
set -e

HEALTH_STATUS="stable"

# Checks local health endpoint; if it is unavailable, mark app as failed.
if ! curl -fsS http://localhost:8005/api/helper/v1/healthcare/ > /dev/null 2>&1; then
    HEALTH_STATUS="failed"
fi

cat > /app/health.json <<EOF
{
  "status": "$HEALTH_STATUS",
  "timestamp": "$(TZ=America/Sao_Paulo date +"%Y-%m-%dT%H:%M:%S%:z")"
}
EOF