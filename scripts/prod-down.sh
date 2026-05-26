#!/usr/bin/env bash
# Stop the production stack (keeps named volumes, so DB + uploads survive).
#
# To also wipe volumes (irreversible — drops the database!):
#   ./scripts/prod-down.sh -v

source "$(dirname "${BASH_SOURCE[0]}")/_lib.sh"
require_env_file

exec "${COMPOSE[@]}" down "$@"
