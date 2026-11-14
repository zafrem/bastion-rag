#!/usr/bin/env bash
# Start every Bastion-RAG module in the background, tracking each PID so that
# stop-all.sh can shut the whole stack down. Each module still runs as its own
# independent process — this is just a convenience wrapper over the per-module
# start-<module>.sh scripts.
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

echo "================================================================================"
echo "BASTION RAG: starting all modules"
echo "================================================================================"
export TRACKER_REST_PORT="${TRACKER_REST_PORT:-8084}"
export TRACKER_GRPC_PORT="${TRACKER_GRPC_PORT:-9094}"
echo "  Tracker REST/UI port: :$TRACKER_REST_PORT (gRPC :$TRACKER_GRPC_PORT) to avoid colliding with Sentinel (:8080/:9090)"
echo

for m in "${ALL_MODULES[@]}"; do
  start_bg "$m" -- bash "$SCRIPTS_DIR/start-$m.sh"
done

echo
echo "All modules launched. Watch logs with:  tail -f logs/<module>.log"
echo "Stop everything with:                   scripts/stop-all.sh"
