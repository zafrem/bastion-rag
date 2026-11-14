#!/usr/bin/env bash
# Wrapper around scripts/run-integration.sh so `make run-tests` and
# `./scripts/run-tests.sh` run the full integration test suite cleanly
# without writing log files into the repository root.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec "$SCRIPT_DIR/run-integration.sh" "$@"
