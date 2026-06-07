#!/usr/bin/env bash
# Start every Bastion-RAG module in the background, tracking each PID so that
# stop-all.sh can shut the whole stack down. Each module still runs as its own
# independent process — this is just a convenience wrapper over the per-module
# start-<module>.sh scripts.
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

echo "================================================================================"
echo "BASTION RAG: starting all modules"
echo "================================================================================"
echo "WARNING: default configs put Tracker REST on :8080 (clashes with Sentinel) and"
echo "         Tracker WebSocket on :8081 (clashes with Vault). Override those ports"
echo "         if you need Sentinel/Vault and Tracker up at the same time (see TESTING.md)."
echo

for m in "${ALL_MODULES[@]}"; do
  start_bg "$m" -- bash "$SCRIPTS_DIR/start-$m.sh"
done

echo
echo "All modules launched. Watch logs with:  tail -f logs/<module>.log"
echo "Stop everything with:                   scripts/stop-all.sh"
