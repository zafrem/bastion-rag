#!/usr/bin/env bash
# Shared helpers for the Bastion-RAG module start/stop scripts.
# Sourced by start-<module>.sh, start-all.sh and stop-all.sh.

set -euo pipefail

# Repo root = parent of the scripts/ directory, resolved from this file's path
# so the scripts work no matter what directory they are invoked from.
SCRIPTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPTS_DIR/.." && pwd)"
LOG_DIR="$ROOT/logs"
PID_DIR="$ROOT/run"

# Modules in start order. NOTE: Tracker REST (8080) overlaps Sentinel REST
# (8080) and Tracker WebSocket (8081) overlaps Vault REST (8081) in the default
# configs, so running the whole stack natively needs those overridden — see
# TESTING.md. Tracker is started last so the conflict surfaces clearly.
ALL_MODULES=(sentinel vault navigator anchor tracker)

ensure_dirs() {
  mkdir -p "$LOG_DIR" "$PID_DIR"
}

# Pick a Python interpreter: a module-local virtualenv if present, else python3.
python_for() {
  local module_dir="$1"
  if [[ -x "$module_dir/.venv/bin/python" ]]; then
    echo "$module_dir/.venv/bin/python"
  elif [[ -x "$module_dir/venv/bin/python" ]]; then
    echo "$module_dir/venv/bin/python"
  else
    echo "python3"
  fi
}

# start_bg <name> -- <command...>
# Runs the command in the background, recording its PID so stop-all can find it.
# Each module resolves to a single long-lived process (a built Go binary or a
# python -m invocation), so a plain PID is enough to track and later signal it.
start_bg() {
  local name="$1"
  shift 2  # drop name and the literal "--"
  ensure_dirs
  local pidfile="$PID_DIR/$name.pid"
  if is_running "$name"; then
    echo "  $name already running (PID $(cat "$pidfile"))"
    return 0
  fi
  ( exec "$@" ) >"$LOG_DIR/$name.log" 2>&1 &
  echo $! >"$pidfile"
  echo "  $name started (PID $!) → logs/$name.log"
}

is_running() {
  local name="$1"
  local pidfile="$PID_DIR/$name.pid"
  [[ -f "$pidfile" ]] || return 1
  local pid
  pid="$(cat "$pidfile")"
  [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null
}

stop_module() {
  local name="$1"
  local pidfile="$PID_DIR/$name.pid"
  if [[ ! -f "$pidfile" ]]; then
    echo "  $name not tracked (no pidfile)"
    return 0
  fi
  local pid
  pid="$(cat "$pidfile")"
  if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
    pkill -TERM -P "$pid" 2>/dev/null || true  # any stray children first
    kill -TERM "$pid" 2>/dev/null || true
    echo "  $name stopped (PID $pid)"
  else
    echo "  $name was not running"
  fi
  rm -f "$pidfile"
}
