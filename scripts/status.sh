#!/usr/bin/env bash
# Show which modules started by start-all.sh are currently running.
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

echo "Bastion-RAG module status:"
for m in "${ALL_MODULES[@]}"; do
  if is_running "$m"; then
    printf "  %-10s RUNNING (PID %s)\n" "$m" "$(cat "$PID_DIR/$m.pid")"
  else
    printf "  %-10s stopped\n" "$m"
  fi
done
