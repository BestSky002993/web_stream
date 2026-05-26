#!/usr/bin/env bash
# Tail logs from the production stack.
#
# Usage:
#   ./scripts/prod-logs.sh          # all services
#   ./scripts/prod-logs.sh api      # just one
#   ./scripts/prod-logs.sh api nginx

source "$(dirname "${BASH_SOURCE[0]}")/_lib.sh"
require_env_file

exec "${COMPOSE[@]}" logs -f "$@"
