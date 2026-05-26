#!/usr/bin/env bash
# Build + start the production stack in detached mode.
#
# Forwards any extra args to `docker compose up`, so you can e.g.
# rebuild just the API:
#   ./scripts/prod-up.sh api

source "$(dirname "${BASH_SOURCE[0]}")/_lib.sh"
require_env_file

exec "${COMPOSE[@]}" up -d --build "$@"
