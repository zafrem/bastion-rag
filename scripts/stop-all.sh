#!/usr/bin/env bash
# Stop every module previously started by start-all.sh (reverse order).
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

echo "================================================================================"
echo "BASTION RAG: stopping all modules"
echo "================================================================================"

# Reverse of ALL_MODULES so dependents go down before their dependencies.
for (( i=${#ALL_MODULES[@]}-1; i>=0; i-- )); do
  stop_module "${ALL_MODULES[$i]}"
done

echo "Done."
